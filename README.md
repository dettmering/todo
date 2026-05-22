# Todo

Schlanke, persönliche, webbasierte Todo-Anwendung nach den Vorgaben in [LASTENHEFT.md](LASTENHEFT.md).

- **Backend:** Go, stdlib `net/http`, pure-Go SQLite (`modernc.org/sqlite`)
- **Frontend:** Vanilla HTML/CSS/JS, an Todoist orientiert, in das Binary eingebettet
- **Status:** MVP-Kern (Tasks, Projekte, Listenansicht, REST-API, Wiederholungen, NL-Datums-Parser). Erweiterungspunkte für API-Auth und Webhooks sind angelegt.

## Starten

```bash
go run ./cmd/server
# oder mit eigenen Pfaden
go run ./cmd/server -addr :8080 -db ./todo.db
```

Dann http://localhost:8080 öffnen.

## Architektur

```
cmd/server/             main, embed der Web-Assets, http.Server
internal/domain/        Task, Project, LocalDate, Priority, Sichtbarkeitsregel
internal/storage/       Repository-Interfaces (TaskRepo, ProjectRepo)
internal/storage/sqlite SQLite-Implementierung + Migrations + Inbox-Seed
internal/service/       Geschäftslogik, Validierung, ID-Vergabe, Event-Emit
internal/events/        In-Process Pub/Sub (künftiger Webhook-Subscriber-Slot)
internal/recurrence/    RRULE-Subset (FREQ/INTERVAL/BYDAY/BYMONTHDAY/BYMONTH)
internal/nldate/        Deutscher Natural-Language-Datums-Parser
internal/httpapi/       REST-Handler nach Lastenheft 5.9
internal/web/           Eingebettete HTML/CSS/JS-Assets
```

Erweiterungspunkte:
- **Webhooks:** `bus.Subscribe(handler)` in `cmd/server/main.go`. Events sind in `internal/events/events.go` definiert.
- **API-Auth:** Middleware vor `httpapi.New(svc)` einhängen, Bearer-Token prüfen.

## REST-API

Die API ist unter dem Präfix `/api/` erreichbar, akzeptiert und liefert JSON
(`Content-Type: application/json; charset=utf-8`) und kennt **keine
Authentifizierung** — der Server ist für den persönlichen Einsatz hinter einem
vertrauenswürdigen Netz oder Reverse-Proxy gedacht.

