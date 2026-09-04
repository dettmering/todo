# Todo

Schlanke, persönliche, webbasierte Todo-Anwendung nach den Vorgaben in [LASTENHEFT.md](LASTENHEFT.md).

- **Backend:** Go, stdlib `net/http`, pure-Go SQLite (`modernc.org/sqlite`)
- **Frontend:** Vanilla HTML/CSS/JS, an gaengigen Task-Managern orientiert, in das Binary eingebettet
- **Umfang:** Tasks, Projekte, Epics (langlaufende Initiativen), Kommentare, Aktivitätsprotokoll, Abwesenheiten mit Arbeitstag-Logik, Wiederholungen, deutscher NL-Datums-Parser, REST-API, Webhooks, MCP-Server, iCal-Feed und anonymisierte Share-Seite.

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
internal/domain/        Task, Project, Epic, LocalDate, Priority, Sichtbarkeitsregel
internal/storage/       Repository-Interfaces (TaskRepo, ProjectRepo, EpicRepo)
internal/storage/sqlite SQLite-Implementierung + Migrations + Inbox-Seed
internal/service/       Geschäftslogik, Validierung, ID-Vergabe, Event-Emit
internal/events/        In-Process Pub/Sub
internal/webhook/       Event-Subscriber, der jedes Event an WEBHOOK_URL postet
internal/recurrence/    RRULE-Subset (FREQ/INTERVAL/BYDAY/BYMONTHDAY/BYMONTH)
internal/nldate/        Deutscher Natural-Language-Datums-Parser
internal/httpapi/       REST-Handler nach Lastenheft 5.9
internal/mcp/           MCP-Server (Streamable HTTP) auf derselben Service-Schicht
internal/web/           Eingebettete HTML/CSS/JS-Assets
```

Weitere Event-Subscriber werden über `bus.Subscribe(handler)` in
`cmd/server/main.go` angehängt; die Event-Typen stehen in
`internal/events/events.go`.

### Umgebungsvariablen

| Variable | Wirkung |
| --- | --- |
| `ICAL_TOKEN` | Aktiviert `GET /api/absences.ics` (leer ⇒ Route `404`) |
| `SHARE_TOKEN` | Aktiviert `GET /api/absences.html` (leer ⇒ Route `404`) |
| `SHARE_NAME` | Name in der Statuszeile der Share-Seite **und** im Kalendernamen des iCal-Feeds |
| `MCP_TOKEN` | Aktiviert den MCP-Endpunkt `/mcp` (leer ⇒ Route `404`) |
| `WEBHOOK_URL` | Aktiviert den Webhook-Versand (siehe [Webhooks](#webhooks)) |

Flags: `-addr` (Default `:8080`), `-db` (Default `./todo.db`).

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

- IDs sind opaque Strings mit Präfix: `task_<16 hex>`, `project_<16 hex>`,
  `epic_<16 hex>`, `absence_<16 hex>`, `comment_<16 hex>`, `audit_<16 hex>`.
  Die Inbox hat die reservierte ID `inbox` und kann nicht gelöscht werden.
- Zeitstempel (`created_at`, `updated_at`, `completed_at`) sind RFC3339-UTC.
- `LocalDate`-Felder sind timezone-naiv (`YYYY-MM-DD`); leerer String =
  „nicht gesetzt".
- **Soft-Delete:** `DELETE /api/tasks/{id}` setzt `status = "deleted"` —
  die Zeile bleibt erhalten. Projekt- und Epic-Delete archivieren nur
  (`archived = true`).
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
| `404 Not Found` | Ressource (`task_*` / `project_*` / `epic_*` / `absence_*`) existiert nicht |
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
| `GET` | `/api/tasks/{id}/comments` | Kommentare eines Tasks listen |
| `POST` | `/api/tasks/{id}/comments` | Kommentar an einen Task anhängen |
| `GET` | `/api/tasks/{id}/audit` | Audit-Log eines Tasks |
| `GET` | `/api/audit` | Audit-Log über alle Tasks (gefiltert) |
| `GET` | `/api/projects` | Projekte listen |
| `POST` | `/api/projects` | Projekt anlegen |
| `GET` | `/api/projects/{id}` | Projekt lesen |
| `PATCH` | `/api/projects/{id}` | Projekt aktualisieren (auch entarchivieren) |
| `DELETE` | `/api/projects/{id}` | Archivieren (Tasks bleiben erhalten) |
| `GET` | `/api/epics` | Epics listen (Filter: Projekt, Status, archivierte) |
| `POST` | `/api/epics` | Epic anlegen |
| `GET` | `/api/epics/{id}` | Epic lesen |
| `PATCH` | `/api/epics/{id}` | Epic teilaktualisieren (ohne Status) |
| `POST` | `/api/epics/{id}/complete` | Epic beenden (nur ohne offene Tasks) |
| `POST` | `/api/epics/{id}/reopen` | Epic wieder öffnen |
| `DELETE` | `/api/epics/{id}` | Archivieren (Tasks behalten ihre `epic_id`) |
| `GET` | `/api/absences` | Abwesenheiten listen (View- oder Range-Modus) |
| `GET` | `/api/absences.ics` | Abwesenheiten als abonnierbarer iCal-Feed (Token nötig) |
| `GET` | `/api/absences.html` | Anonymisierte Abwesenheits-Übersicht als Webseite (Token nötig) |
| `POST` | `/api/absences` | Abwesenheit anlegen |
| `GET` | `/api/absences/{id}` | Abwesenheit lesen |
| `PATCH` | `/api/absences/{id}` | Abwesenheit teilaktualisieren |
| `DELETE` | `/api/absences/{id}` | Abwesenheit löschen (hart) |
| `GET` | `/api/next-workday` | Nächsten Arbeitstag berechnen |
| `GET` | `/api/share-links` | Fertige Token-URLs für Share-Seite und iCal-Feed (für die interne UI) |
| `POST` | `/api/parse` | NL-Datums-/Wiederholungs-Eingabe parsen |

---

### Tasks

#### `GET /api/tasks`

Query-Parameter (alle optional):

| Parameter | Werte | Wirkung |
| --- | --- | --- |
| `project_id` | `inbox`, `project_…` | Nur Tasks dieses Projekts |
| `epic_id` | `epic_…` | Nur Tasks dieses Epics |
| `status` | `open`, `completed`, `deleted` | Status-Filter; ohne Angabe gilt mit Default-Scope „nur open" |
| `scope` | `all` | Hebt den Sichtbarkeits-Filter (Lastenheft 5.5) auf — auch zukünftige `visible_from`-Tasks sind dabei |
| `date` | `YYYY-MM-DD` | Nur Tasks, deren `visible_from` exakt dieser Tag ist (Shortcut für `from=…&to=…` mit demselben Wert) |
| `from` | `YYYY-MM-DD` | Untere Grenze für `visible_from` (inklusiv) |
| `to` | `YYYY-MM-DD` | Obere Grenze für `visible_from` (inklusiv) |
| `week` | `YYYY-Www` (z. B. `2026-W21`) | ISO-Kalenderwoche; entspricht `from=Montag&to=Sonntag` dieser KW |
| `completed_date` | `YYYY-MM-DD` | Nur Tasks, die an diesem UTC-Tag erledigt wurden |
| `completed_from` | `YYYY-MM-DD` | Untere Grenze für `completed_at` (inklusiv, UTC-Tag) |
| `completed_to` | `YYYY-MM-DD` | Obere Grenze für `completed_at` (inklusiv, UTC-Tag) |
| `completed_week` | `YYYY-Www` | ISO-Kalenderwoche auf `completed_at` |
| `today` | `YYYY-MM-DD` | Überschreibt das vom Server angenommene „heute" für den Sichtbarkeits-Filter (`visible_from ≤ heute`). Ohne Wirkung bei `scope=all` |
| `sort` | `priority`, `deadline`, `visible_from`, `created_at` | Sortierung; ohne Angabe gilt die Default-Sortierung aus 5.6 |

`date`, `from`, `to` und `week` filtern über die Spalte **`visible_from`**
(die geplante Anzeige-/Erscheinungs­datum­spalte). Die `completed_*`-Pendants
filtern über **`completed_at`** (UTC). Beide Achsen sind mit `project_id`,
`status`, `scope`, `sort` **und** miteinander frei kombinierbar.

- `date=X` ist exakt äquivalent zu `from=X&to=X` (analog für `completed_*`).
- `week`/`completed_week` darf **nicht** mit `date`/`from`/`to` derselben
  Achse kombiniert werden (400).
- Tasks ohne `completed_at` werden bei aktivem `completed_*`-Filter
  ausgeschlossen — typischerweise willst du also `status=completed` dazu.
- Achtung Zeitzone: `completed_at` wird in **UTC** gespeichert; ein Task,
  der lokal um 23:30 (CEST) erledigt wurde, fällt schon auf den UTC-Folgetag.
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
curl 'http://localhost:8080/api/tasks?week=2026-W21&scope=all&project_id=project_…&sort=priority'

# Bereits erledigte Tasks an einem bestimmten Tag (geplanter Tag)
curl 'http://localhost:8080/api/tasks?date=2026-05-20&status=completed&scope=all'

# Was wurde diese Woche erledigt?
curl 'http://localhost:8080/api/tasks?status=completed&completed_week=2026-W21&scope=all'
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
| `project_id` | string | | `"inbox"` | unbekannte ID ⇒ 400; wird von `epic_id` überstimmt |
| `epic_id` | string | | `""` | Epic, zu dem der Task gehört. Setzt `project_id` auf das Projekt des Epics. Unbekanntes, beendetes oder archiviertes Epic ⇒ 400 |
| `priority` | int | | `4` | `1`–`4`; `0` wird auf `4` gesetzt |
| `visible_from` | string | | `""` | ISO-Datum **oder** deutsche NL-Phrase |
| `deadline` | string | | `""` | ISO-Datum **oder** deutsche NL-Phrase |
| `recurrence_rule` | string | | `""` | RRULE-Subset, siehe [Wiederholungen](#wiederholungen) |
| `recurrence_mode` | string | | `"fixed"` | `fixed` oder `after_completion`, siehe [Wiederholungen](#wiederholungen); nur mit `recurrence_rule` wirksam |
| `recurrence_anchor` | string | | `visible_from` | Serientermin bei `fixed`; ISO-Datum oder deutsche NL-Phrase |

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

`epic_id` folgt derselben Regel wie beim Anlegen: ein gesetztes Epic zieht den
Task in dessen Projekt, ein leerer String löst die Verknüpfung. Wird umgekehrt
nur `project_id` geändert, während der Task an einem Epic hängt, **wird die
Verknüpfung gelöst und der Task verschoben** — das zuletzt genannte Feld
gewinnt. Liegen beide im selben Request, gewinnt `epic_id`.

Partial-Update. Jedes Feld ist optional; weggelassene Felder bleiben
unverändert. **Ein explizit gesetzter Leerstring** (z. B. `"deadline": ""`)
löscht das Datum.

Akzeptierte Felder: `title`, `description`, `project_id`, `priority`,
`visible_from`, `deadline`, `recurrence_rule`, `recurrence_mode`,
`recurrence_anchor`. Selbe Validierungen wie bei
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
Recurrence (Regel **und** Modus) und die Deadline-Spanne. Woher ihr
`visible_from` kommt, entscheidet `recurrence_mode`:

- **`fixed`** (Default): nächste Regel-Belegung nach dem **`recurrence_anchor`**
  (nicht nach `visible_from`!), auf heute hochgerollt, wenn die alte Instanz
  überfällig war. Verpasste Termine werden übersprungen, das Raster bleibt
  erhalten — ein Geburtstag am 15.08. liegt auch dann wieder auf dem 15.08.,
  wenn er am 17.08. abgehakt wird.
- **`after_completion`**: ein Regelschritt ab dem **Abhak-Tag**. Das Intervall
  startet also bei jedem Erledigen neu — „Kaffeemaschine alle 4 Wochen
  reinigen", am 17.08. abgehakt, erscheint wieder am 14.09.

Die abgeschlossene Instanz bleibt erhalten; die neue erscheint in einer
separaten Antwort/Event-Sequenz.

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
  "project_name": "Inbox",
  "epic_id": "epic_5c2a8e1f3b407d96",
  "epic_title": "Küche renovieren",
  "status": "open",
  "priority": 2,
  "visible_from": "2026-05-23",
  "deadline": "2026-05-27",
  "recurrence_rule": "FREQ=YEARLY;BYMONTH=5;BYMONTHDAY=27",
  "recurrence_mode": "fixed",
  "recurrence_anchor": "2026-05-27",
  "recurrence_label": "jährlich am 27.05.",
  "created_at": "2026-05-22T18:21:04Z",
  "updated_at": "2026-05-22T18:21:04Z",
  "completed_at": null,
  "comment_count": 1,
  "comments": [
    {
      "id": "comment_2b91c07de4a35f68",
      "task_id": "task_8f3e2c1d4a5b6789",
      "body": "Belege für 2025 sind vollständig.",
      "created_at": "2026-05-24T07:11:39Z"
    }
  ]
}
```

