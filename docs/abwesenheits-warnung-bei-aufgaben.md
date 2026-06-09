# Warnung beim Anlegen einer Aufgabe auf eine Abwesenheit

## Context

Heute kann man problemlos eine Aufgabe anlegen, deren Datum mitten in eine
hinterlegte Abwesenheit (Urlaub, Feiertag, Krankheit, Dienstreise, …) fällt —
ohne jeden Hinweis. Erst in der „Demnächst"-Ansicht sieht man nachträglich das
Abwesenheits-Badge an der Kalenderwoche. Gewünscht ist eine **proaktive
Warnung schon während der Eingabe**.

Entscheidungen (mit dem User abgestimmt):
- **Warn-Stil:** Live-Hinweis im/unter dem Eingabefeld — nicht blockierend,
  fügt sich in die bestehende Quick-Add-Overlay-Logik ein.
- **Auslöser-Datum:** sowohl `visible_from` (Quick-Add `@datum` + Datum-Feld im
  Dialog) **als auch** `deadline` (Deadline-Feld im Bearbeiten-Dialog).
- **Typen:** **alle** Abwesenheitstypen inkl. Dienstreise lösen die Warnung aus.

Reines Frontend-Feature. Keine API-Änderung → kein README-Update nötig
(CLAUDE.md verlangt das nur bei Endpoint-Änderungen). Keine neue
Domain-/Service-Logik → keine Go-Tests nötig; die Überlappungsprüfung ist eine
triviale String-Vergleichslogik, die im Frontend bereits an einer Stelle
existiert (`app.js:1025`, `a.start_date <= … && a.end_date >= …`).

## Umsetzung

Alle Änderungen in `internal/web/static/` (über `go:embed` eingebettet — nur
neu kompilieren, kein Build-Step).

### 1. Abwesenheiten app-weit verfügbar machen — `app.js`

`state.absences` wird heute nur on-demand geladen (Abwesenheiten-/„Demnächst"-
Ansicht, `app.js:523`/`561`). Für eine Warnung in jeder Ansicht muss die Liste
beim Start vorliegen.

- In `boot()` (`app.js:1972`, nach `state.projects = await api.listProjects()`)
  ergänzen: `try { state.absences = await api.listAbsences(); } catch { /* best-effort */ }`.
- Die bestehenden Reloads nach Abwesenheits-CRUD (`app.js:678/729/743`) halten
  den State frisch — nichts weiter nötig. `listAbsences()` liefert via
  `?today=` die aktuellen + kommenden Abwesenheiten, also genau die für
  zukünftige Aufgaben-Daten relevanten.

### 2. Wiederverwendbare Helfer — `app.js`

Nahe den anderen Absence-Helfern (`ABSENCE_TYPE_LABEL`, `app.js:479`) einfügen:

- `absencesCoveringDate(iso)` → filtert `state.absences` auf
  `a.start_date <= iso && iso <= a.end_date` (gleiches Muster wie `app.js:1025`).
  Leerer/`null` iso → `[]`.
- `absenceWarningText(iso)` → nimmt die erste überlappende Abwesenheit und baut
  den Hinweistext, z.B. `⚠ fällt in Urlaub (Kreta, bis 19.6.)`. Nutzt
  `ABSENCE_TYPE_LABEL[a.type]`, `a.label` (optional) und `isoToDE(a.end_date)`.
  Keine Überlappung → `""`.

### 3. Quick-Add Live-Hinweis — `index.html` + `app.js`

- **index.html:** unter `.quick-add-input-wrap` (nach `app.js`-Zeilenpendant
  `index.html:72–84`) ein Element ergänzen:
  `<div class="qa-absence-hint hidden" id="qa-absence-hint"></div>`.
