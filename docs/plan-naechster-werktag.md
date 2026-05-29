# Konzept: „Verschieben auf nächsten Werktag"-Button

## Context

In den Quick-Reschedule-Buttons des Task-Modals (`Heute`, `Morgen`, `Wochenende`, `Montag`) fehlt eine Option, die ein Datum auf den nächsten **echten Werktag** verschiebt — also unter Berücksichtigung von Wochenenden, gesetzlichen Feiertagen (Hessen) und persönlichen Urlaubstagen. Ziel ist ein neuer Button im Task-Modal, hinter dem eine korrekte Werktagslogik steht, sowie ein Settings-Bereich zur Pflege der Urlaubszeiträume. Feiertage werden deterministisch im Code berechnet (keine externe API).

## Architektur-Übersicht

Single Source of Truth liegt im Backend: das Frontend ruft `GET /api/workday/next?from=YYYY-MM-DD` auf und patcht das Ergebnis als `visible_from` in den Task. Damit bleibt die Werktagslogik konsistent unabhängig von Aufrufer (Modal, später Bulk-Operationen, evtl. Recurrence).

```
internal/workday          Reine Werktagslogik (Feiertage + NextWorkday)
internal/storage           VacationRepo (Interface)
internal/storage/sqlite    VacationRepo Impl + Migration #8
internal/service           VacationService + NextWorkday-Resolver
internal/httpapi           Routen /api/vacations, /api/workday/next
internal/web/static        Quick-Button + Settings-UI
```

## 1. Feiertagsberechnung — `internal/workday/workday.go` (NEU)

Reines Go-Modul, keine Abhängigkeiten außer `internal/domain`.

- **Osterformel**: Meeus/Butcher (Gaussianischer Osteralgorithmus) → liefert Ostersonntag für gegebenes Jahr.
- **`HolidaysHessen(year int) map[domain.LocalDate]string`** — gibt die 10 hessischen Feiertage zurück:
  - Neujahr 1.1.
  - Karfreitag (Ostern − 2)
  - Ostermontag (Ostern + 1)
  - Tag der Arbeit 1.5.
  - Christi Himmelfahrt (Ostern + 39)
  - Pfingstmontag (Ostern + 50)
  - Fronleichnam (Ostern + 60)
  - Tag der Deutschen Einheit 3.10.
  - 1. Weihnachtstag 25.12.
  - 2. Weihnachtstag 26.12.
- **`IsHoliday(d domain.LocalDate) bool`** — Lookup mit jahrweisem Cache (sync.Map).
- **`IsWeekend(d domain.LocalDate) bool`** — Sa/So.
- **`NextWorkday(from domain.LocalDate, vacationCheck func(domain.LocalDate) bool) domain.LocalDate`** — startet bei `from+1`, iteriert vorwärts und überspringt Wochenende/Feiertag/Urlaub. Cap auf z.B. 365 Iterationen als Sicherheitsnetz.

`domain.LocalDate` hat aktuell **kein** `AddDays()` — kleine Helper-Methode dazu (oder im `workday`-Paket lokal halten über Konversion nach `time.Time`).

Hessen ist fest verdrahtet (kein Settings-Toggle). Falls später Bundesland-Wahl gewünscht: Funktion zu `Holidays(year int, region string)` erweitern.

## 2. Urlaubsdaten — Speicherung

### Schema (Migration #8 in `internal/storage/sqlite/sqlite.go`)

```sql
CREATE TABLE vacations (
  id         TEXT PRIMARY KEY,
  start_date TEXT NOT NULL,   -- YYYY-MM-DD inklusiv
  end_date   TEXT NOT NULL,   -- YYYY-MM-DD inklusiv
  label      TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX idx_vacations_range ON vacations(start_date, end_date);
```

Folge dem bestehenden `PRAGMA user_version`-Pattern: neuen Step **anhängen**, niemals umordnen.

### Repo

- `internal/storage/vacations.go` — Interface `VacationRepo` mit `List(ctx) ([]domain.Vacation, error)`, `Create/Update/Delete/GetByID`.
- `internal/storage/sqlite/vacations.go` — Implementierung analog zu `projects.go`.
- `internal/domain/domain.go` erweitert um `type Vacation struct { ID string; StartDate, EndDate LocalDate; Label string; CreatedAt, UpdatedAt time.Time }`.

### Service

- `internal/service/service.go`: neuer `VacationService` (oder Methoden an `Service`) — IDs als `vac_*` mintet (wie `task_*`/`project_*`), Timestamps, Validierung (`start_date ≤ end_date`).
- `IsVacation(d domain.LocalDate) bool` als Service-Methode (lädt einmalig + cached in-memory, invalidiert bei Mutation). Nutzt der `NextWorkday`-Resolver als Closure.

## 3. HTTP-API

- `GET /api/vacations` — Liste
- `POST /api/vacations` — `{ start_date, end_date, label? }`
- `PATCH /api/vacations/{id}` — partielles Update
- `DELETE /api/vacations/{id}`
- `GET /api/workday/next?from=2026-05-22` → `{ "date": "2026-05-26", "reason": "next workday after weekend + Fronleichnam" }` (reason optional/debug)

