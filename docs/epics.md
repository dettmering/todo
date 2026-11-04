# Epics — langlaufende Initiativen

## Context

Die App kannte zwei Ebenen: Projekt und Aufgabe. Was fehlte, war die Ebene
dazwischen — ein Vorhaben, das über Monate läuft („Küche renovieren",
„ISO-Audit 2027"), Aufgaben bündelt und einen eigenen Anfang, ein eigenes Ende
und einen eigenen Abschluss hat. Ohne sie zerfällt so etwas in zwanzig lose
Aufgaben in einem Projekt, und niemand sieht, wie weit es ist.

Umgesetzt als neue Kernentität **Epic** (Lastenheft 4.11 / LA-F-014 in 5.15),
Gitea-Issue #15, Branch `feature-epics`.

### Entscheidungen (mit dem Nutzer abgestimmt)

* **Ein Task wandert mit seinem Epic.** `epic_id` setzen ⇒ `project_id` wird auf
  das Projekt des Epics gesetzt. Die Alternative — beides frei koppelbar — macht
  die geforderte Gruppierung in der Projektansicht unlesbar.
* **Eigene Ansicht `/epics/<slug>`** statt eines Sprungs in die Projektansicht:
  Zeitraum, Status und Fortschritt brauchen einen Ort.
* **Beendete Epics verlassen die Sidebar.** Sie bleiben über URL und API
  erreichbar. Eine Liste, die nur wächst, ertränkt die laufenden Einträge.
* **Archivieren statt löschen**, wie bei Projekten: Tasks behalten ihre
  `epic_id`, damit alte Beschriftungen weiter auflösen.

### Daraus abgeleitet (nicht separat gefragt, aber notwendig)

* Eine Aufgabe gehört höchstens **einem** Epic — sonst hat „darunter gruppiert"
  keine eindeutige Lesart.
* Ein **beendetes oder archiviertes** Epic nimmt keine Aufgaben an (400).
* Wird eine Aufgabe eines beendeten Epics wieder geöffnet, geht das **Epic
  zurück auf `active`**. Das Wiederöffnen zu verbieten machte ein beendetes Epic
  zur Falle.
* Wird nur `project_id` einer verknüpften Aufgabe geändert, **löst sich die
  Epic-Verknüpfung und die Aufgabe wird verschoben**. Das zuletzt genannte Feld
  gewinnt; liegen beide im selben Request, gewinnt `epic_id`.
* Wechselt ein **Epic** das Projekt, wandern **alle seine Aufgaben** mit.

## Umsetzung

Die Abwesenheiten-Entität war die Blaupause: dieselbe Dateiliste, dieselbe
Schichtenfolge. `internal/service` bleibt die einzige Stelle, die IDs vergibt,
Zeitstempel setzt, validiert und Events publiziert.

## Änderungen

### 1. Domäne — `internal/domain/epic.go`

`EpicStatus` (`active`/`completed`) mit `Valid()`, das `Epic`-Struct und die
reinen Helfer, in denen die testbare Fachlogik liegt: `ValidEpicRange`,
`Epic.Covers`, `Epic.Assignable`, `CountEpicTasks`, `EpicCanComplete`,
`OpenTaskCount`, `ActiveEpics`. `Task` bekommt `EpicID string` (leer = keins),
`AuditAction` die Aktion `epic_changed`.

### 2. Storage — `internal/storage/storage.go`

`EpicFilter{ProjectID, Status, IncludeArchived}` und `EpicRepo` (Create/Get/
Update/List — **kein Delete**, Epics werden über `Update` archiviert).
`TaskFilter` bekommt `EpicID`.

### 3. SQLite — `internal/storage/sqlite/{sqlite.go, epics.go, tasks.go}`

Migrationen **18–22** angehängt: Tabelle `epics` (FK auf `projects`), eindeutiger
Slug-Index, Projekt-Index, `tasks.epic_id` plus Index. `tasks.epic_id` bekommt
bewusst **keinen** Foreign Key — der leere String steht für „kein Epic" und
verletzte jede REFERENCES-Constraint.

### 4. Service — `internal/service/epic.go` + `service.go`

CRUD, `CompleteEpic` (prüft `domain.EpicCanComplete`), `ReopenEpic`,
`DeleteEpic` (archivierend, nach dem Vorbild von `DeleteProject`),
`EpicTaskCounts`, `ensureUniqueEpicSlug`. In `service.go`: `resolveEpic` als
Torwächter, Epic-Kopplung in `CreateTask`/`UpdateTask`, impliziter Epic-Reopen in
`ReopenTask`, Vererbung der `epic_id` an die Folgeinstanz einer Wiederholung.
`service.New` bekommt einen sechsten Repo-Parameter.

### 5. Events — `internal/events/events.go`

`epic.created`, `epic.updated`, `epic.completed`, `epic.deleted`.

### 6. HTTP-API — `internal/httpapi/{epics.go, json.go, server.go, tasks.go}`

Sieben Routen (`GET`/`POST /api/epics`, `GET`/`PATCH`/`DELETE /api/epics/{id}`,
`POST …/complete`, `POST …/reopen`). `PATCH` nimmt **kein** `status` — der läuft
nur über `complete`/`reopen`, damit die Beendigungsregel eine Stelle hat.
`epicDTO` mit berechnetem `open_tasks`/`done_tasks`; `taskDTO` bekommt `epic_id`
und `epic_title`, `GET /api/tasks` den Filter `epic_id`.

### 7. MCP — `internal/mcp/mcp.go`

`list_epics`, `get_epic`, `create_epic`, `update_epic`, `complete_epic`,
`reopen_epic`, `delete_epic`; `epic_id` an den Task-Tools.

### 8. Frontend — `internal/web/static/{index.html, app.js, style.css}`

Sidebar-Abschnitt „Epics" (nur laufende, Punkt in der Projektfarbe), Router für
`/epics/<slug>`, Epic-Kopf über dem Quick-Add, Gruppierung der Projektansicht
unter den Epics (flach, wenn ein Projekt keine hat), Epic-Dialog, Epic-Feld im
Aufgaben-Modal, Epic-Marke auf der Aufgabenzeile, Quick-Add legt in der
Epic-Ansicht direkt ins Epic. Kein neues Farbtoken.

### 9. Doku

`README.md` (Endpunkttabelle, Abschnitt „Epics", Epic-Objekt, Task-Felder,
Events, MCP-Tools), `LASTENHEFT.md` (4.11, 5.15/LA-F-014, Ansichtstabelle,
API-Pfade, Event-Liste, Datenmodell, Funktionsumfang, Abnahmekriterien 31–35),
`STYLEGUIDE.md` (7.11, Typo-Ausnahme in 3.1, Icon-Inventar), `CLAUDE.md`.

### 10. Tests

`internal/domain/epic_test.go` (Statuswerte, Zeitraum, Zuweisbarkeit, Zählung,
Beendbarkeit, Filterung/Sortierung) und `internal/service/epic_test.go`
(CRUD, Validierung, Slug-Eindeutigkeit, Beendigungsregel, impliziter Reopen,
Projektkopplung in beide Richtungen, Archivierung, Filter, Wiederholungen).

## Verifikation

```bash
go build ./... && go vet ./... && go test ./...

# Migration gegen eine Kopie der echten DB
cp todo.db /tmp/epics-check.db
go run ./cmd/server -db /tmp/epics-check.db -addr :8099

# Rundlauf
curl -XPOST localhost:8099/api/epics -H 'Content-Type: application/json' \
  -d '{"title":"Küche renovieren","project_id":"<id>","start_date":"2026-09-01"}'
curl -XPOST localhost:8099/api/tasks -H 'Content-Type: application/json' \
  -d '{"title":"Fliesen","epic_id":"<epic>"}'   # project_id wandert mit
curl -XPOST localhost:8099/api/epics/<epic>/complete   # 400, solange offen
curl -XPOST localhost:8099/api/tasks/<task>/complete
curl -XPOST localhost:8099/api/epics/<epic>/complete   # 200
curl -XPOST localhost:8099/api/tasks/<task>/reopen     # Epic wird wieder aktiv
curl -XDELETE localhost:8099/api/epics/<epic>          # 204, archiviert
```

Im Browser unter `http://localhost:8099/`: Sidebar-Abschnitt „Epics", Anlegen
über `+`, Klick öffnet `/epics/<slug>`, Doppelklick den Dialog, Projektansicht
gruppiert unter den Epics, „Beenden" scheitert sichtbar bei offenen Aufgaben.
Hell- und Dunkelmodus sowie die Breakpoints 1024 px und 720 px gegenprüfen.
