# Todo

Schlanke, persönliche, webbasierte Todo-Anwendung nach den Vorgaben in [LASTENHEFT.md](LASTENHEFT.md).

- **Backend:** Go, stdlib `net/http`, pure-Go SQLite (`modernc.org/sqlite`)
- **Frontend:** Vanilla HTML/CSS/JS, an Todoist orientiert, in das Binary eingebettet
- **Status:** MVP-Kern (Tasks, Projekte, Listenansicht, REST-API). Erweiterungspunkte für Wiederholungen, NLP-Parser, API-Auth und Webhooks sind angelegt.

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
internal/httpapi/       REST-Handler nach Lastenheft 5.9
internal/web/           Eingebettete HTML/CSS/JS-Assets
```

Erweiterungspunkte:
- **Webhooks:** `bus.Subscribe(handler)` in `cmd/server/main.go`. Events sind in `internal/events/events.go` definiert.
- **Wiederholungsregeln:** Feld `recurrence_rule` ist in Storage und API durchgereicht. `service.CompleteTask` hat die Stelle markiert, an der `task.recurrence_generated` ausgelöst werden soll.
- **API-Auth:** Middleware vor `httpapi.New(svc)` einhängen, Bearer-Token prüfen.
- **NLP-Parser:** Eigenes Paket `internal/parse/`, mit Eingang über `POST /api/parse`.

## REST-API (umgesetzt)

| Methode | Pfad | Wirkung |
| --- | --- | --- |
| GET | `/api/tasks` | Liste; Query: `project_id`, `status`, `scope=all`, `sort=priority\|deadline\|visible_from\|created_at` |
| POST | `/api/tasks` | Anlegen |
| GET | `/api/tasks/{id}` | Einzel |
| PATCH | `/api/tasks/{id}` | Teilupdate |
| POST | `/api/tasks/{id}/complete` | Abschließen |
| DELETE | `/api/tasks/{id}` | Soft-Delete |
| GET | `/api/projects` | Liste; `?include_archived=true` |
| POST | `/api/projects` | Anlegen |
| GET/PATCH/DELETE | `/api/projects/{id}` | Standard |

## Noch nicht implementiert (bewusst aus dem MVP-Kern ausgenommen)

- Wiederkehrende Aufgaben (Modell B aus 4.6) — Datenfeld vorhanden, Engine fehlt
- Natürlichsprachige Datumseingabe (4.7) — UI-Feld kann später NLP-Result vorbefüllen
- Webhooks (5.11) — Eventbus liegt bereit
- API-Authentifizierung (5.10) — Middleware-Slot frei
- Komplexere Filter & manuelle Sortierung (5.6/5.7)
