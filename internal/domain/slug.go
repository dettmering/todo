package domain

import "strings"

// Slugify converts a human-readable name into a URL-safe identifier.
// German umlauts are transliterated; all other non-[a-z0-9] runs collapse to a
// single hyphen. Leading/trailing hyphens are trimmed. Returns "projekt" if
// the result would otherwise be empty.
func Slugify(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	prevHyphen := false
	emit := func(r rune) {
		b.WriteRune(r)
		prevHyphen = false
	}
	emitHyphen := func() {
		if b.Len() == 0 || prevHyphen {
			return
		}
		b.WriteByte('-')
		prevHyphen = true
	}
	for _, r := range strings.ToLower(name) {
		switch r {
		case 'ä':
			emit('a')
			emit('e')
		case 'ö':
			emit('o')
			emit('e')
		case 'ü':
			emit('u')
			emit('e')
		case 'ß':
			emit('s')
			emit('s')
		default:
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				emit(r)
			} else {
				emitHyphen()
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "projekt"
	}
	return out
}
