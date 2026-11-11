package sqlite

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dettmering/todo/internal/domain"
)

// marker is a string that must never show up in an encrypted file.
const marker = "geheime-steuererklaerung-7f3a"

func testKey(b byte) []byte { return bytes.Repeat([]byte{b}, KeySize) }

func TestParseKey(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"valid 32 bytes", base64.StdEncoding.EncodeToString(testKey(1)), false},
		{"not base64", "not valid base64!!!", true},
		{"16 bytes", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 16)), true},
		{"33 bytes", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 33)), true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ParseKey(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && len(key) != KeySize {
				t.Fatalf("len(key) = %d", len(key))
			}
		})
	}
}

func TestInspectFile(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	tests := []struct {
		name string
		path string
		want fileState
	}{
		{"missing", filepath.Join(dir, "nope.db"), fileNew},
		{"empty", write("empty.db", nil), fileNew},
		{"plain header", write("plain.db", append([]byte("SQLite format 3\x00"), make([]byte, 100)...)), filePlain},
		{"noise", write("noise.db", bytes.Repeat([]byte{0xa5}, 4096)), fileEncrypted},
		{"short noise", write("short.db", []byte{1, 2, 3}), fileEncrypted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inspectFile(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDSNEscapesPath(t *testing.T) {
	got := dsn("/data/a?b#c.db", nil)
	if strings.ContainsAny(strings.TrimPrefix(got, "file:"), "?#") {
		t.Fatalf("path not escaped: %q", got)
	}
}

func openStore(t *testing.T, path string, key []byte) *Store {
	t.Helper()
	s, err := Open(path, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func addTask(t *testing.T, s *Store, id, title string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	err := s.Tasks().Create(context.Background(), &domain.Task{
		ID: id, Title: title, Description: title, ProjectID: domain.InboxID,
		Status: domain.StatusOpen, Priority: 4, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
}

func taskTitle(t *testing.T, s *Store, id string) string {
	t.Helper()
	task, err := s.Tasks().Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	return task.Title
}

// assertEncryptedOnDisk checks the database file and anything next to it: no
// plaintext header, no marker.
func assertEncryptedOnDisk(t *testing.T, path string) {
	t.Helper()
	files, _ := filepath.Glob(path + "*")
	if len(files) == 0 {
		t.Fatal("no database files found")
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(marker)) {
			t.Errorf("%s contains the plaintext marker", filepath.Base(f))
		}
		if bytes.HasPrefix(data, sqliteHeader) {
			t.Errorf("%s has a plaintext SQLite header", filepath.Base(f))
		}
	}
}

func TestEncryptedRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.db")
	s := openStore(t, path, testKey(1))
	addTask(t, s, "task_1", marker)

	// While open, the WAL holds the write: it must be encrypted as well.
	assertEncryptedOnDisk(t, path)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	assertEncryptedOnDisk(t, path)

	s = openStore(t, path, testKey(1))
	defer s.Close()
	if got := taskTitle(t, s, "task_1"); got != marker {
		t.Fatalf("title = %q", got)
	}
}

func TestPlaintextDatabaseIsConvertedOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.db")
	s := openStore(t, path, nil)
	addTask(t, s, "task_1", marker)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !bytes.Contains(data, []byte(marker)) {
		t.Fatal("precondition: plaintext file should contain the marker")
	}

	s = openStore(t, path, testKey(1))
	if got := taskTitle(t, s, "task_1"); got != marker {
		t.Fatalf("title after conversion = %q", got)
	}
	// Data written after the conversion lands in the same encrypted file.
	addTask(t, s, "task_2", marker+"-2")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	assertEncryptedOnDisk(t, path)
	if _, err := os.Stat(path + ".encrypting"); !os.IsNotExist(err) {
		t.Fatalf("temporary copy left behind: %v", err)
	}

	s = openStore(t, path, testKey(1))
	defer s.Close()
	if got := taskTitle(t, s, "task_2"); got != marker+"-2" {
		t.Fatalf("title = %q", got)
	}
}

func TestStaleTemporaryCopyIsReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.db")
	s := openStore(t, path, nil)
	addTask(t, s, "task_1", marker)
	s.Close()
	// A crash during an earlier conversion left garbage behind.
	if err := os.WriteFile(path+".encrypting", []byte("half written"), 0o600); err != nil {
		t.Fatal(err)
	}

	s = openStore(t, path, testKey(1))
	defer s.Close()
	if got := taskTitle(t, s, "task_1"); got != marker {
		t.Fatalf("title = %q", got)
	}
}

func TestOpenEncryptedFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.db")
	s := openStore(t, path, testKey(1))
	addTask(t, s, "task_1", marker)
	s.Close()

	tests := []struct {
		name    string
		key     []byte
		wantErr string
	}{
		{"without key", nil, "ENCRYPTION_KEY is not set"},
		{"with wrong key", testKey(2), "does not match"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Open(path, tt.key)
			if err == nil {
				s.Close()
				t.Fatal("Open succeeded")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}

	// The failed attempts must not have damaged anything.
	s = openStore(t, path, testKey(1))
	defer s.Close()
	if got := taskTitle(t, s, "task_1"); got != marker {
		t.Fatalf("title = %q", got)
	}
}
