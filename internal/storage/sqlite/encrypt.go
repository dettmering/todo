package sqlite

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
)

// Encryption at rest uses the "adiantum" VFS of ncruces/go-sqlite3: every
// 4 KiB block of the database file, its WAL and any temp file is encrypted
// below SQLite, so the schema, indexes and free pages are covered too and the
// repos never see ciphertext. The threat model is "someone has a copy of the
// file" (backup, snapshot, disk). It is disk encryption: deterministic, no
// MACs — an attacker holding two copies can tell which blocks changed, and a
// forged block is not detected.

// KeySize is the length of the encryption key in bytes.
const KeySize = 32

// ParseKey decodes a base64-encoded 32-byte key, the format of ENCRYPTION_KEY
// (`openssl rand -base64 32`).
func ParseKey(b64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	if len(key) != KeySize {
		return nil, fmt.Errorf("encryption key must be %d bytes (got %d)", KeySize, len(key))
	}
	return key, nil
}

type fileState int

const (
	fileNew fileState = iota // missing or empty: SQLite creates it in whatever mode we open it
	filePlain
	fileEncrypted
)

// sqliteHeader starts every plaintext SQLite file. An encrypted file starts
// with ciphertext instead, which is how the two are told apart.
var sqliteHeader = []byte("SQLite format 3\x00")

func inspectFile(path string) (fileState, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return fileNew, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()
	head := make([]byte, len(sqliteHeader))
	n, err := io.ReadFull(f, head)
	switch {
	case n == 0 && (err == io.EOF):
		return fileNew, nil
	case err != nil && !errors.Is(err, io.ErrUnexpectedEOF):
		return 0, err
	case bytes.Equal(head[:n], sqliteHeader):
		return filePlain, nil
	default:
		return fileEncrypted, nil
	}
}

// dsn builds the URI for path. Escaping the path keeps a '?' or '#' in a file
// name from being read as the start of the query.
func dsn(path string, params url.Values) string {
	u := url.URL{Scheme: "file", Opaque: (&url.URL{Path: path}).EscapedPath(), RawQuery: params.Encode()}
	return u.String()
}

// openDB opens the pool. The key is set per connection with PRAGMA hexkey —
// not as a URI parameter, where it would be readable through the VFS API — and
// it must come before anything that touches the file, so all pragmas live in
// the connect hook instead of _pragma parameters.
func openDB(path string, key []byte) (*sql.DB, error) {
	params := url.Values{}
	if key != nil {
		params.Set("vfs", "adiantum")
	}
	hexKey := hex.EncodeToString(key)
	return driver.Open(dsn(path, params), func(c *sqlite3.Conn) error {
		if key != nil {
			// hexKey is [0-9a-f] only, so concatenating it is safe.
			if err := c.Exec(`PRAGMA hexkey = '` + hexKey + `'`); err != nil {
				return err
			}
		}
		return c.Exec(`PRAGMA busy_timeout = 5000;
			PRAGMA foreign_keys = ON;
			PRAGMA temp_store = MEMORY;
			PRAGMA journal_mode = WAL;`)
	})
}

// checkReadable reads the schema once. With the wrong key every page decrypts
// to noise and SQLite reports "not a database"; that turns into a message that
// names the actual cause.
func checkReadable(db *sql.DB, key []byte) error {
	var n int
	err := db.QueryRow(`SELECT count(*) FROM sqlite_schema`).Scan(&n)
	if errors.Is(err, sqlite3.NOTADB) {
		if key != nil {
			return errors.New("ENCRYPTION_KEY does not match the database")
		}
		return errors.New("file is not a SQLite database")
	}
	return err
}

// encryptInPlace converts a plaintext database into an encrypted one at the
// same path. VACUUM INTO writes a fresh, compacted copy through the adiantum
// VFS, so no free page of the old file is carried over. The copy is checked
// before it replaces the original; a crash at any point leaves the original
// untouched and the next start simply tries again.
func encryptInPlace(path string, key []byte) error {
	tmp := path + ".encrypting"
	if err := removeDBFiles(tmp); err != nil {
		return err
	}

	src, err := openDB(path, nil)
	if err != nil {
		return err
	}
	defer src.Close()
	want, err := tableCounts(src)
	if err != nil {
		return err
	}
	target := dsn(tmp, url.Values{"vfs": {"adiantum"}, "hexkey": {hex.EncodeToString(key)}})
	if _, err := src.Exec(`VACUUM INTO ?`, target); err != nil {
		return fmt.Errorf("vacuum into encrypted copy: %w", err)
	}
	if err := src.Close(); err != nil {
		return err
	}

	if err := verifyCopy(tmp, key, want); err != nil {
		_ = removeDBFiles(tmp)
		return err
	}
	// The source is closed, so its WAL has been checkpointed; a leftover -wal
	// would otherwise be read through the adiantum VFS as if it were encrypted.
	if err := removeSidecars(path); err != nil {
		return err
	}
	// Keep the original file's permissions; VACUUM INTO creates with defaults.
	if fi, err := os.Stat(path); err == nil {
		if err := os.Chmod(tmp, fi.Mode().Perm()); err != nil {
			return err
		}
	}
	return os.Rename(tmp, path)
}

func verifyCopy(path string, key []byte, want map[string]int) error {
	db, err := openDB(path, key)
	if err != nil {
		return err
	}
	defer db.Close()
	var check string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&check); err != nil {
		return fmt.Errorf("integrity check of encrypted copy: %w", err)
	}
	if check != "ok" {
		return fmt.Errorf("integrity check of encrypted copy: %s", check)
	}
	got, err := tableCounts(db)
	if err != nil {
		return err
	}
	for table, n := range want {
		if got[table] != n {
			return fmt.Errorf("encrypted copy has %d rows in %s, want %d", got[table], table, n)
		}
	}
	if err := db.Close(); err != nil {
		return err
	}
	return removeSidecars(path)
}

func tableCounts(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(tables))
	for _, t := range tables {
		var n int
		// Table names come from sqlite_schema, not from input; quoted anyway.
		if err := db.QueryRow(`SELECT count(*) FROM "` + t + `"`).Scan(&n); err != nil {
			return nil, err
		}
		out[t] = n
	}
	return out, nil
}

func removeSidecars(path string) error {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if err := os.Remove(path + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func removeDBFiles(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return removeSidecars(path)
}