Datums-Felder kommen als ISO-Strings (`YYYY-MM-DD`) in den Body. Beim **Schreiben**
darf zusätzlich eine deutsche natürlichsprachige Phrase stehen (`morgen`,
`Freitag`, `27.5.`, `nächste Woche`, …); siehe Abschnitt
[Datumseingabe](#datumseingabe).

### Konventionen

- IDs sind opaque Strings mit Präfix: `task_<16 hex>`, `project_<16 hex>`.
  Die Inbox hat die reservierte ID `inbox` und kann nicht gelöscht werden.
- Zeitstempel (`created_at`, `updated_at`, `completed_at`) sind RFC3339-UTC.
- `LocalDate`-Felder sind timezone-naiv (`YYYY-MM-DD`); leerer String =
  „nicht gesetzt".
- **Soft-Delete:** `DELETE /api/tasks/{id}` setzt `status = "deleted"` —
  die Zeile bleibt erhalten. Projekt-Delete archiviert nur (`archived = true`).
- **Priorität:** `1` = höchste, `4` = niedrigste. `0` oder fehlend ⇒ Service
  setzt `P4`.
- **Sichtbarkeit:** Listenabfragen filtern standardmäßig nach Lastenheft 5.5
  (`status = open` und `visible_from ≤ heute`). Mit `?scope=all` werden alle
  Tasks zurückgegeben, unabhängig von `visible_from`.

### Fehlerformat

Alle Fehler liefern `{"error": "<text>"}` als Body. Status-Codes:

| Code | Bedeutung |
| --- | --- |
| `400 Bad Request` | Validierungsfehler (z. B. leerer `title`, ungültige Priorität, kaputtes JSON) |
| `404 Not Found` | Ressource (`task_*` / `project_*`) existiert nicht |
| `422 Unprocessable Entity` | nur `POST /api/parse`: Eingabe konnte nicht interpretiert werden |
| `500 Internal Server Error` | unerwarteter Storage- oder Logikfehler |

---

### Endpunkt-Übersicht

| Methode | Pfad | Wirkung |
| --- | --- | --- |
| `GET` | `/api/tasks` | Tasks listen (gefiltert) |
| `POST` | `/api/tasks` | Task anlegen |
| `GET` | `/api/tasks/{id}` | Task lesen |
| `PATCH` | `/api/tasks/{id}` | Task teilaktualisieren |
| `POST` | `/api/tasks/{id}/complete` | Task abschließen (spawnt ggf. Folge-Instanz) |
| `POST` | `/api/tasks/{id}/reopen` | Abschluss zurücknehmen |
| `DELETE` | `/api/tasks/{id}` | Soft-Delete |
| `GET` | `/api/projects` | Projekte listen |
| `POST` | `/api/projects` | Projekt anlegen |
| `GET` | `/api/projects/{id}` | Projekt lesen |
| `PATCH` | `/api/projects/{id}` | Projekt aktualisieren (auch entarchivieren) |
| `DELETE` | `/api/projects/{id}` | Archivieren (Tasks bleiben erhalten) |
| `POST` | `/api/parse` | NL-Datums-/Wiederholungs-Eingabe parsen |

---

### Tasks

#### `GET /api/tasks`

Query-Parameter (alle optional):

| Parameter | Werte | Wirkung |
| --- | --- | --- |
| `project_id` | `inbox`, `project_…` | Nur Tasks dieses Projekts |
| `status` | `open`, `completed`, `deleted` | Status-Filter; ohne Angabe gilt mit Default-Scope „nur open" |
| `scope` | `all` | Hebt den Sichtbarkeits-Filter (Lastenheft 5.5) auf — auch zukünftige `visible_from`-Tasks sind dabei |
| `date` | `YYYY-MM-DD` | Nur Tasks, deren `visible_from` exakt dieser Tag ist (Shortcut für `from=…&to=…` mit demselben Wert) |
| `from` | `YYYY-MM-DD` | Untere Grenze für `visible_from` (inklusiv) |
| `to` | `YYYY-MM-DD` | Obere Grenze für `visible_from` (inklusiv) |
| `sort` | `priority`, `deadline`, `visible_from`, `created_at` | Sortierung; ohne Angabe gilt die Default-Sortierung aus 5.6 |

`date`, `from` und `to` filtern über die Spalte **`visible_from`** (die
geplante Anzeige-/Erscheinungs­datum­spalte) und sind mit `project_id`,
`status`, `scope` und `sort` frei kombinierbar.

- `date=X` ist exakt äquivalent zu `from=X&to=X`. Wird `date` zusätzlich zu
  `from`/`to` gesetzt, gewinnt `date`.
- Tasks **ohne** `visible_from` werden bei aktivem Datumsfilter **nicht**
  zurückgegeben — sie haben kein planbares Datum, das gematcht werden könnte.
- Für Zukunfts-Ranges willst du in der Regel `scope=all` ergänzen, sonst
  greift zusätzlich der Sichtbarkeits-Filter (`visible_from ≤ heute`) und
  schneidet die Zukunft weg.
- `to` muss `≥ from` sein, sonst 400.

```bash
# Alle Tasks für heute (inkl. Vergangenheit-Sichtbarkeit egal)
curl 'http://localhost:8080/api/tasks?date=2026-05-22&scope=all'

# Diese Woche, sortiert nach Priorität, nur Projekt "renovierung"
curl 'http://localhost:8080/api/tasks?from=2026-05-18&to=2026-05-24&scope=all&project_id=project_…&sort=priority'

# Bereits erledigte Tasks an einem bestimmten Tag
curl 'http://localhost:8080/api/tasks?date=2026-05-20&status=completed&scope=all'
```

Antwort: `200 OK`, JSON-Array aus Task-Objekten (siehe unten).

```bash
curl 'http://localhost:8080/api/tasks?project_id=inbox&sort=deadline'
curl 'http://localhost:8080/api/tasks?scope=all&status=completed'
```

#### `POST /api/tasks`

Body:

| Feld | Typ | Pflicht | Default | Bemerkung |
| --- | --- | --- | --- | --- |
| `title` | string | ✓ | — | wird getrimmt; leer ⇒ 400 |
| `description` | string | | `""` | Markdown wird im Frontend gerendert |
| `project_id` | string | | `"inbox"` | unbekannte ID ⇒ 400 |
| `priority` | int | | `4` | `1`–`4`; `0` wird auf `4` gesetzt |
| `visible_from` | string | | `""` | ISO-Datum **oder** deutsche NL-Phrase |
| `deadline` | string | | `""` | ISO-Datum **oder** deutsche NL-Phrase |
| `recurrence_rule` | string | | `""` | RRULE-Subset, siehe [Wiederholungen](#wiederholungen) |

Antwort: `201 Created`, das vollständige Task-Objekt.

```bash
curl -X POST http://localhost:8080/api/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Steuererklärung machen",
    "project_id": "inbox",
    "priority": 2,
    "visible_from": "morgen",
    "deadline": "27.5.",
    "recurrence_rule": "FREQ=YEARLY;BYMONTH=5;BYMONTHDAY=27"
  }'
```

#### `GET /api/tasks/{id}`

Antwort: `200 OK` und das Task-Objekt; `404` wenn nicht vorhanden.

#### `PATCH /api/tasks/{id}`

Partial-Update. Jedes Feld ist optional; weggelassene Felder bleiben
unverändert. **Ein explizit gesetzter Leerstring** (z. B. `"deadline": ""`)
löscht das Datum.

Akzeptierte Felder: `title`, `description`, `project_id`, `priority`,
`visible_from`, `deadline`, `recurrence_rule`. Selbe Validierungen wie bei
`POST`.

```bash
# Nur Titel umbenennen
curl -X PATCH http://localhost:8080/api/tasks/task_abc... \
  -H 'Content-Type: application/json' \
  -d '{"title": "Steuern fertig machen"}'

# Deadline entfernen
curl -X PATCH http://localhost:8080/api/tasks/task_abc... \
  -H 'Content-Type: application/json' \
  -d '{"deadline": ""}'
```

Ein geänderter `visible_from` löst zusätzlich ein `task.rescheduled`-Event aus,
ein geänderter `deadline` ein `task.deadline_changed`-Event.

#### `POST /api/tasks/{id}/complete`

Setzt `status = "completed"` und `completed_at = now`. Idempotent (zweiter
Aufruf liefert den Task unverändert).

**Wenn der Task eine `recurrence_rule` hat**, wird automatisch eine neue offene
Folge-Instanz angelegt. Sie erbt Titel, Beschreibung, Projekt, Priorität,
Recurrence und die Deadline-Spanne, ihr `visible_from` ist die nächste
Regel-Belegung nach dem alten `visible_from` (auf heute hochgerollt, wenn die
alte Instanz überfällig war). Die abgeschlossene Instanz bleibt erhalten; die
neue erscheint in einer separaten Antwort/Event-Sequenz.

Antwort: `200 OK`, das **abgeschlossene** Task-Objekt. (Die neu erzeugte
Folge-Instanz wird nicht mitgeliefert — sie taucht beim nächsten `GET
/api/tasks` auf.)

#### `POST /api/tasks/{id}/reopen`

Setzt `status = "open"` und `completed_at = null` zurück. Idempotent.

#### `DELETE /api/tasks/{id}`

Soft-Delete: `status = "deleted"`. Antwort: `204 No Content`.

#### Task-Objekt

```json
{
  "id": "task_8f3e2c1d4a5b6789",
  "title": "Steuererklärung machen",
  "description": "Belege liegen im Ordner '2025'.",
  "project_id": "inbox",
  "status": "open",
  "priority": 2,
  "visible_from": "2026-05-23",
  "deadline": "2026-05-27",
  "recurrence_rule": "FREQ=YEARLY;BYMONTH=5;BYMONTHDAY=27",
  "recurrence_label": "jährlich am 27.05.",
  "created_at": "2026-05-22T18:21:04Z",
  "updated_at": "2026-05-22T18:21:04Z",
  "completed_at": null
}
```

- `recurrence_label` ist server-generiert (deutsche, menschenlesbare Form
  der `recurrence_rule`) und nur lesbar.
- `visible_from` und `deadline` sind weggelassen, wenn nicht gesetzt
  (`omitempty`).

---

### Projekte

#### `GET /api/projects`

Query: `include_archived=true` ⇒ archivierte Projekte werden mitgeliefert.
Sonst nur aktive. Antwort: `200 OK`, JSON-Array.

#### `POST /api/projects`

Body:

| Feld | Typ | Pflicht | Bemerkung |
| --- | --- | --- | --- |
| `name` | string | ✓ | wird getrimmt; leer ⇒ 400. Slug wird automatisch abgeleitet und ist über alle Projekte eindeutig |
| `description` | string | | |
| `sort_order` | int | | Manuelle Sortierung im Frontend |
| `color` | string | | Hex (`#1a2b3c` oder `#abc`); leer ⇒ Default |

#### `GET /api/projects/{id}`, `PATCH /api/projects/{id}`

`PATCH` akzeptiert `name`, `description`, `sort_order`, `color` und
zusätzlich `archived` (bool) — damit lassen sich archivierte Projekte auch
wieder reaktivieren.

#### `DELETE /api/projects/{id}`

Archiviert das Projekt (`archived = true`). Tasks bleiben unverändert dem
Projekt zugeordnet, damit historische Beschriftungen erhalten bleiben.
`DELETE /api/projects/inbox` ⇒ 400 (Inbox ist reserviert).

#### Projekt-Objekt

```json
{
  "id": "project_5c2a8e1f3b407d96",
  "name": "Renovierung",
  "slug": "renovierung",
  "description": "",
  "sort_order": 2,
  "archived": false,
  "color": "#db4c3f",
  "created_at": "2026-04-11T09:02:55Z",
  "updated_at": "2026-04-11T09:02:55Z"
}
```

---

### `POST /api/parse`

Spielt die Eingabezeile aus dem Quick-Add gegen den deutschen NL-Parser und
liefert das Ergebnis als strukturiertes JSON zurück — nützlich, um in eigenen
Clients eine Vorschau anzubieten, ohne die Task-Erstellung tatsächlich
auszulösen.

Request:

```json
{
  "text": "jeden Montag",
  "today": "2026-05-22"
}
```

`today` ist optional und überschreibt das vom Server angenommene „heute"
(praktisch für Tests).

Antwort `200 OK`:

```json
{
  "input": "jeden Montag",
  "visible_from": "2026-05-25",
  "recurrence_rule": "FREQ=WEEKLY;BYDAY=MO"
}
```

Antwort `422 Unprocessable Entity`, wenn nichts gematcht werden konnte:

```json
{"error": "cannot parse \"foobar\" as a date"}
```

---

### Datumseingabe

Beim Schreiben (`POST`/`PATCH` von `visible_from`, `deadline`) sind folgende
Eingaben äquivalent — der Server normalisiert intern auf ISO-Datum:

- **ISO:** `2026-05-27`
- **Punkt-Form:** `27.5.`, `27.05.`, `27.5.2027`
- **Relativ:** `heute`, `morgen`, `übermorgen`, `gestern`
- **Wochentage:** `montag`..`sonntag`, Kürzel `mo`/`di`/`mi`/`do`/`fr`/`sa`/`so`
- **Sprünge:** `nächste woche`, `nächsten monat`, `nächstes jahr`
- **`in N …`:** `in 3 tagen`, `in 2 wochen`, `in 1 monat`, `in 1 jahr`
- **`+N …`:** `+3`, `+3 tage`, `+3d`
- **Monatsanker:** `anfang juni`, `mitte juni`, `ende juni`, `monatsende`,
  `monatsanfang`
- **Kalenderwoche:** `kw 32`, `kw32` ⇒ Montag dieser ISO-Woche

### Wiederholungen

`recurrence_rule` ist eine pragmatische Teilmenge von RFC 5545 (RRULE):

```
FREQ=DAILY|WEEKLY|MONTHLY|YEARLY     (Pflicht)
INTERVAL=<positive int>              (Default 1)
BYDAY=MO,TU,WE,TH,FR,SA,SU           (nur WEEKLY)
BYMONTHDAY=1..31                     (MONTHLY/YEARLY)
BYMONTH=1..12                        (YEARLY)
```

Beispiele:

| Regel | Bedeutung |
| --- | --- |
| `FREQ=DAILY` | täglich |
| `FREQ=DAILY;INTERVAL=3` | alle 3 Tage |
| `FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR` | jeden Werktag |
| `FREQ=WEEKLY;INTERVAL=2;BYDAY=SA` | jeden zweiten Samstag |
| `FREQ=MONTHLY;BYMONTHDAY=1` | 1. jeden Monats (BYMONTHDAY=31 ⇒ kürzt auf Monatsende) |
| `FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1` | jährlich am 1. Januar |

Beim Senden ist auch die Kurzform aus dem NL-Parser zulässig — `POST /api/parse`
liefert die kanonische RRULE-Form zurück, die du dann in `recurrence_rule`
einsetzen kannst.

---

## Events (intern)

Der `events.Bus` veröffentlicht folgende Event-Typen. Subscribers werden in
goroutines aufgerufen und blockieren keine HTTP-Anfragen.

| Typ | Ausgelöst durch |
| --- | --- |
| `task.created` | `POST /api/tasks` und Recurrence-Spawn beim `complete` |
| `task.updated` | `PATCH /api/tasks/{id}`, `reopen` |
| `task.completed` | `POST /api/tasks/{id}/complete` |
| `task.deleted` | `DELETE /api/tasks/{id}` |
| `task.rescheduled` | `PATCH` mit geändertem `visible_from` |
| `task.deadline_changed` | `PATCH` mit geändertem `deadline` |
| `task.recurrence_generated` | nach `complete` auf einem wiederkehrenden Task |
| `project.created` | `POST /api/projects` |
| `project.updated` | `PATCH /api/projects/{id}` |
| `project.deleted` | `DELETE /api/projects/{id}` (Archivierung) |

## Noch nicht implementiert (bewusst aus dem MVP-Kern ausgenommen)

- Webhooks (Lastenheft 5.11) — Eventbus liegt bereit, Subscriber fehlt
- API-Authentifizierung (5.10) — Middleware-Slot frei
- Komplexere Filter & manuelle Sortierung (5.6/5.7)