- **app.js – `processQuickAddInput()` (~`app.js:1238`):** nachdem `qaInline.date`
  aufgelöst ist, den Hinweis rendern: bei `qaInline.date?.iso` mit Überlappung
  Text aus `absenceWarningText` setzen + `hidden` entfernen, sonst leeren +
  `hidden` setzen. (Sauber als kleine `renderQuickAddAbsenceHint()`-Funktion,
  aufgerufen am Ende von `processQuickAddInput`.)
- **app.js – Submit/Reset (`app.js:1410–1413`):** nach `form.reset()` den
  Hinweis ebenfalls leeren/ausblenden.

### 4. Bearbeiten-Dialog: `visible_from` — `index.html` + `app.js`

- **index.html:** nach dem `td-visible-from`-Control (`index.html:209–211`) eine
  Hinweiszeile ergänzen, z.B.
  `<span class="absence-warn hidden" id="td-visible-from-absence"></span>`.
- **app.js – `wireTaskDialog()` (~`app.js:1622`):** Change/Input-Handler auf
  `#td-visible-from` (ISO-Wert) → `absenceWarningText(value)` in das Element
  schreiben. Zusätzlich nach den Quick-Buttons (`app.js:1719–1730`, inkl.
  `workday`/`nextWorkday`) denselben Render-Aufruf, da diese den Wert
  programmatisch setzen. Beim Öffnen des Dialogs
  (`openTaskDialogInternal`, `app.js:1559`) initial rendern bzw. clearen.

### 5. Bearbeiten-Dialog: `deadline` — `app.js`

Das Deadline-Feld nutzt bereits `wireNLDateInput("td-deadline")` (`app.js:296`),
das den geparsten Tag als `→ Datum`-Hint zeigt (`.date-hint[data-for=...]`).

- `wireNLDateInput` erweitern: nach erfolgreichem Parse (`res.visible_from`)
  zusätzlich `absenceWarningText(res.visible_from)` ans Hint-Element anhängen
  (z.B. als zweite Zeile / `<span class="absence-warn">`). Bei keiner
  Überlappung nichts anhängen. So profitieren beide nldate-Felder automatisch,
  ohne Sonderfall.
- `clearDateHint` (`app.js:325`) entsprechend mit-leeren.

### 6. Styling — `style.css`

- Klassen `.qa-absence-hint` und `.absence-warn`: kleiner Warntext in
  Danger-Farbe. Bestehende Tokens wiederverwenden (`--danger-bg`/`--p1` wie in
  `.dialog-error`, bzw. `.date-hint.bad`). Dezent, eine Zeile, mit ⚠-Glyph.
- Hidden-Verhalten über die vorhandene `.hidden`-Utility.

## Kritische Dateien

- `internal/web/static/app.js` — Helfer, Quick-Add-Hint, Dialog-Hooks, Boot-Load
- `internal/web/static/index.html` — zwei Hinweis-Elemente (Quick-Add + Dialog)
- `internal/web/static/style.css` — `.qa-absence-hint` / `.absence-warn`

## Verifikation

1. `go run ./cmd/server` starten, App öffnen.
2. Über die Abwesenheiten-Ansicht (`/absences`) eine Test-Abwesenheit anlegen,
   z.B. Urlaub „Kreta" 15.–19.6.2026.
3. **Quick-Add:** `Steuererklärung @15.6.` tippen → unter dem Feld erscheint
   `⚠ 15.6. fällt in Urlaub (Kreta, bis 19.6.)`. Datum auf `@22.6.` ändern →
   Hinweis verschwindet.
4. **Dialog `visible_from`:** beliebige Aufgabe öffnen, Datum auf den 16.6.
   setzen (auch via Quick-Button) → Warnung erscheint neben dem Datumsfeld.
5. **Dialog `deadline`:** im Deadline-Feld `17.6.` eingeben, Feld verlassen →
   `→`-Hint plus Abwesenheits-Warnung. Datum außerhalb der Abwesenheit → nur
   `→`-Hint.
6. `go build ./...` (Embed kompiliert), `go vet ./...`.
