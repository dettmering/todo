# Anonymisierte Abwesenheits-Webseite (Token-geschützt)

## Context

Der Nutzer teilt seine Abwesenheiten heute schon per iCal-Feed
(`/api/absences.ics`, token-geschützt) in Kalender-Clients. Er möchte
zusätzlich einen **teilbaren Weblink** haben, den er Kollegen gibt, damit
diese ein aktuelles Bild seiner Verfügbarkeit bekommen — **anonymisiert**,
d. h. ohne Grund (Urlaub/krank/…) und ohne Label wie „Kreta".

Entscheidungen (mit dem Nutzer geklärt):
- **Darstellung:** chronologische **Liste von Zeiträumen** (kein Kalendergitter).
- **Anonymisierung:** nur zwei Zustände — **„abwesend"** für alles außer
  Dienstreisen, **„unterwegs"** für Dienstreisen (`business_trip`, blockiert
  keine Arbeit, für Kollegen evtl. erreichbar). Grund und Label werden nie
  ausgeliefert.
- **Token:** eigenes `SHARE_TOKEN`, getrennt vom `ICAL_TOKEN`, damit der
  Weblink nicht den vollständigen ICS-Feed freischaltet und separat
  widerrufbar ist.

## Ansatz

Die Seite wird **serverseitig als vollständiges, self-contained HTML-Dokument
gerendert** — exakt analog zum ICS-Feed in `internal/httpapi/ical.go`. Vorteil:
Die sensiblen Felder (`label`, exakter `type`) verlassen den Server nie; die
Anonymisierung passiert vor dem Rendern, nicht im Browser. Kein JS-Fetch, kein
zweiter JSON-Endpoint, keine Token-in-JS-Problematik.

Neuer Endpoint: **`GET /api/absences.html?token=…`** — liegt bewusst unter
`/api/` (wird vom `httpapi`-Mux behandelt, nicht vom SPA-Catch-all in
`main.go`) und teilt sich die Reverse-Proxy-Ausnahme mit `/api/absences.ics`.

## Änderungen

### 1. Domain-Helper (getestet) — `internal/domain/absence.go`
- Neue reine Funktion `UpcomingAbsences(absences []*Absence, today LocalDate) []*Absence`:
  liefert nur Abwesenheiten mit `End >= today`, **sortiert nach `Start`
  aufsteigend**. (Vergangene werden verworfen — für eine Verfügbarkeitsseite
  irrelevant.) Analog zu den vorhandenen `AbsencesForView` / `AbsencesInRange`.
- Kategorisierung wiederverwendet die bestehende Methode `Absence.BlocksWork()`
  (Dienstreise → `false`): „unterwegs" = `!BlocksWork()`, sonst „abwesend".
  Kein neuer Typ nötig.

### 2. Neuer Handler — `internal/httpapi/share.go` (neu)
Aufbau gespiegelt von `ical.go`:
- `handleAbsencesShare(w, r)`:
  - `h.shareToken == ""` → `http.NotFound` (Route de facto deaktiviert, wie ICS).
  - Token per `crypto/subtle.ConstantTimeCompare` gegen `h.shareToken` prüfen,
    sonst `401`.
  - `h.svc.ListAbsences(ctx)` → `domain.UpcomingAbsences(all, domain.DateOf(time.Now()))`.
  - `Content-Type: text/html; charset=utf-8`, dann `buildAbsencesSharePage(...)` schreiben.
- `buildAbsencesSharePage(absences []*domain.Absence, now time.Time) string`:
  vollständiges HTML mit **inline-CSS** (kein externer Request, responsive,
  hell/dunkel via `prefers-color-scheme`). Pro Abwesenheit eine Zeile:
  deutscher Datumsbereich (`DD.MM.YYYY`, Einzeltag wenn `Start==End`) + Badge
  („abwesend" / „unterwegs"). Leerer Zustand: „Keine geplanten Abwesenheiten".
  Fußzeile mit „Stand: <Datum>". Alle dynamischen Strings HTML-escapen
  (`html.EscapeString`) — auch wenn hier keine User-Freitexte durchgereicht
  werden, defensiv.
- Kleine Datums-Helfer (deutsches Format) lokal; `icalDate`/`icalDateTime`
  bleiben ICS-spezifisch und werden nicht vermischt.

### 3. Verdrahtung — `internal/httpapi/server.go`
- `Handler`-Struct um `shareToken string` erweitern.
- `New(svc *service.Service, icalToken, shareToken string)` — zusätzlicher
  Parameter. **Alle Aufrufer anpassen** (main.go + evtl. Tests).
- Route registrieren: `h.mux.HandleFunc("GET /api/absences.html", h.handleAbsencesShare)`.

### 4. Verdrahtung — `cmd/server/main.go`
- `shareToken := os.Getenv("SHARE_TOKEN")`, an `httpapi.New(...)` durchreichen.
- Log-Zeile analog zum ICS-Feed: bei gesetztem Token
  `slog.Info("absence share page enabled", "path", "/api/absences.html")`.

### 5. Doku — `README.md`
- Endpoint-Übersichtstabelle: Zeile `GET /api/absences.html` ergänzen.
- Neuer Abschnitt analog zu `GET /api/absences.ics`: Token via `SHARE_TOKEN`,
  Anonymisierung („abwesend"/„unterwegs", keine Labels/Gründe), nur
  gegenwärtige + kommende Zeiträume.
- Reverse-Proxy-Hinweis (aktuell Zeile ~408): jetzt **beide** Pfade
  (`/api/absences.ics` **und** `/api/absences.html`) von der Auth ausnehmen.
- Env-Var-Abschnitt (bei `ICAL_TOKEN`/`MCP_TOKEN`, ~Zeile 580): `SHARE_TOKEN`
  dokumentieren (leer ⇒ Route deaktiviert, 404).

### 6. Tests
- `internal/domain/absence_test.go`: Tabellentests für `UpcomingAbsences`
  (vergangene raus, Grenzfall `End==today` bleibt, Sortierung nach `Start`).
- `internal/httpapi/share_test.go` (neu, gespiegelt von `ical_test.go`):
  - `shareToken==""` → 404; falsches/fehlendes Token → 401; korrektes Token → 200.
  - `business_trip` erscheint als „unterwegs", andere Typen als „abwesend".
  - **Label taucht im Output nicht auf** (Anonymisierung), z. B. Absence mit
    Label „Kreta" prüfen, dass „Kreta" nicht im HTML steht.
  - Vergangene Abwesenheit erscheint nicht.

## Verifikation

```bash
go build ./...
go vet ./...
go test ./...            # neue Domain- + Handler-Tests grün

# manuell:
SHARE_TOKEN=geheim ICAL_TOKEN=x go run ./cmd/server
curl -s 'http://localhost:8080/api/absences.html?token=geheim' | head -40
curl -s -o /dev/null -w '%{http_code}\n' 'http://localhost:8080/api/absences.html'          # 401
curl -s -o /dev/null -w '%{http_code}\n' 'http://localhost:8080/api/absences.html?token=x'  # 401
# ohne SHARE_TOKEN gestartet → 404
```
Im Browser `http://localhost:8080/api/absences.html?token=geheim` öffnen:
Liste der Zeiträume mit Badges, hell/dunkel, keine Grund-/Label-Angaben.