Dateien:
- `internal/httpapi/vacations.go` (NEU) — analog zu `projects.go`
- `internal/httpapi/workday.go` (NEU) — schmaler Handler ruft `workday.NextWorkday` mit `service.IsVacation` als Closure auf
- `internal/httpapi/json.go` — `VacationDTO`
- `internal/httpapi/server.go` — Routen registrieren (Go 1.22 Pattern-Routing, siehe bestehende Muster)

`cmd/server/main.go` verdrahtet den neuen Repo + Service.

## 4. Frontend

### Quick-Button im Task-Modal

`internal/web/static/index.html` — neuen Button in `.td-quick-buttons` Container (nach Vorbild der bestehenden `.td-quick`-Buttons bei index.html:134-151):

```html
<button class="td-quick" data-quick="next-workday" type="button">
  <svg>...</svg><span>Nächster Werktag</span>
</button>
```

`app.js:1078-1102` (`quickRescheduleISO`) erweitern: für `kind === "next-workday"` async fetch `/api/workday/next?from=<heute oder aktuelles visible_from>` und ISO aus Response zurückgeben. Den Loop in `app.js:1189-1198` muss async-fähig gemacht werden (er ruft `quickRescheduleISO` heute synchron).

Auswahl der Basis: bevorzugt aktuelles `td-visible-from` des Tasks, sonst `today`. Damit funktioniert wiederholtes Klicken („übernächster Werktag") natürlich.

### Settings-UI für Urlaubsdaten

Neuer Eintrag in der Sidebar („Einstellungen") oder kleines Modal, das über ein Icon (z.B. Zahnrad in Header) öffnet. Inhalt:
- Liste aller Urlaube: Start–Ende, Label, Löschen-Knopf
- Eingabezeile / Button „Urlaub hinzufügen" → kleines Inline-Form mit 2 Date-Inputs + Label-Feld
- Speichert via `POST/PATCH/DELETE /api/vacations`
- Nach jeder Mutation: Client invalidiert nichts weiter — der nächste Workday-Lookup geht eh über den Server.

CSS: bestehende `.td-quick`-Token (style.css:835-851) und Project-CRUD-Form-Styles wiederverwenden — kein neuer Designsprachenbedarf. Siehe `STYLEGUIDE.md` für Token-Mapping.

## 5. Edge-Cases & Entscheidungen

- **Wenn `from` selbst kein Werktag ist**: `NextWorkday` startet bei `from+1` (Button heißt „verschieben auf nächsten Werktag" → impliziert Bewegung in die Zukunft, nicht „korrigiere zum aktuellen Werktag"). Sollte das anders gewünscht sein, ist es ein 2-Zeilen-Switch.
- **Urlaub vergangener Zeit**: bleibt in DB, aber zukünftige `NextWorkday`-Calls überspringen ihn ohnehin nicht (Datum ist Vergangenheit). Keine Cleanup-Logik nötig.
- **Recurrence**: Bestehende Wiederkehr-Logik (`internal/service/service.go:223 ff.`, `spawnNextInstance`) wird **nicht** angefasst. Der Button schiebt nur die aktuelle Instanz; künftige Instanzen kommen weiter aus dem Recurrence-Rule. Bewusste Entscheidung — Feature-Creep vermeiden.

## 6. Verifikation

1. **Unit-Tests für `internal/workday`** (neu — bisher keine Tests im Repo):
   - `TestEaster_2024_2030` — bekannte Termine (31.3.2024, 20.4.2025, 5.4.2026)
   - `TestHessenHolidays_2026` — alle 10 Daten matchen
   - `TestNextWorkday_SkipsWeekend` — Freitag → Montag
   - `TestNextWorkday_SkipsHolidayPlusWeekend` — z.B. Mittwoch vor Fronleichnam → Freitag (FL ist Do)
   - `TestNextWorkday_SkipsVacation` — Mocked vacationCheck
2. **End-to-End manuell**:
   - `go run ./cmd/server` → Browser öffnen
   - Task öffnen → „Nächster Werktag" klicken bei `visible_from = heute (Fr)` → erwartet Mo
   - Settings → Urlaub 25.–29.5.2026 eintragen → Task mit `visible_from = 22.5.` (Fr) → Button → erwartet 1.6. (Mo nach Urlaub, weil 29.5. ist Fr aber bis 29. Urlaub, dann Sa/So)
   - Bestehende Buttons („Heute"/„Morgen") weiter funktionieren
3. **API-Smoke**:
   - `curl -s 'http://localhost:8080/api/workday/next?from=2026-05-22'` → erwartet `2026-05-26` (Fr→Mo, Fronleichnam ist 4.6.)
   - `curl -X POST .../api/vacations -d '{"start_date":"2026-07-01","end_date":"2026-07-14","label":"Sommer"}'` → 201
4. **Migration**: erste Server-Start mit existierender `todo.db` → `PRAGMA user_version` von 7 auf 8, `vacations`-Tabelle existiert, keine vorhandenen Daten betroffen.

## Aufwand

Backend ca. 250 Zeilen Go (Migration, Repo, Service, Handler, Workday-Modul). Frontend ca. 150 Zeilen JS + 50 HTML/CSS. Tests ca. 80 Zeilen.