- `recurrence_label` ist server-generiert (deutsche, menschenlesbare Form
  der `recurrence_rule`) und nur lesbar. Bei `recurrence_mode
  = "after_completion"` endet es auf „ab Erledigung".
- `recurrence_mode` wird nur zusammen mit einer `recurrence_rule` gespeichert;
  ohne Regel ist es leer und wird weggelassen.
- `recurrence_anchor` ist der **Serientermin** und existiert nur bei
  `recurrence_mode = "fixed"`. Bei `after_completion` ist er immer leer, weil
  dort jede Erledigung neu ankert.
- `visible_from` und `deadline` sind weggelassen, wenn nicht gesetzt
  (`omitempty`).
- `comments` ist **immer** vorhanden (leeres Array, wenn es keine gibt) und
  auch in der Listenantwort von `GET /api/tasks` eingebettet — neueste
  zuerst. `comment_count` entfällt bei `0` (`omitempty`). Beide Felder sind
  nur lesbar; geschrieben wird über
  [`POST /api/tasks/{id}/comments`](#post-apitasksidcomments).

---

#### `GET /api/tasks/{id}/comments`

Kommentare des Tasks, **neueste zuerst**. Antwort: `200 OK`, JSON-Array aus
Kommentar-Objekten (leeres Array, wenn keine existieren).

#### `POST /api/tasks/{id}/comments`

Hängt eine Markdown-Notiz an den Task. Das ist auch der Weg, um Kommentare
programmatisch über die API zu setzen.

| Feld | Typ | Pflicht | Bemerkung |
| --- | --- | --- | --- |
| `body` | string | ✓ | Markdown-Quelltext; wird getrimmt, leer ⇒ 400. Wird unverändert gespeichert und erst im Frontend gerendert |

Antwort: `201 Created`, das Kommentar-Objekt. `404`, wenn der Task nicht
existiert. Kommentare sind **append-only** — es gibt weder `PATCH` noch
`DELETE`. Sie lassen sich an Tasks in **jedem** Status anhängen, also auch an
bereits abgeschlossene.

```bash
curl -X POST http://localhost:8080/api/tasks/task_8f3e2c1d4a5b6789/comments \
  -H 'Content-Type: application/json' \
  -d '{"body": "Belege für 2025 sind **vollständig**."}'
```

#### Kommentar-Objekt

```json
{
  "id": "comment_2b91c07de4a35f68",
  "task_id": "task_8f3e2c1d4a5b6789",
  "body": "Belege für 2025 sind **vollständig**.",
  "created_at": "2026-05-24T07:11:39Z"
}
```

---

### Audit-Log

Das Audit-Log protokolliert Task-Änderungen. Einträge werden von der
Service-Schicht geschrieben — bei `create`, `complete`, `reopen`, `delete`,
beim Recurrence-Spawn sowie bei `PATCH`-Änderungen an `visible_from`,
`deadline`, `project_id` und `priority`. Titel- und Beschreibungsänderungen
werden **nicht** protokolliert. Das Log ist append-only und wird nie
aufgeräumt.

#### `GET /api/audit`

Query-Parameter (alle optional):

| Parameter | Werte | Wirkung |
| --- | --- | --- |
| `task_id` | `task_…` | Nur Einträge zu diesem Task |
| `date` | `YYYY-MM-DD` | Nur Einträge dieses Kalendertags (Shortcut für `from=…&to=…`) |
| `from` | `YYYY-MM-DD` | Untere Grenze (inklusiv) |
| `to` | `YYYY-MM-DD` | Obere Grenze (inklusiv) |
| `week` | `YYYY-Www` | ISO-Kalenderwoche |

Die Datumsfilter werden als **UTC-Tage** interpretiert (gleiche Zeitzonen-Falle
wie bei `completed_*`). `week` darf nicht mit `date`/`from`/`to` kombiniert
werden (400). Antwort: `200 OK`, JSON-Array, **neueste zuerst**.

```bash
# Was ist diese Woche passiert?
curl 'http://localhost:8080/api/audit?week=2026-W21'

# Historie eines einzelnen Tasks
curl 'http://localhost:8080/api/audit?task_id=task_8f3e2c1d4a5b6789'
```

#### `GET /api/tasks/{id}/audit`

Kurzform für `/api/audit?task_id={id}`, identische Antwort. Ein unbekannter
Task liefert ein leeres Array, keinen `404` — das Log ist unabhängig vom
Task-Datensatz.

#### Audit-Objekt

```json
{
  "id": "audit_71c4de9a03b8f625",
  "task_id": "task_8f3e2c1d4a5b6789",
  "task_title": "Steuererklärung machen",
  "action": "deadline_changed",
  "old_value": "2026-05-27",
  "new_value": "2026-06-15",
  "at": "2026-05-24T09:40:12Z"
}
```

`task_title` ist ein **Snapshot** zum Zeitpunkt der Änderung — das Log bleibt
lesbar, auch wenn der Task später umbenannt wird. `old_value`/`new_value` sind
opake Strings, deren Bedeutung von `action` abhängt:

| `action` | `old_value` / `new_value` |
| --- | --- |
| `created`, `completed`, `reopened`, `deleted` | beide leer (entfallen per `omitempty`) |
| `visible_from_set`, `visible_from_cleared`, `visible_from_changed` | `YYYY-MM-DD` bzw. leer |
| `deadline_set`, `deadline_cleared`, `deadline_changed` | `YYYY-MM-DD` bzw. leer |
| `project_changed` | Projekt**name** (Snapshot), nicht die ID |
| `priority_changed` | `"P1"` … `"P4"` |

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

### Epics

Ein **Epic** ist eine langlaufende Initiative — die Ebene zwischen Projekt
(ein Fach, das bleibt) und Task (ein einzelner Schritt). „Küche renovieren"
oder „ISO-Audit 2027": etwas, das über Monate läuft, Aufgaben bündelt und
irgendwann beendet ist.

- Ein Epic gehört zu **genau einem Projekt**. Die **Inbox ist ausgeschlossen** —
  sie ist der Eingangskorb, kein Vorhaben (⇒ 400).
- `start_date` und `end_date` sind **beide optional**; eine Initiative darf ohne
  festen Anfang, ohne festes Ende oder ganz ohne Rahmen laufen.
- `status` ist `active` („laufend") oder `completed` („beendet") und wird
  **nicht** per `PATCH` gesetzt, sondern nur über `complete` / `reopen`.
- Ein Task gehört höchstens **einem** Epic (`epic_id`) und liegt dann **immer**
  im Projekt des Epics — siehe [`POST /api/tasks`](#post-apitasks). Wird er in
  ein anderes Projekt verschoben, löst sich die Verknüpfung.

#### `GET /api/epics`

Query-Parameter (alle optional):

| Parameter | Werte | Wirkung |
| --- | --- | --- |
| `project_id` | `project_…` | Nur Epics dieses Projekts |
| `status` | `active`, `completed` | Status-Filter; anderer Wert ⇒ 400 |
| `include_archived` | `true` | Archivierte Epics werden mitgeliefert |

Antwort: `200 OK`, JSON-Array, sortiert nach `sort_order`, dann `title`.

#### `POST /api/epics`

Body:

| Feld | Typ | Pflicht | Default | Bemerkung |
| --- | --- | --- | --- | --- |
| `title` | string | ✓ | — | wird getrimmt; leer ⇒ 400. Slug wird abgeleitet und ist über alle Epics eindeutig |
| `project_id` | string | ✓ | — | unbekannte ID oder `inbox` ⇒ 400 |
| `description` | string | | `""` | Markdown wird im Frontend gerendert |
| `start_date` | string | | `""` | ISO-Datum; leer = offener Anfang |
| `end_date` | string | | `""` | ISO-Datum; leer = offenes Ende. Vor `start_date` ⇒ 400 |
| `sort_order` | int | | `0` | Manuelle Sortierung im Frontend |

Datumsangaben werden hier — wie bei Abwesenheiten — **nur als ISO-Datum**
entgegengenommen; die NL-Eingabe aus 4.7 gilt für Aufgaben.

Antwort: `201 Created`, das vollständige Epic-Objekt.

```bash
curl -X POST http://localhost:8080/api/epics \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Küche renovieren",
    "project_id": "project_5c2a8e1f3b407d96",
    "start_date": "2026-09-01",
    "end_date": "2027-03-31"
  }'
```

#### `GET /api/epics/{id}`, `PATCH /api/epics/{id}`

`PATCH` akzeptiert `title`, `description`, `project_id`, `start_date`,
`end_date`, `sort_order` und `archived` (bool — damit lassen sich archivierte
Epics wieder reaktivieren). **Kein `status`**: der läuft über `complete` und
`reopen`, damit die Beendigungsregel genau eine Stelle hat.

Der Slug bleibt bei einer Umbenennung stehen, damit geteilte Links weiter
funktionieren. Ein Projektwechsel **zieht alle Tasks des Epics mit** — sonst
läge ein Task in einem anderen Projekt als sein Epic.

#### `POST /api/epics/{id}/complete`

Setzt `status = "completed"` und `completed_at`. **Schlägt fehl (400), solange
noch eine Aufgabe des Epics offen ist** — erledigte und soft-gelöschte Tasks
blockieren nicht. Ein bereits beendetes Epic ist ein No-op (`200`).

```bash
curl -X POST http://localhost:8080/api/epics/epic_5c2a8e1f3b407d96/complete
# {"error":"validation error: epic still has 3 open task(s)"}
```

#### `POST /api/epics/{id}/reopen`

Setzt `status = "active"` und löscht `completed_at`. Passiert außerdem
**implizit**, wenn eine Aufgabe eines beendeten Epics über
`POST /api/tasks/{id}/reopen` wieder geöffnet wird — ein beendetes Epic mit
offener Aufgabe darunter wäre eine Falle.

#### `DELETE /api/epics/{id}`

Archiviert das Epic (`archived = true`). Tasks behalten ihre `epic_id`, damit
historische Beschriftungen erhalten bleiben; ein archiviertes Epic nimmt aber
keine neuen Tasks mehr an (⇒ 400). Antwort `204`, mehrfaches Löschen ist ein
No-op.

#### Epic-Objekt

`open_tasks` und `done_tasks` sind berechnet, nicht gespeichert:
soft-gelöschte Tasks zählen in keiner der beiden Zahlen mit.

```json
{
  "id": "epic_5c2a8e1f3b407d96",
  "title": "Küche renovieren",
  "slug": "kueche-renovieren",
  "description": "Alles von Fliesen bis Armatur.",
  "project_id": "project_6b0f0e5b2891b2c0",
  "project_name": "Renovierung",
  "status": "active",
  "start_date": "2026-09-01",
  "end_date": "2027-03-31",
  "sort_order": 0,
  "archived": false,
  "open_tasks": 3,
  "done_tasks": 5,
  "created_at": "2026-09-04T09:57:13Z",
  "updated_at": "2026-09-04T09:57:13Z",
  "completed_at": null
}
```

---

### Abwesenheiten

Eine **Abwesenheit** ist ein zusammenhängender Zeitraum, in dem der Nutzer weg
ist (Urlaub, Dienstreise, krank, Feiertag, …). `start_date` und `end_date` sind
**inklusive** Kalendertage. Abwesenheiten fließen in die „Nächster Arbeitstag"-
Berechnung ein.

`type` ist eines von:

| Typ | Bedeutung | Blockiert Arbeit? |
| --- | --- | --- |
| `vacation` | Urlaub | ✓ |
| `holiday` | Feiertag | ✓ |
| `sick` | Krank | ✓ |
| `other` | Sonstiges (Default, wenn leer) | ✓ |
| `business_trip` | Dienstreise | ✗ — zählt als normaler Arbeitstag |

`business_trip` ist der Sonderfall: Es ist eine Abwesenheit (taucht in Listen
und Range-Abfragen auf), gilt aber bei `GET /api/next-workday` als regulärer
Arbeitstag und blockiert die Planung nicht.

#### `GET /api/absences`

Zwei Modi, abhängig von den Query-Parametern:

| Modus | Parameter | Wirkung |
| --- | --- | --- |
| **View** (Default) | `today` (optional, `YYYY-MM-DD`) | Aktuelle und kommende Abwesenheiten plus vergangene aus dem **laufenden Kalenderjahr**. `today` überschreibt das vom Server angenommene „heute". |
| **Range** | `from` und/oder `to` (`YYYY-MM-DD`) | Alle Abwesenheiten, die das inklusive Fenster `[from, to]` **überlappen** — auch laufende, die vor `from` begonnen haben und nach `to` enden. |

Sobald `from` **oder** `to` gesetzt ist, gilt der Range-Modus. Antwort:
`200 OK`, JSON-Array aus Abwesenheits-Objekten.

```bash
# View: was steht an + dieses Jahr schon gewesen
curl 'http://localhost:8080/api/absences?today=2026-06-08'

# Range: alle Abwesenheiten im Juni 2026 (auch über die Grenzen laufende)
curl 'http://localhost:8080/api/absences?from=2026-06-01&to=2026-06-30'
```

**„Bin ich heute abwesend?"** — `from` und `to` auf denselben Tag setzen:

```bash
curl 'http://localhost:8080/api/absences?from=2026-06-08&to=2026-06-08'
```

Leeres Array ⇒ nicht abwesend. Mindestens ein Eintrag ⇒ abwesend; über den
`type` (siehe Tabelle) erkennt der Client, ob es sich um eine echte (Arbeit
blockierende) Abwesenheit oder nur eine `business_trip` handelt.

#### `GET /api/absences.ics`

Liefert **alle** Abwesenheiten als iCalendar-Feed (`text/calendar`, RFC 5545)
zum Abonnieren in Google Calendar, iOS, Thunderbird usw. Jede Abwesenheit wird
ein ganztägiges `VEVENT`. Der Feed pollt keine Auth-Session — er wird über den
Query-Parameter `token` geschützt:

- Der Token muss exakt der Umgebungsvariable `ICAL_TOKEN` des Servers
  entsprechen (Vergleich in konstanter Zeit).
- **Ist `ICAL_TOKEN` nicht gesetzt, ist der Feed deaktiviert** und der Endpoint
  antwortet mit `404` — er kann also nie ungeschützt ausgeliefert werden.
- Falscher oder fehlender Token ⇒ `401`.

```bash
curl 'http://localhost:8080/api/absences.ics?token=DEINGEHEIMNIS'
```

Hinweise zur Erzeugung:

- Ganztägige Events nutzen `VALUE=DATE`; `DTEND` ist dabei **exklusiv**. Da
  `end_date` inklusiv ist, wird `DTEND = end_date + 1 Tag` gesetzt, damit der
  letzte Abwesenheitstag im Kalender belegt bleibt.
- `SUMMARY` ist deutsch nach Typ (`Urlaub`, `Feiertag`, `Dienstreise`, `Krank`,
  sonst `Abwesend`), optional mit `: <label>`.
- **`comment` wird zur Event-`DESCRIPTION`** — dort zeigen Kalender-Clients die
  Details (Buchungsnummer, Hotel, Flugzeiten), für die der Titel keinen Platz
  hat. Zeilenumbrüche werden als `\n` escaped. Ist der Kommentar leer (oder nur
  Whitespace), fehlt die Property ganz; eine leere `DESCRIPTION` lässt manche
  Clients einen leeren Notizblock rendern.
- `UID` ist stabil (`<id>@todo`), sodass erneutes Abrufen Events aktualisiert
  statt zu duplizieren.
- Jedes Event bekommt `STATUS:CONFIRMED` — Abwesenheiten sind Tatsachen, keine
  vorläufigen Einladungen. Die meisten Clients nehmen das ohnehin an, einige
  wollen es ausgeschrieben.
- **Der Kalendername folgt `SHARE_NAME`**: gesetzt ⇒ `<name>s Abwesenheiten`
  (z. B. `Tills Abwesenheiten`), sonst generisch `Abwesenheiten`. Dieselbe
  Variable personalisiert die Statuszeile der Share-Seite.
- **Frei/Belegt-Status hängt am Typ.** `TRANSP` ist das portable Feld,
  `X-MICROSOFT-CDO-BUSYSTATUS` das Outlook-Pendant (`OOF` = Out of Office,
  lila schraffiert, Teams-Status springt auf „nicht im Büro"):

  | Typ | `TRANSP` | `X-MICROSOFT-CDO-BUSYSTATUS` |
  | --- | --- | --- |
  | `vacation`, `holiday`, `sick`, `other` | `OPAQUE` | `OOF` |
  | `business_trip` | `TRANSPARENT` | `WORKINGELSEWHERE` |

  Nur die Dienstreise bleibt terminierbar — analog zu `Absence.BlocksWork()`,
  dem einzigen Typ, an dem trotzdem gearbeitet wird.
- Für Outlook wird zusätzlich `X-MICROSOFT-CDO-ALLDAYEVENT:TRUE` je Event
  gesetzt (sonst rendert Outlook `VALUE=DATE`-Events unzuverlässig und lässt
  einzelne aus), dazu `SEQUENCE` und `CREATED`/`LAST-MODIFIED`. Andere Clients
  (Google, Apple) ignorieren die `X-`-Felder.
- **Kalender-Metadaten stehen doppelt drin**, weil keine Schreibweise alle
  Clients erreicht — die vorstandardisierten `X-`-Felder und ihre Nachfolger aus
  RFC 7986:

  | Zweck | vorstandardisiert | RFC 7986 |
  | --- | --- | --- |
  | Kalendername | `X-WR-CALNAME` | `NAME` |
  | Beschreibung | — | `DESCRIPTION` |
  | Refresh-Intervall (`PT4H`) | `X-PUBLISHED-TTL` | `REFRESH-INTERVAL;VALUE=DURATION` |

  Outlook liest `X-PUBLISHED-TTL`, Apple Calendar und Thunderbird
  `REFRESH-INTERVAL`. Google ignoriert beide und pollt nach eigenem Zeitplan.
  Die vier Stunden orientieren sich am ungünstigsten Fall: Urlaub steht Wochen
  vorher fest, eine Krankmeldung entsteht am selben Tag und soll nicht bis zum
  Abend unsichtbar bleiben.

Hinter einem Reverse-Proxy mit Auth (z. B. Traefik BasicAuth) müssen die Pfade
`/api/absences.ics` **und** `/api/absences.html` von der Auth ausgenommen
werden, damit Kalender-Clients bzw. Kollegen sie ohne interaktiven Login
erreichen; der jeweilige `token` ist dann der einzige Schutz.

#### `GET /api/absences.html`

Liefert eine **anonymisierte** HTML-Übersicht der Abwesenheiten — ein Link zum
Teilen mit Kollegen, damit sie ein aktuelles Bild der Verfügbarkeit haben. Die
Seite wird serverseitig gerendert (self-contained HTML, keine externen
Requests), sodass die sensiblen Felder gar nicht erst im Browser landen:

- Die Seite ist durchgängig **englisch**. Ganz oben steht die **aktuelle
  Verfügbarkeit** als Satz — z. B. „Till is currently available / away / on a
  business trip", abgeleitet aus den Abwesenheiten, die **heute** greifen (eine
  arbeitsblockierende Abwesenheit ⇒ *away*, nur eine Dienstreise ⇒ *on a
  business trip*, sonst *available*). Der Name wird über `SHARE_NAME` gesetzt;
  ohne diese Variable heißt es „Currently available".
- Direkt darunter steht — **nur im Status *away*** — die Zeile **„Back &lt;in N
  days&gt; on &lt;Wochentag, Datum&gt;"**, z. B. „Back in 4 days on **Monday, 3
  August 2026**". Das ist der nächste Arbeitstag nach heute: Wochenenden und
  weitere blockierende Abwesenheiten werden übersprungen, Dienstreisetage zählen
  als Arbeitstage (gleiche Regel wie `GET /api/next-workday`). Bei *available*
  und *on a business trip* entfällt die Zeile — da ist nichts anzukündigen.
- Die **Distanz steht vorn**, weil sie die eigentlich gesuchte Information ist;
  das exakte Datum folgt, damit ein weitergereichter Screenshot weiterhin etwas
  Wahres sagt. Staffelung: 2–6 Tage ⇒ „in N days", 7–13 Tage ⇒ „in a week", ab
  14 Tagen ⇒ „in N weeks" (abgerundet). Bewusst vermieden: „next Monday" —
  mehrdeutig, und der Wochentag steht ohnehin daneben.
- **Ein Tag** bekommt einen eigenen Satzbau: „Back tomorrow, **8 July 2026**".
  „tomorrow" ist ein vollständiges Zeitadverbial und verträgt kein „on" hinter
  sich; der Wochentag entfällt hier, weil er neben „tomorrow" nichts trägt.
- Darunter — durch eine Trennlinie vom Kopfbereich abgesetzt — folgt die
  Zwischenüberschrift **„Scheduled absences in the next 12 weeks"**. Die
  Wochenzahl stammt aus `domain.ShareWindowWeeks`, benennt also dasselbe
  Fenster, das Zeitleiste und Liste zeigen.
- Direkt unter der Überschrift liegt die **Zeitleiste**: eine
  schmale waagerechte Schiene, auf der jede Abwesenheit als farbiger Balken
  sitzt (rot = *Away*, blau = *Business Trip*), dazu ein Punkt für **heute**,
  feine Striche als Wochengrenzen und Monatskürzel darunter. Sie beantwortet
  „wo sind die Lücken?" auf einen Blick — etwas, das eine Liste von Zeiträumen
  nicht leistet. Details:
  - Das Fenster beginnt am **Montag der laufenden Woche** (nicht bei „heute"),
    damit die Leiste nicht mitten in der Woche weiterkriecht, und umfasst
    12 × 7 Tage.
  - Balken laufender Abwesenheiten werden am linken Fensterrand
    **abgeschnitten**; was komplett außerhalb liegt, entfällt — in der Liste
    darunter ebenso, Leiste und Liste zeigen dieselben Einträge
    (`domain.ShareWindow`).
  - **Über** der Schiene stehen die **Kalenderwochen** (ISO 8601, nackte Zahl
    ohne „KW"), je Woche mittig über ihrer Zelle. Die laufende Woche — immer
    die erste Zelle — ist hervorgehoben.
  - Monatslabel: der Monat des Fensterstarts plus jeder Monatserste; Labels,
    die näher als ~9 % am vorherigen liegen oder am rechten Rand kleben,
    entfallen statt sich zu überlappen.
  - Die Leiste trägt **keine Tooltips** und ist `aria-hidden` — sie ist
    Dekoration, die Liste bleibt die lesbare Quelle. Auch sie ist
    **anonymisiert**: sie entsteht ausschließlich aus Datumsangaben.
  - Ohne Abwesenheiten wird gar keine Leiste gerendert.
- Die Liste darunter zeigt nur den **Datumsbereich** plus einen groben Status —
  **„Away"** für alles außer Dienstreisen, **„Business Trip"** für
  `business_trip` (blockiert keine Arbeit, i. d. R. erreichbar). **Grund,
  `label` und `comment` werden nie ausgeliefert** — der Kommentar trägt
  typischerweise Buchungsnummern und hat auf einer geteilten Seite nichts
  verloren.
- Der Eintrag, der **heute gerade läuft**, trägt zusätzlich den Marker **„now"**
  zwischen Zeitraum und Badge. Ohne ihn müsste der Leser jeden Zeitraum gegen
  das aktuelle Datum abgleichen, um zu sehen, welcher der laufende ist.
- Unterhalb der Liste steht eine **Legende**, die beide Badges erklärt („Away" =
  not working, not reachable; „Business Trip" = travelling for work, usually
  reachable). Sie erscheint nur, wenn es auch Einträge gibt — der Leerzustand
  bleibt legendenfrei.
- Nur **gegenwärtige und kommende** Zeiträume (`end_date >= heute`),
  chronologisch sortiert, und davon nur, was **innerhalb der 12 Wochen** der
  Leiste beginnt. Vergangene Abwesenheiten erscheinen nicht, weiter entfernte
  ebenso wenig — die Seite beantwortet „wann bist du in absehbarer Zeit weg",
  nicht die Jahresplanung.
- Die Fußzeile „Last updated" zeigt das **Alter der letzten
  Abwesenheits-Änderung** (jüngstes `updated_at` aller Abwesenheiten, gerechnet
  in der Server-Zeitzone `TZ`), nicht die Abrufzeit: „Last updated today /
  yesterday / 3 days ago / a week ago / 2 weeks ago / 2 months ago". Hier ist
  die Frage nicht „wann genau", sondern „ist das noch aktuell" — und „2 months
  ago" ist dafür ein sofortiges Urteil, ein Datum nicht. Das **exakte Datum
  hängt als `title`-Attribut** daran (Hover). Ohne Abwesenheiten entfällt die
  Zeile; bei einem Zeitstempel aus der Zukunft (Uhrenversatz) fällt sie auf die
  alte Form „Last updated: 7 July 2026" zurück.
- Schutz identisch zum iCal-Feed: `token`-Query-Parameter muss der
  Umgebungsvariable `SHARE_TOKEN` entsprechen (Vergleich in konstanter Zeit).
  **Ist `SHARE_TOKEN` nicht gesetzt, ist die Seite deaktiviert** (`404`);
  falscher oder fehlender Token ⇒ `401`.

```bash
SHARE_TOKEN=DEINGEHEIMNIS SHARE_NAME=Till go run ./cmd/server
curl 'http://localhost:8080/api/absences.html?token=DEINGEHEIMNIS'
```

`SHARE_TOKEN` ist bewusst getrennt von `ICAL_TOKEN`: Der geteilte Weblink schaltet
so nicht den vollständigen iCal-Feed (mit Labels, Kommentaren und Typen) frei und ist separat
widerrufbar.

#### `GET /api/share-links`

Liefert die **fertigen Token-URLs** der beiden öffentlichen Abwesenheits-Routen,
damit die interne UI sie in der Abwesenheiten-Ansicht anbieten kann, ohne die
Tokens im Frontend zu duplizieren — die leben weiterhin nur in der Umgebung.
„Teilen" öffnet die Share-Seite in einem neuen Tab, „iCal" kopiert die
absolute Feed-URL in die Zwischenablage (ein Kalender-Client will den Link,
keine heruntergeladene `.ics`-Datei).

```json
{
  "share_url": "/api/absences.html?token=DEINGEHEIMNIS",
  "ical_url": "/api/absences.ics?token=DEINKALENDERTOKEN"
}
```

| Feld | Bedingung |
| --- | --- |
| `share_url` | nur wenn `SHARE_TOKEN` gesetzt ist |
| `ical_url` | nur wenn `ICAL_TOKEN` gesetzt ist |

Ein Feld **fehlt komplett**, wenn der zugehörige Token nicht gesetzt ist (die
Route ist dann ohnehin `404`); ohne beide Tokens ist die Antwort `{}`. Die
Antwort trägt `Cache-Control: no-store`.

Achtung beim Reverse-Proxy: Anders als `/api/absences.ics` und
`/api/absences.html` gibt diese Route die Tokens *heraus* und muss deshalb
**hinter der Auth bleiben** — sie also nicht von der BasicAuth ausnehmen.

#### `POST /api/absences`

Body:

| Feld | Typ | Pflicht | Default | Bemerkung |
| --- | --- | --- | --- | --- |
| `type` | string | | `"other"` | einer der Werte oben; unbekannt ⇒ 400 |
| `label` | string | | `""` | freier Name, z. B. `"Kreta"` |
| `comment` | string | | `""` | freier Notiztext, z. B. Buchungsnummern; mehrzeilig erlaubt |
| `start_date` | string | ✓ | — | ISO-Datum `YYYY-MM-DD` |
| `end_date` | string | ✓ | — | ISO-Datum; muss `≥ start_date` sein, sonst 400 |

Anders als bei Tasks erwarten `start_date`/`end_date` hier **reines ISO-Datum**
— der NL-Parser greift bei Abwesenheiten nicht. Antwort: `201 Created`, das
vollständige Abwesenheits-Objekt.

```bash
curl -X POST http://localhost:8080/api/absences \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "vacation",
    "label": "Kreta",
    "comment": "Buchung AB-12345, Hotel Elounda",
    "start_date": "2026-07-10",
    "end_date": "2026-07-24"
  }'
```

#### `GET /api/absences/{id}`

Antwort: `200 OK` und das Abwesenheits-Objekt; `404` wenn nicht vorhanden.

#### `PATCH /api/absences/{id}`

Partial-Update; weggelassene Felder bleiben unverändert. Akzeptiert `type`,
`label`, `comment`, `start_date`, `end_date`. Selbe Validierungen wie bei `POST` (gültiger
Typ, `end_date ≥ start_date`).

```bash
# Urlaub verlängern
curl -X PATCH http://localhost:8080/api/absences/absence_abc... \
  -H 'Content-Type: application/json' \
  -d '{"end_date": "2026-07-28"}'
```

#### `DELETE /api/absences/{id}`

**Harter Delete** (anders als bei Tasks/Projekten): die Zeile wird entfernt.
Antwort: `204 No Content`.

#### Abwesenheits-Objekt

```json
{
  "id": "absence_7a1b2c3d4e5f6071",
  "type": "vacation",
  "label": "Kreta",
  "comment": "Buchung AB-12345, Hotel Elounda",
  "start_date": "2026-07-10",
  "end_date": "2026-07-24",
  "created_at": "2026-06-08T10:15:00Z",
  "updated_at": "2026-06-08T10:15:00Z"
}
```

#### `GET /api/next-workday`

Liefert den ersten Tag **strikt nach heute**, der weder Wochenende noch von
einer Arbeit blockierenden Abwesenheit abgedeckt ist (`business_trip` blockiert
nicht). Kein Body. Antwort:

```json
{"date": "2026-06-09"}
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
  "text": "nächste Woche",
  "today": "2026-05-22"
}
```

`today` ist optional und überschreibt das vom Server angenommene „heute"
(praktisch für Tests).

Antwort `200 OK`:

```json
{
  "input": "nächste Woche",
  "visible_from": "2026-05-25"
}
```

Der Parser deckt **Datumsphrasen** ab (siehe [Datumseingabe](#datumseingabe)).
Wiederholungen werden nicht aus freiem Text erkannt, sondern strukturiert als
RRULE gesetzt — im Frontend über das Auswahlfeld, über die API im Feld
`recurrence_rule` (siehe [Wiederholungen](#wiederholungen)).

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

#### Bezugspunkt: `recurrence_mode`

Die Regel sagt, *wie weit* der nächste Termin entfernt liegt; `recurrence_mode`
sagt, *wovon aus* gerechnet wird.

| Modus | Anker | Verwenden für |
| --- | --- | --- |
| `fixed` (Default) | geplantes `visible_from` | Termine mit festem Datum: Geburtstage, Miete, Quartalsabschluss |
| `after_completion` | Tag des Abhakens | Intervall-Aufgaben: Kaffeemaschine reinigen, Filter wechseln, Rasen mähen |

Beispiel — beide am **17.08.2026** abgehakt:

| `visible_from` | Regel | Modus | nächste Instanz |
| --- | --- | --- | --- |
| `2026-08-15` | `FREQ=YEARLY` | `fixed` | `2027-08-15` |
| `2026-07-20` | `FREQ=WEEKLY;INTERVAL=4` | `after_completion` | `2026-09-14` |

#### Serientermin vs. Startdatum

Bei `fixed` sind **zwei** Daten im Spiel, und die Trennung ist der Punkt:

| Feld | Bedeutung | Wer ändert es |
| --- | --- | --- |
| `visible_from` | wann diese *eine* Instanz auftaucht | Verschieben („auf Mittwoch"), Quick-Buttons, Drag |
| `recurrence_anchor` | wo die *Serie* im Kalender liegt | nur explizit |

Ein `PATCH` auf `visible_from` lässt `recurrence_anchor` unangetastet. Damit
bleibt „Geburtstag am 1.8., dieses Jahr auf Mittwoch den 19.8. verschoben, dort
abgehakt" eine Serie am **1.8.** — der nächste Termin ist der 01.08.2027, nicht
der 19.08.2027. Wer die Serie wirklich verlegen will, patcht
`recurrence_anchor` explizit.

Beim Anlegen wird `recurrence_anchor` aus `visible_from` übernommen, wenn es
nicht mitgeschickt wird. Die Deadline-Spanne wird ebenfalls gegen den
Serientermin gerechnet, nicht gegen ein verschobenes `visible_from`.

`after_completion` erlaubt **nur `FREQ` und `INTERVAL`**. Kalender-Anker
(`BYDAY`, `BYMONTHDAY`, `BYMONTH`) werden mit `400` abgelehnt: „jeden Montag,
gerechnet ab dem Abhaken" hat keine sinnvolle Lesart.

Bestandsdaten und Clients, die das Feld nicht senden, verhalten sich unverändert
wie `fixed`.

---

## MCP-Server

Neben der REST-API stellt der Server einen [Model-Context-Protocol](https://modelcontextprotocol.io)-Endpunkt
bereit, über den LLM-Clients (Claude Code, Claude Desktop) die Todo-App als
Werkzeuge nutzen können. Der MCP-Server ist **in dasselbe Binary eingebettet**
und ruft dieselbe Service-Schicht wie die REST-API auf — gleiche Validierung,
ID-Vergabe, Timestamps und Events.

- **Transport:** Streamable HTTP (offizielles Go-SDK
  `github.com/modelcontextprotocol/go-sdk`).
- **Pfad:** `/mcp`
- **Aktivierung:** nur wenn die Umgebungsvariable `MCP_TOKEN` gesetzt ist
  (leer ⇒ Route deaktiviert, 404 — analog zu `ICAL_TOKEN`).
- **Auth:** der Token wird auf zwei Wegen akzeptiert (jeweils `401` bei
  Fehlschlag) — als `Authorization: Bearer <MCP_TOKEN>`-Header **oder** als
  `?token=<MCP_TOKEN>`-Query-Parameter (wie der iCal-Feed). Hinter Traefik läuft
  `/mcp` über einen eigenen Router ohne BasicAuth (MCP-Clients können kein
  interaktives Login), der Token ist der einzige Schutz.

Client-Anbindung **Claude Code / Desktop** (Header):

```bash
claude mcp add --transport http todo https://todo.till.fi/mcp \
  --header "Authorization: Bearer <MCP_TOKEN>"
```

Client-Anbindung **claude.ai** (Custom Connector): dessen UI bietet nur OAuth an.
Den Token stattdessen in die URL hängen und das OAuth-Feld leer lassen — bei
gültigem Token challengt der Server nie, also wird kein OAuth-Flow ausgelöst:

```
https://todo.till.fi/mcp?token=<MCP_TOKEN>
```

### Tools

Datumsfelder akzeptieren wie die REST-API ISO-Datum oder deutsche NL-Phrasen
(siehe [Datumseingabe](#datumseingabe)). Tools geben ihr Ergebnis als
strukturiertes JSON (snake_case, wie die REST-Objekte) zurück.

| Tool | Beschreibung | Service |
| --- | --- | --- |
| `list_tasks` | Tasks auflisten (Filter: `project_id`, `epic_id`, `status`, `scope`, `sort`) | `ListTasks` |
| `get_task` | Einzelnen Task per `id` holen | `GetTask` |
| `create_task` | Task anlegen (nur `title` Pflicht; Default Inbox/P4) | `CreateTask` |
| `update_task` | Felder eines Tasks ändern (nur gesetzte Felder) | `UpdateTask` |
| `complete_task` | Task abschließen (spawnt ggf. nächste Instanz) | `CompleteTask` |
| `reopen_task` | Abgeschlossenen Task wieder öffnen | `ReopenTask` |
| `delete_task` | Task soft-löschen (`status = deleted`) | `DeleteTask` |
| `list_projects` | Projekte auflisten (`include_archived`) | `ListProjects` |
| `create_project` | Projekt anlegen (nur `name` Pflicht) | `CreateProject` |
| `list_epics` | Epics auflisten (Filter: `project_id`, `status`, `include_archived`) | `ListEpics` |
| `get_epic` | Einzelnes Epic per `id` holen (inkl. offener/erledigter Tasks) | `GetEpic` |
| `create_epic` | Epic anlegen (`title` und `project_id` Pflicht, Inbox ausgeschlossen) | `CreateEpic` |
| `update_epic` | Felder eines Epics ändern (nur gesetzte Felder) | `UpdateEpic` |
| `complete_epic` | Epic beenden — schlägt fehl, solange Tasks offen sind | `CompleteEpic` |
| `reopen_epic` | Beendetes Epic wieder öffnen | `ReopenEpic` |
| `delete_epic` | Epic archivieren (Tasks behalten ihre Verknüpfung) | `DeleteEpic` |
| `list_absences` | Abwesenheiten auflisten | `ListAbsences` |
| `create_absence` | Abwesenheit `start_date`..`end_date` anlegen (optional `label`, `comment`) | `CreateAbsence` |
| `get_absence` | Einzelne Abwesenheit per `id` holen | `GetAbsence` |
| `update_absence` | Felder einer Abwesenheit ändern (nur gesetzte Felder) | `UpdateAbsence` |
| `delete_absence` | Abwesenheit hart löschen | `DeleteAbsence` |
| `next_workday` | Nächsten Arbeitstag ermitteln | `NextWorkday` |
| `parse_task` | Deutsche NL-Phrase in Datum/RRULE parsen (ohne anzulegen) | `nldate.Parse` |

Fehler aus der Service-Schicht (z. B. Validierung, „not found") werden als
MCP-Tool-Fehler mit der Fehlermeldung zurückgegeben, nicht als Transportfehler.

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
| `comment.created` | `POST /api/tasks/{id}/comments` |
| `epic.created` | `POST /api/epics` |
| `epic.updated` | `PATCH /api/epics/{id}`, `reopen` (auch implizit beim Wiederöffnen einer Aufgabe) |
| `epic.completed` | `POST /api/epics/{id}/complete` |
| `epic.deleted` | `DELETE /api/epics/{id}` (Archivierung) |
| `absence.created` | `POST /api/absences` |
| `absence.updated` | `PATCH /api/absences/{id}` |
| `absence.deleted` | `DELETE /api/absences/{id}` |

---

## Webhooks

Ist `WEBHOOK_URL` gesetzt, hängt sich beim Start ein Dispatcher
(`internal/webhook`) an den Eventbus und schickt **jedes** Event als
`POST` an diese eine URL. Minimale Umsetzung von Lastenheft 5.11: eine URL,
keine Auth, keine Filterung pro Event-Typ, kein Retry.

```json
{
  "event": "task.completed",
  "timestamp": "2026-05-22T18:30:00Z",
  "data": { "id": "task_8f3e2c1d4a5b6789", "title": "…", "status": "completed" }
}
```

- `Content-Type: application/json`, Timeout **10 s**.
- Der Versand läuft in der Bus-Goroutine — er blockiert nie die auslösende
  HTTP-Anfrage. Fehler und `4xx`/`5xx`-Antworten werden nur geloggt
  (`slog.Warn`), die Auslieferung wird **nicht** wiederholt.
- `data` enthält bei **Task- und Projekt**-Events eine eigene snake_case-Form
  nach Lastenheft 5.11 — nicht identisch mit den REST-DTOs: die Regel heißt
  hier `recurrence` (statt `recurrence_rule`); `project_name`,
  `recurrence_label`, `comments` und `comment_count` fehlen.
- Alle übrigen Event-Typen (`comment.*`, `absence.*`, `epic.*`) werden **ohne**
  Umformung serialisiert, d. h. mit den Go-Feldnamen in PascalCase,
  `LocalDate` als verschachteltes Objekt und Zeitstempeln in
  Nanosekunden-Auflösung:

  ```json
  {
    "event": "absence.created",
    "timestamp": "2026-08-01T09:12:44Z",
    "data": {
      "ID": "absence_fd3a509d29cc66d3",
      "Type": "vacation",
      "Label": "Kreta",
      "Start": { "Year": 2026, "Month": 8, "Day": 3 },
      "End": { "Year": 2026, "Month": 8, "Day": 7 },
      "CreatedAt": "2026-08-01T09:12:44.099151559Z",
      "UpdatedAt": "2026-08-01T09:12:44.099151559Z"
    }
  }
  ```

  Wer darauf hört, sollte das wissen — eine Angleichung an snake_case wäre
  ein Breaking Change für bestehende Empfänger.

**Zustellung:** ein Versuch je Event, Timeout 10 s, asynchron auf der
Bus-Goroutine — eine HTTP-Anfrage wartet also nie auf den Webhook. Fehler und
Nicht-2xx-Antworten landen im Server-Log.

**Absicherung:** `/api/tasks`, `/api/projects`, `/api/epics`, `/api/absences`, `/api/audit`
und `/api/parse` tragen keine eigene Authentifizierung — sie liegen hinter der
BasicAuth des Reverse-Proxys. Token-geschützt und von dieser Auth ausgenommen
sind nur die Sonderrouten `.ics`, `.html` und `/mcp` (siehe oben).
