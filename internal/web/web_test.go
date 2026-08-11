package web

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// The server sends "script-src 'self'" without 'unsafe-inline'
// (httpapi.SecurityHeaders). That policy is only tenable as long as the
// frontend keeps its scripts in separate files: an inline <script> block or an
// on*="…" attribute would be silently dropped by the browser, and the failure
// would show up as a dead button rather than as an error anyone can grep for.
//
// These patterns are deliberately crude — they only need to catch the shapes
// this codebase would plausibly grow, not to parse HTML.
var (
	inlineScript  = regexp.MustCompile(`<script(?:\s[^>]*)?>\s*[^<\s]`)
	inlineHandler = regexp.MustCompile(`\son[a-z]+\s*=\s*["']`)
)

func TestStaticAssetsHaveNoInlineScript(t *testing.T) {
	assets := Static()
	err := fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".html") && !strings.HasSuffix(path, ".js") {
			return nil
		}
		data, err := fs.ReadFile(assets, path)
		if err != nil {
			return err
		}
		src := string(data)
		if m := inlineScript.FindString(src); m != "" {
			t.Errorf("%s: inline <script> found (%q) — blocked by script-src 'self'", path, m)
		}
		if m := inlineHandler.FindString(src); m != "" {
			t.Errorf("%s: inline event handler found (%q) — blocked by script-src 'self'", path, m)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk static assets: %v", err)
	}
}
