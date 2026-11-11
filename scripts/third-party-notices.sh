#!/bin/sh
# Regenerates THIRD_PARTY_NOTICES.md: the license texts of everything that ends
# up in the binary. Go modules come from `go list -deps`, so test-only and
# tooling dependencies stay out. Run it after changing go.mod or replacing an
# embedded frontend library.
set -eu
cd "$(dirname "$0")/.."

out=THIRD_PARTY_NOTICES.md

# section <title> <file> — one license text as a fenced block. The fence is
# four backticks because some license files contain Markdown themselves.
section() {
	printf '\n### %s\n\n````text\n' "$1"
	cat "$2"
	printf '\n````\n'
}

{
	cat <<'HEADER'
# Third-Party Notices

Diese Datei wird von `scripts/third-party-notices.sh` erzeugt. Nicht von Hand
bearbeiten.

Das Projekt selbst steht unter der MIT-Lizenz (siehe `LICENSE`). Das Binary und
das Docker-Image enthalten zusätzlich die folgenden Fremdkomponenten unter ihren
jeweiligen Lizenzen.

Die Schriften Inter und JetBrains Mono werden nicht mitgeliefert: Der Browser
lädt sie zur Laufzeit von Google Fonts.

## Eingebettetes Frontend

| Komponente | Version | Lizenz | Fundstelle |
| --- | --- | --- | --- |
| [marked](https://github.com/markedjs/marked) | 12.0.2 | MIT | `internal/web/static/marked.min.js` |
| [DOMPurify](https://github.com/cure53/DOMPurify) | 3.4.17 | Apache-2.0 oder MPL-2.0, hier genutzt unter Apache-2.0 | `internal/web/static/purify.min.js` |
| [Feather](https://github.com/feathericons/feather) | | MIT | SVG-Pfade in `internal/web/static/app.js` (`ICON`) |
| [Lucide](https://github.com/lucide-icons/lucide) | | ISC | SVG-Pfade in `internal/web/static/app.js` (`ICON`) |
HEADER

	section marked third_party/licenses/marked.txt
	section DOMPurify third_party/licenses/dompurify.txt
	section Feather third_party/licenses/feather.txt
	section Lucide third_party/licenses/lucide.txt

	printf '\n## Go\n'
	section "Go-Standardbibliothek" "$(go env GOROOT)/LICENSE"

	printf '\n## Go-Module\n'
	go list -deps -f '{{with .Module}}{{.Path}} {{.Version}} {{.Dir}}{{end}}' ./cmd/server |
		grep -v '^github.com/dettmering/todo ' | sort -u |
		while read -r path version dir; do
			section "$path $version" "$dir/LICENSE"
			# Extra license files of a module (modernc.org/libc, .../memory).
			# The logo's license covers no code that is linked in.
			for extra in "$dir"/LICENSE-*; do
				[ -f "$extra" ] || continue
				case "$extra" in *LICENSE-LOGO) continue ;; esac
				section "$path $version (${extra##*/})" "$extra"
			done
		done
} >"$out"
