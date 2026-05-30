# Vereinfachtes Lastenheft: Webbasierte Todo-Liste mit API und Webhooks

**Ziel:** Minimal nutzbare Aufgabenmanagement-Anwendung nach ausgewählten Todoist-Prinzipien
**Scope:** Single-User, webbasiert, Listenansicht, API-first
**Nicht-Ziel:** Kollaboration, Mobile/Desktop-Apps, SSO, Analytics, Alarme, Kalenderintegration, Board-/Kanban-Ansicht

> **Hinweis zum Dokumentstatus:** Dieses Lastenheft ist mit dem Implementierungsstand
> abgeglichen. Abschnitte, die über den aktuellen Code hinausgehen oder bewusst
> reduziert umgesetzt wurden, sind mit einem **Status:**-Vermerk gekennzeichnet
> (umgesetzt · teilweise · geplant). Maßgeblich für das tatsächliche Verhalten ist
> der Code; siehe `CLAUDE.md` für die Architektur.

---

## 1. Zielsetzung

Ziel ist die Entwicklung einer schlanken webbasierten Aufgabenmanagement-Anwendung für persönliche Aufgabenverwaltung. Frontend: Ein modernes Web-Frontend. Backend: Backend in Go. Datenbank: SQLite.

Die Anwendung soll insbesondere folgende Fähigkeiten bieten:

* Aufgaben in Projekten organisieren
* Aufgaben in einer Listenansicht anzeigen
* Aufgaben mit Priorität versehen
* Aufgaben mit einem Erscheinungsdatum versehen
* Aufgaben mit einer Deadline versehen
* natürlichsprachige Eingabe für Termine und Wiederholungen unterstützen
* wiederkehrende Aufgaben verwalten
* eine API zur Automatisierung bereitstellen
* Webhooks zur Ereignisweitergabe bereitstellen

Die Anwendung soll nicht Todoist vollständig nachbauen, sondern nur die Funktionen abbilden, die für den konkreten Nutzungsfall relevant sind.

---

## 2. Abgrenzung

### In Scope

* Weboberfläche
* Listenansicht
* Projekte
* Aufgaben
* Prioritäten
* Erscheinungsdatum / Sichtbarkeitsdatum
* Deadline
* Wiederkehrende Aufgaben
* Natürlichsprachige Datumseingabe
* REST API oder vergleichbare HTTP API
* Webhooks
* Basisdatenhaltung
* Import/Export optional

### Out of Scope

* Mehrbenutzerfähigkeit
* Kollaboration
* Kommentare
* Datei-Anhänge
* Mobile Apps
* Desktop Apps
* Board-Ansicht
* Kalenderansicht
* Erinnerungen / Alarme / Push Notifications
* E-Mail-Benachrichtigungen
* Single Sign-on
* Rollen- und Rechtemanagement
* Analytics
* Team-Workspaces
* Gäste
* Aufgaben-Zuweisungen
* komplexe Admin-Funktionen
* KI-Funktionen

---

## 3. Nutzerrolle

Da keine Kollaboration vorgesehen ist, gibt es im MVP nur eine Rolle.

| Rolle  | Beschreibung                                                                                |
| ------ | ------------------------------------------------------------------------------------------- |
| Nutzer | Verwaltet eigene Aufgaben, Projekte, Prioritäten, Termine, Deadlines und Automatisierungen. |

Eine Authentifizierung kann für den MVP entfallen, sofern die Anwendung nur lokal, in einem geschützten Netzwerk oder hinter einem externen Schutzmechanismus betrieben wird.

---

## 4. Fachliche Kernobjekte

## 4.1 Aufgabe

Eine Aufgabe ist das zentrale Objekt der Anwendung.

### Attribute einer Aufgabe

| Attribut           | Beschreibung                                                 |
| ------------------ | ------------------------------------------------------------ |
| ID                 | Eindeutige technische Kennung                                |
| Titel              | Kurzbeschreibung der Aufgabe                                 |
| Beschreibung       | Optionaler Langtext                                          |
| Projekt            | Zugeordnetes Projekt                                         |
| Status             | Offen, erledigt, gelöscht/archiviert                         |
| Priorität          | Prioritätsstufe                                              |
| Erscheinungsdatum  | Datum, ab dem die Aufgabe in der aktiven Liste sichtbar wird |
| Deadline           | Datum, bis zu dem die Aufgabe erledigt sein sollte           |
| Wiederholungsregel | Optional, z. B. täglich, jeden Montag, jeden ersten Werktag  |
| Erstellt am        | Erstellzeitpunkt                                             |
| Aktualisiert am    | Letzte Änderung                                              |
| Erledigt am        | Zeitpunkt der Erledigung                                     |

Wichtig ist die klare Trennung von **Erscheinungsdatum** und **Deadline**.

Beispiel:

> „Steuerunterlagen vorbereiten“ erscheint am 01.03., hat aber Deadline am 31.03.

Das Erscheinungsdatum beantwortet: **Wann soll ich die Aufgabe sehen?**
Die Deadline beantwortet: **Bis wann muss sie erledigt sein?**

---

## 4.2 Projekt

Ein Projekt gruppiert Aufgaben thematisch.

### Attribute eines Projekts

| Attribut           | Beschreibung                                                     |
| ------------------ | ---------------------------------------------------------------- |
| ID                 | Eindeutige technische Kennung (`project_*`, Inbox = `inbox`)     |
| Name               | Projektname                                                      |
| Slug               | URL-sichere Kennung, aus dem Namen abgeleitet, stabil bei Umbenennung; eindeutig; dient dem Routing (`/projects/<slug>`) |
| Beschreibung       | Optional                                                        |
| Farbe              | Optionale CSS-Hex-Farbe (`#246fe0`), leer = Standardfarbe       |
| Sortierreihenfolge | Reihenfolge in der UI                                           |
| Aktiv/Inaktiv      | Projekt kann archiviert werden                                  |
| Erstellt am        | Erstellzeitpunkt                                               |
| Aktualisiert am    | Letzte Änderung                                                |

**Status:** umgesetzt. Slug und Farbe sind über die Implementierung hinzugekommen
und im Datenmodell (Abschnitt 7) sowie der API enthalten.

Beispiele:

* Privat
* Arbeit
* Garten
* Homelab
* Familie
* adesso
* Inbox

Ein Standardprojekt **Inbox** sollte automatisch existieren.

---

## 4.3 Priorität

Aufgaben müssen priorisierbar sein.

### Prioritätsmodell

| Priorität | Bedeutung    |
| --------- | ------------ |
| P1        | Sehr wichtig |
| P2        | Wichtig      |
| P3        | Normal       |
| P4        | Niedrig      |

Alternativ kann intern mit Zahlenwerten gearbeitet werden:

* `1 = höchste Priorität`
* `4 = niedrigste Priorität`

Die UI sollte Prioritäten sichtbar machen und Sortierung nach Priorität erlauben.

---

## 4.4 Erscheinungsdatum

Das Erscheinungsdatum bestimmt, ab wann eine Aufgabe in der Standardliste sichtbar wird.

### Anforderungen

* Aufgabe kann kein Erscheinungsdatum haben
* Aufgabe kann ein konkretes Erscheinungsdatum haben
* Aufgaben mit zukünftigem Erscheinungsdatum werden in der Standardansicht ausgeblendet
* Aufgaben erscheinen automatisch ab dem angegebenen Datum
* Überfällige Aufgaben bleiben sichtbar, bis sie erledigt sind
* Erscheinungsdatum kann unabhängig von Deadline geändert werden

Beispiele:

| Aufgabe                    | Erscheinungsdatum |   Deadline |
| -------------------------- | ----------------: | ---------: |
| Rechnung prüfen            |        20.05.2026 | 25.05.2026 |
| TÜV-Termin organisieren    |        01.06.2026 | 30.06.2026 |
| Quartalsreview vorbereiten |        15.06.2026 | 30.06.2026 |

---

## 4.5 Deadline

Die Deadline ist das echte Fälligkeitsdatum.

### Anforderungen

* Aufgabe kann keine Deadline haben
* Aufgabe kann eine Deadline haben
* Deadline ist unabhängig vom Erscheinungsdatum
* Deadline kann in der Liste angezeigt werden
* Aufgaben mit überschrittener Deadline werden visuell hervorgehoben
* Sortierung nach Deadline muss möglich sein
* Filterung nach Deadline muss möglich sein

Es werden keine Alarme oder Erinnerungen aus Deadlines abgeleitet.

---

## 4.6 Wiederkehrende Aufgaben

Das System muss wiederkehrende Aufgaben unterstützen.

### Anforderungen

Eine Aufgabe kann eine Wiederholungsregel besitzen, z. B.:

* täglich
* jeden Montag
* jeden Werktag
* jeden ersten Tag im Monat
* jeden letzten Freitag im Monat
* alle 2 Wochen
* alle 3 Monate
* jedes Jahr am 1. Januar

Beim Abschließen einer wiederkehrenden Aufgabe wird automatisch eine neue Instanz oder ein neuer Termin erzeugt.

### Offene fachliche Entscheidung

Es gibt zwei mögliche Modelle:

#### Modell A: Aufgabe wird weitergeschoben

Beim Erledigen wird dieselbe Aufgabe auf das nächste Erscheinungsdatum gesetzt.

Vorteile:

* einfach
* wenig Daten
* gute Todoist-Nähe

Nachteile:

* keine Historie einzelner Erledigungen

#### Modell B: Neue Aufgabeninstanz wird erzeugt

Beim Erledigen wird die aktuelle Aufgabe abgeschlossen und eine neue Aufgabe für den nächsten Termin erzeugt.

Vorteile:

* bessere Historie
* API-seitig sauberer
* spätere Auswertungen möglich

Nachteile:

* etwas komplexer

Für dieses Lastenheft wird **Modell B empfohlen**, weil es API- und Webhook-freundlicher ist.

---

## 4.7 Natürlichsprachige Datumseingabe

Das System muss natürlichsprachige Eingaben für Datum und Wiederholung unterstützen.

### Datumsphrasen (umgesetzt)

Der Parser (`internal/nldate`, Endpunkt `POST /api/parse`) erkennt deutsche
Datumsangaben. Erkannte Phrasen (case-insensitiv, Whitespace egal):

| Eingabe                          | Erwartetes Ergebnis                          |
| -------------------------------- | -------------------------------------------- |
| `heute` / `heu`                  | heute                                        |
| `morgen` / `mor`                 | morgen                                       |
| `übermorgen`                     | in zwei Tagen                                |
| `gestern`                        | gestern                                      |
| `montag`..`sonntag`, `mo`..`so`  | nächstes Vorkommen des Wochentags            |
| `nächsten Montag`                | wie bloßer Wochentag (nächstes Vorkommen)    |
| `nächste Woche`                  | Montag der nächsten ISO-Woche                |
| `nächsten Monat` / `nächstes Jahr` | gleicher Tag, begrenzt auf Monatslänge      |
| `in 3 Tagen` / `in 2 Wochen` / `in 1 Monat` / `in 1 Jahr` | relatives Datum     |
| `+5` / `+5 Tage` / `+5d`         | heute + N Tage                               |
| `anfang/mitte/ende <Monat>`      | 1. / 15. / letzter Tag des Monats            |
| `monatsanfang` / `monatsende`    | erster / letzter Tag des aktuellen Monats    |
| `27.1.` / `27.01.`               | konkretes Datum (laufendes Jahr, sonst nächstes) |
| `27.1.2027`                      | exaktes Datum                                |
| `2026-05-21`                     | ISO-Datum                                    |
| `KW32` / `kw 32`                 | Montag der 32. ISO-Kalenderwoche             |

### Wiederholungsphrasen

**Status:** **noch nicht umgesetzt.** Die natürlichsprachige Erkennung von
Wiederholungen (`jeden Montag`, `alle 2 Wochen`, `jeden ersten des Monats`) ist
in `internal/nldate` als dokumentierter Stub angelegt, aber nicht aktiv. `POST
/api/parse` liefert daher heute nur ein `visible_from`, kein `recurrence_rule`.

Wiederholungen werden stattdessen **strukturiert** gesetzt:

* im Frontend über ein Auswahlfeld mit Presets (Täglich, Werktäglich,
  Wöchentlich, Alle 2 Wochen, Monatlich, Alle 3 Monate, Jährlich, eigene RRULE)
* über die API direkt als RRULE-String im Feld `recurrence_rule`

Siehe Abschnitt 4.6 für den unterstützten RRULE-Umfang.

### Anforderungen

* natürliche Sprache mindestens auf Deutsch
* optional Englisch (**nicht umgesetzt**)
* Parser muss Datum, Deadline und Wiederholung unterscheiden können
  (Datum/Deadline: umgesetzt; Wiederholung: siehe oben)
* erkannte Werte müssen vor dem Speichern sichtbar oder prüfbar sein
* fehlerhafte oder mehrdeutige Eingaben dürfen nicht stillschweigend falsch
  interpretiert werden (unerkannte Eingabe → HTTP 422)

---

## 4.8 Kommentare

Eine Aufgabe kann beliebig viele Kommentare tragen. Ein Kommentar ist eine
zeitgestempelte Notiz mit Markdown-Inhalt.

### Attribute eines Kommentars

* `id` — eindeutige ID (`comment_*`)
* `task_id` — zugehörige Aufgabe
* `body` — Inhalt als Markdown-Quelltext
* `created_at` — Zeitpunkt der Erstellung

### Anforderungen

* Kommentare sind append-only (kein Bearbeiten oder Löschen im MVP)
* der Inhalt unterstützt Markdown; die Darstellung rendert das Markup
* Kommentare werden absteigend nach Aktualität (neueste zuerst) angezeigt
* das Setzen von Kommentaren muss auch automatisiert über die API möglich sein
* Kommentare können an Aufgaben in jedem Status gehängt werden (auch erledigte)

---

## 4.9 Aktivitätsprotokoll (Audit-Log)

**Status:** umgesetzt (über das ursprüngliche Lastenheft hinausgehend).

Das System führt ein Protokoll über die wichtigsten Zustandsänderungen einer
Aufgabe. Jeder Eintrag wird beim Eintreten der Änderung im Service-Layer
geschrieben und ist über die API sowie die Frontend-Ansicht „Aktivität"
einsehbar.

### Protokollierte Aktionen

| Aktion                   | Bedeutung                                |
| ------------------------ | ---------------------------------------- |
| `created`                | Aufgabe erstellt                         |
| `completed`              | Aufgabe abgeschlossen                    |
| `reopened`               | Abgeschlossene Aufgabe wieder geöffnet   |
| `deleted`                | Aufgabe gelöscht                         |
| `visible_from_set/cleared/changed` | Erscheinungsdatum gesetzt/entfernt/geändert |
| `deadline_set/cleared/changed`     | Deadline gesetzt/entfernt/geändert        |
| `project_changed`        | Projekt gewechselt (Werte = Projektnamen) |

### Attribute eines Eintrags

* `id` — eindeutige ID (`audit_*`)
* `task_id` — zugehörige Aufgabe
* `task_title` — Titel-Schnappschuss (bleibt aussagekräftig, auch wenn der
  Titel später geändert oder die Aufgabe hart gelöscht würde)
* `action` — eine der obigen Aktionen
* `old_value` / `new_value` — kontextabhängig (Datum `YYYY-MM-DD` bzw. `""`,
  Projektname; bei created/completed/reopened/deleted leer)
* `at` — Zeitpunkt der Änderung

### Anforderungen

* Einträge sind append-only
* abrufbar gefiltert nach Aufgabe und nach Kalendertag/-bereich
* die Frontend-Ansicht „Aktivität" zeigt das Protokoll nach Tag gruppiert

---

## 5. Funktionale Anforderungen

## 5.1 Aufgaben erstellen

### LA-F-001 Aufgabe erstellen

Das System muss das Erstellen neuer Aufgaben ermöglichen.

**Muss-Kriterien:**

* Titel erfassen
* optional Projekt setzen
* optional Priorität setzen
* optional Erscheinungsdatum setzen
* optional Deadline setzen
* optional Wiederholungsregel setzen
* natürlichsprachige Eingabe verarbeiten
* Aufgabe speichern
* Aufgabe in Liste anzeigen, sofern sichtbar

Beispiel:

> `Rasen mähen jeden Samstag ab nächstem Samstag P2`

Erwartetes Ergebnis:

* Titel: Rasen mähen
* Wiederholung: wöchentlich Samstag
* Erscheinungsdatum: nächster Samstag
* Priorität: P2

---

## 5.2 Aufgaben bearbeiten

### LA-F-002 Aufgabe bearbeiten

Das System muss bestehende Aufgaben bearbeiten können.

**Muss-Kriterien:**

* Titel ändern
* Beschreibung ändern
* Projekt ändern
* Priorität ändern
* Erscheinungsdatum ändern
* Deadline ändern
* Wiederholungsregel ändern
* Status ändern
* Änderungen über UI und API möglich

---

## 5.3 Aufgaben abschließen

### LA-F-003 Aufgabe abschließen

Das System muss Aufgaben als erledigt markieren können.

**Muss-Kriterien:**

* Aufgabe erhält Status `erledigt`
* Erledigungszeitpunkt wird gespeichert
* Aufgabe verschwindet aus Standardliste
* bei wiederkehrender Aufgabe wird nächster Termin erzeugt
* Webhook `task.completed` wird ausgelöst
* bei wiederkehrender Aufgabe zusätzlich `task.created` und `task.recurrence_generated`
  (Modell B, siehe 4.6)

### LA-F-003a Aufgabe wieder öffnen

**Status:** umgesetzt (über das ursprüngliche Lastenheft hinausgehend).

Eine abgeschlossene Aufgabe kann wieder geöffnet werden.

**Muss-Kriterien:**

* Status wechselt zurück auf `offen`, der Erledigungszeitpunkt wird entfernt
* idempotent: bei bereits offener Aufgabe passiert nichts
* über UI und API (`POST /api/tasks/{id}/reopen`) möglich
* erzeugt einen `reopened`-Eintrag im Aktivitätsprotokoll und ein `task.updated`-Event

> Hinweis: Das Wiederöffnen erzeugt **keine** neue Instanz einer wiederkehrenden
> Aufgabe (diese entsteht nur beim Abschließen).

---

## 5.4 Aufgaben löschen oder archivieren

### LA-F-004 Aufgabe löschen

Das System muss Aufgaben entfernen können.

**Muss-Kriterien:**

* Aufgabe kann gelöscht oder archiviert werden
* gelöschte Aufgabe erscheint nicht mehr in Standardliste
* Löschung löst Webhook aus
* API erlaubt Löschen

Empfehlung: fachlich zunächst **Soft Delete** verwenden, damit Webhook- und API-Konsistenz einfacher bleibt.

---

## 5.5 Listenansicht

### LA-F-005 Listenansicht

Das System muss eine zentrale Listenansicht bereitstellen.

**Muss-Kriterien:**

* Anzeige aller offenen, aktuell sichtbaren Aufgaben
* zukünftige Aufgaben standardmäßig ausblenden
* überfällige Aufgaben anzeigen
* Deadline anzeigen
* Projekt anzeigen
* Priorität anzeigen
* Wiederholungsstatus anzeigen
* Sortierung ermöglichen
* Filterung ermöglichen

### Standardlogik der Liste

Eine Aufgabe erscheint in der Standardliste, wenn:

```text
Status = offen
UND
(Erscheinungsdatum leer ODER Erscheinungsdatum <= heute)
```

Deadlines beeinflussen die Sichtbarkeit nicht, sondern nur Anzeige, Hervorhebung und Sortierung.

### Implementierte Ansichten

Das Frontend ist eine Single-Page-App mit clientseitigem Routing. Folgende
Ansichten sind umgesetzt:

| Ansicht     | Route                | Inhalt                                                       |
| ----------- | -------------------- | ------------------------------------------------------------ |
| Heute       | `/today` (Default)   | offene, aktuell sichtbare Aufgaben; zusätzlich heute erledigte (ausgegraut) |
| Demnächst   | `/upcoming`          | künftige Aufgaben, nach Kalenderwoche gruppiert/hervorgehoben |
| Inbox       | `/inbox`             | Aufgaben des Standardprojekts                                |
| Projekt     | `/projects/<slug>`   | Aufgaben eines Projekts                                      |
| Aktivität   | `/activity`          | Aktivitätsprotokoll (siehe 4.9), nach Tag gruppiert          |

Die gruppierten Ansichten rufen die API mit `scope=all` ab und wenden die
Sichtbarkeitslogik clientseitig an. Aufgabendetails sind direkt verlinkbar
(`/tasks/<id>`).

---

## 5.6 Sortierung

### LA-F-006 Aufgaben sortieren

Das System muss Aufgaben sortieren können.

**Muss-Kriterien:**

Sortierung nach (API-Parameter `sort`):

* Priorität (`priority`)
* Deadline (`deadline`)
* Erscheinungsdatum (`visible_from`)
* Erstellzeitpunkt (`created_at`)
* Standard (leer) — empfohlene Standardsortierung (siehe unten)
* Projekt — **nicht als eigener Sort-Modus umgesetzt** (Projekte sind in der UI
  ohnehin eigene Ansichten)
* manuelle Reihenfolge — **nicht umgesetzt** für Aufgaben (nur Projekte besitzen
  ein `sort_order`)

Empfohlene Standardsortierung (umgesetzt als Default `SortDefault`):

1. überfällige Deadlines zuerst
2. hohe Priorität zuerst
3. frühere Deadline zuerst
4. ältere Aufgaben zuerst

---

## 5.7 Filter

### LA-F-007 Aufgaben filtern

Das System muss einfache Filter unterstützen.

**Muss-Kriterien:**

Filter nach (API-Query-Parameter auf `GET /api/tasks`):

* Projekt (`project_id`) — umgesetzt
* Status (`status` = open/completed/deleted) — umgesetzt
* Sichtbarkeit (`scope=all` schaltet die Erscheinungsdatum-Logik ab) — umgesetzt
* Erscheinungsdatum als Tag/Bereich/KW (`date` · `from`/`to` · `week=YYYY-Www`) — umgesetzt
* Erledigungsdatum als Tag/Bereich/KW (`completed_date` · `completed_from`/`completed_to` · `completed_week`) — umgesetzt
* Priorität — **derzeit nicht über die HTTP-API filterbar** (das Storage-Filter
  unterstützt es, der Handler liest den Parameter aber noch nicht aus)
* wiederkehrend / nicht wiederkehrend — **nicht umgesetzt**

Beispiele:

* alle offenen Aufgaben
* alle Aufgaben im Projekt „Homelab“
* alle Aufgaben mit Erscheinungsdatum in einer bestimmten Kalenderwoche
* alle in einem Zeitraum erledigten Aufgaben

Komplexe Todoist-Filter-Syntax ist für den MVP nicht erforderlich.

---

## 5.8 Projekte verwalten

### LA-F-008 Projekte erstellen und bearbeiten

Das System muss Projekte verwalten können.

**Muss-Kriterien:**

* Projekt erstellen
* Projekt umbenennen
* Projekt archivieren
* Projekt löschen, sofern keine aktiven Aufgaben enthalten sind oder explizit bestätigt
* Aufgaben einem Projekt zuordnen
* Inbox als Standardprojekt

---

## 5.9 API

### LA-F-009 API bereitstellen

Das System muss eine HTTP-basierte API bereitstellen.

**Muss-Kriterien:**

API-Endpunkte für:

* Aufgaben erstellen, lesen, aktualisieren, abschließen, wieder öffnen, löschen
* Kommentare einer Aufgabe lesen / erstellen
* Aktivitätsprotokoll lesen (global und je Aufgabe)
* Projekte erstellen, lesen, aktualisieren, löschen/archivieren
* natürlichsprachige Eingabe parsen

### Tatsächlich umgesetzte API-Endpunkte

```http
GET    /api/tasks
POST   /api/tasks
GET    /api/tasks/{id}
PATCH  /api/tasks/{id}
POST   /api/tasks/{id}/complete
POST   /api/tasks/{id}/reopen
DELETE /api/tasks/{id}
GET    /api/tasks/{id}/comments
POST   /api/tasks/{id}/comments
GET    /api/tasks/{id}/audit

GET    /api/audit

GET    /api/projects
POST   /api/projects
GET    /api/projects/{id}
PATCH  /api/projects/{id}
DELETE /api/projects/{id}

POST   /api/parse
```

**Abweichungen gegenüber dem ursprünglichen Entwurf:**

* `POST /api/tasks/{id}/reopen`, `GET /api/tasks/{id}/audit` und `GET /api/audit`
  sind neu hinzugekommen.
* Eigene Endpunkte zum Lesen von „Prioritäten" und „Wiederholungsregeln" gibt es
  nicht — beide sind feste Wertebereiche (P1–P4 bzw. der RRULE-Subset aus 4.6)
  und werden nicht über die API ausgeliefert.
* **Webhook-Verwaltungsendpunkte (`/api/webhooks`) existieren nicht.** Webhooks
  sind reduziert über eine einzige konfigurierte Ziel-URL umgesetzt (siehe 5.11).

Hinweise zu Listenparametern: `GET /api/tasks` akzeptiert die Filter-/Sortier-
Parameter aus 5.6/5.7. `GET /api/projects` liefert standardmäßig nur aktive
Projekte; `include_archived=true` schließt archivierte ein (das Frontend nutzt
dies, damit historische Projektlabels weiter auflösen).

---

## 5.10 API-Authentifizierung

Auch wenn kein Sign-on vorgesehen ist, sollte die API nicht ungeschützt sein, sobald sie außerhalb eines lokalen Netzwerks erreichbar ist.

**Status:** **noch nicht umgesetzt.** Die Anwendung enthält derzeit keine
Authentifizierung; sämtliche API- und UI-Endpunkte sind offen. Der Betrieb setzt
daher voraus, dass die App nur lokal, im geschützten Netz oder hinter einem
externen Schutzmechanismus (Reverse-Proxy mit Auth, VPN o. Ä.) erreichbar ist.

### Minimalanforderung (geplant)

* statischer API-Key
* Übergabe per Header

Beispiel:

```http
Authorization: Bearer <api-key>
```

### Out of Scope

* OAuth
* User Login
* SSO
* Rollenmodell
* Multi-User-Tokens

---

## 5.11 Webhooks

### LA-F-010 Webhooks bereitstellen

Das System muss Webhooks unterstützen.

**Status:** **reduziert umgesetzt.** Aktuell existiert ein einzelner
Webhook-Empfänger: Ist die Umgebungsvariable `WEBHOOK_URL` gesetzt, wird **jedes**
Event des internen Event-Bus als JSON-POST an diese URL zugestellt. Es gibt
**keine** Verwaltungs-API, keine Auswahl einzelner Events pro Webhook, kein
Secret, keine Signatur, keinen Retry und keine Zustelllogs. Die Zustellung läuft
asynchron auf dem Event-Bus und blockiert UI-Aktionen nicht (Timeout 10 s). Die
ausführlichen Muss-Kriterien unten beschreiben den **Zielzustand** und sind
noch offen.

**Muss-Kriterien:**

Webhook-Ereignisse bei (im Code als Event-Typen vorhanden und ausgelöst):

* Aufgabe erstellt
* Aufgabe aktualisiert
* Aufgabe abgeschlossen
* Aufgabe gelöscht
* Aufgabe verschoben / neues Erscheinungsdatum
* Deadline geändert
* Projekt erstellt
* Projekt aktualisiert
* Projekt gelöscht
* wiederkehrende Aufgabe erzeugt nächste Instanz
* Kommentar erstellt

### Beispielhafte Events

```text
task.created
task.updated
task.completed
task.deleted
task.rescheduled
task.deadline_changed
task.recurrence_generated
project.created
project.updated
project.deleted
comment.created
```

### Webhook-Payload Beispiel

```json
{
  "event": "task.completed",
  "timestamp": "2026-05-20T21:30:00Z",
  "data": {
    "id": "task_123",
    "title": "Rasen mähen",
    "project_id": "project_garden",
    "priority": 2,
    "status": "completed",
    "visible_from": "2026-05-20",
    "deadline": null,
    "recurrence": "FREQ=WEEKLY;BYDAY=SA"
  }
}
```

### Technische Muss-Kriterien

| Kriterium                                            | Status                              |
| ---------------------------------------------------- | ----------------------------------- |
| Ziel-URL konfigurierbar                              | umgesetzt (eine URL via `WEBHOOK_URL`) |
| Events pro Webhook auswählbar                        | geplant (heute: alle Events)        |
| Secret pro Webhook                                   | geplant                             |
| Signatur im Header                                   | geplant                             |
| Retry bei Fehlern                                    | geplant (heute: einmaliger Versuch) |
| Timeout-Konfiguration                                | teilweise (fester Timeout 10 s)     |
| Webhook-Logs (letzte Zustellversuche)                | geplant (heute: nur Server-Log)     |
| Deaktivierung fehlerhafter Webhooks                  | geplant                             |

Empfohlener Signaturheader (für die spätere Umsetzung):

```http
X-Webhook-Signature: sha256=<signature>
```

---

## 5.12 Kommentare

### LA-F-011 Aufgaben kommentieren

Das System muss erlauben, eine Aufgabe mit Kommentaren zu versehen.

**Muss-Kriterien:**

* unter einer Aufgabe können Kommentare erfasst werden (Eingabefeld + Absenden)
* das Eingabefeld unterstützt Markdown
* nach dem Absenden erscheinen die Kommentare absteigend nach Aktualität
  (neueste zuerst) unterhalb des Eingabefeldes
* Kommentare können auch automatisiert über die API gesetzt werden
* in der Listenansicht zeigt ein Eintrag eine kleine Sprechblase mit der Anzahl
  seiner Kommentare — nur wenn mindestens ein Kommentar vorhanden ist
* das Erstellen eines Kommentars löst das Event `comment.created` aus

**Verhalten:**

* leere Kommentare (nur Leerzeichen) werden abgelehnt
* Kommentare sind append-only — kein Bearbeiten oder Löschen im MVP

---

## 6. Nicht-funktionale Anforderungen

## 6.1 Web-only

Das System wird ausschließlich als Webanwendung bereitgestellt.

**Muss-Kriterien:**

* responsives Web-UI (Desktop + Mobile, einklappbare Sidebar)
* nutzbar auf Desktop-Browsern
* nutzbar auf mobilen Browsern
* keine native App erforderlich
* keine Desktop-App erforderlich
* **Dark Mode** (umgesetzt): folgt automatisch der Systemeinstellung via
  `prefers-color-scheme`; keine manuelle Umschaltung
* **Kein Build-Schritt** im Frontend: `index.html`, `style.css`, `app.js` werden
  per `go:embed` direkt in das Binary eingebettet

---

## 6.2 Einfachheit der Bedienung

**Muss-Kriterien:**

* neue Aufgabe schnell erfassbar
* Listenansicht steht im Zentrum
* keine überladenen Projektmanagementfunktionen
* natürlichsprachige Eingabe prominent verfügbar
* Tastaturbedienung für zentrale Aktionen

Shortcuts:

| Shortcut | Funktion         | Status            |
| -------- | ---------------- | ----------------- |
| `n`      | neue Aufgabe (Quick-Add fokussieren) | umgesetzt |
| `Enter`  | speichern (Quick-Add, Dialoge), `Strg/Cmd+Enter` für Kommentare | umgesetzt |
| `Escape` | Dialog/Sidebar schließen | umgesetzt |
| `e`      | bearbeiten       | nicht umgesetzt   |
| `d`      | erledigen        | nicht umgesetzt   |
| `p`      | Priorität ändern | nicht umgesetzt   |
| `/`      | Suche/Filter     | nicht umgesetzt   |

Die natürlichsprachige Eingabe ist prominent als Quick-Add verfügbar; im
Quick-Add navigieren Pfeiltasten durch Vorschläge.

---

## 6.3 Performance

**Muss-Kriterien:**

* Listenansicht lädt unter normalen Bedingungen in unter 1 Sekunde
* neue Aufgabe wird ohne spürbare Verzögerung gespeichert
* API antwortet bei Standardoperationen typischerweise unter 300 ms
* Webhook-Versand darf UI-Aktionen nicht blockieren

---

## 6.4 Datenhaltung

**Muss-Kriterien:**

* persistente Speicherung aller Aufgaben und Projekte in SQLite
* Statusänderungen nachvollziehbar über Zeitstempel
* wiederkehrende Aufgaben robust berechenbar
* Zeitzone muss korrekt berücksichtigt werden
* Datum ohne Uhrzeit muss als lokales Datum behandelt werden

Empfehlung:

* Wiederholungsregeln intern idealerweise als RRULE-kompatibles Format speichern

---

## 7. Datenmodell, fachlich

### Task

```json
{
  "id": "task_123",
  "title": "Rasen mähen",
  "description": "",
  "project_id": "project_garden",
  "status": "open",
  "priority": 2,
  "visible_from": "2026-05-23",
  "deadline": null,
  "recurrence_rule": "FREQ=WEEKLY;BYDAY=SA",
  "created_at": "2026-05-20T20:00:00Z",
  "updated_at": "2026-05-20T20:00:00Z",
  "completed_at": null
}
```

Die API-Antwort (`taskDTO`) reichert dieses Modell zusätzlich an: `project_name`
(aufgelöster Projektname), `recurrence_label` (deutsche Klartext-Beschreibung der
Wiederholung), `comment_count` sowie die eingebetteten `comments`.

### Comment

```json
{
  "id": "comment_123",
  "task_id": "task_123",
  "body": "Habe den **Rasenmäher** aus der Werkstatt geholt.",
  "created_at": "2026-05-20T20:05:00Z"
}
```

### Project

```json
{
  "id": "project_garden",
  "name": "Garten",
  "slug": "garten",
  "description": "Aufgaben rund um Kleingarten und Hütte",
  "sort_order": 20,
  "archived": false,
  "color": "#246fe0",
  "created_at": "2026-05-20T20:00:00Z",
  "updated_at": "2026-05-20T20:00:00Z"
}
```

### AuditEntry

```json
{
  "id": "audit_123",
  "task_id": "task_123",
  "task_title": "Rasen mähen",
  "action": "deadline_changed",
  "old_value": "2026-05-25",
  "new_value": "2026-05-30",
  "at": "2026-05-20T21:30:00Z"
}
```

### Webhook (geplant)

Das folgende Modell beschreibt den Zielzustand der Webhook-Verwaltung
(Abschnitt 5.11) und ist **noch nicht umgesetzt**. Heute existiert nur eine
einzelne, per `WEBHOOK_URL` konfigurierte Ziel-URL ohne persistiertes
Webhook-Objekt.

```json
{
  "id": "wh_123",
  "url": "https://example.com/webhook/todos",
  "events": [
    "task.created",
    "task.completed",
    "task.recurrence_generated"
  ],
  "secret": "stored-encrypted-or-hashed",
  "active": true,
  "created_at": "2026-05-20T20:00:00Z"
}
```

---

## 8. MVP-Funktionsumfang

Der MVP enthält ausschließlich:

### Aufgaben

* erstellen
* bearbeiten
* erledigen
* wieder öffnen
* löschen/archivieren
* anzeigen
* kommentieren
* Aktivitätsprotokoll je Aufgabe

### Projekte

* erstellen
* bearbeiten
* archivieren/löschen
* Aufgaben zuordnen

### Terminlogik

* Erscheinungsdatum
* Deadline
* wiederkehrende Aufgaben
* natürlichsprachige Datumseingabe

### Organisation

* Prioritäten
* Listenansicht
* einfache Sortierung
* einfache Filter

### Automatisierung

* API (umgesetzt)
* Webhooks (reduziert: eine Ziel-URL, siehe 5.11)
* API-Key (geplant, noch nicht umgesetzt — siehe 5.10)

### Web-UI

* Ansichten Heute / Demnächst / Inbox / Projekt / Aktivität
* Quick-Add mit natürlichsprachiger Datumseingabe
* Aufgabenformular inkl. Kommentaren
* Projektverwaltung (inkl. Farbe)
* Filter-/Sortierung über API-Parameter
* Dark Mode (systemgesteuert)

---

## 9. Abnahmekriterien

Das System gilt als fachlich abgenommen, wenn folgende Szenarien funktionieren:

1. Eine Aufgabe kann über die Weboberfläche erstellt werden.
2. Eine Aufgabe kann einem Projekt zugeordnet werden.
3. Eine Aufgabe kann eine Priorität P1 bis P4 erhalten.
4. Eine Aufgabe kann ein Erscheinungsdatum erhalten.
5. Eine Aufgabe mit zukünftigem Erscheinungsdatum erscheint nicht in der Standardliste.
6. Eine Aufgabe erscheint automatisch ab ihrem Erscheinungsdatum in der Standardliste.
7. Eine Aufgabe kann eine Deadline erhalten.
8. Eine überfällige Deadline wird in der Liste hervorgehoben.
9. Eine Aufgabe kann als erledigt markiert werden.
10. Eine wiederkehrende Aufgabe erzeugt beim Abschließen den nächsten Termin (Modell B).
11. Natürlichsprachliche Datumseingaben wie `morgen`, `Freitag`, `in 3 Tagen`, `KW32` werden korrekt interpretiert. *(Wiederholungs-Phrasen wie `jeden Montag` werden derzeit **nicht** per natürlicher Sprache erkannt — Wiederholungen werden über Auswahlfeld/RRULE gesetzt; siehe 4.7.)*
12. Aufgaben können nach Projekt gefiltert werden.
13. Aufgaben können nach Priorität sortiert werden.
14. Aufgaben können nach Deadline sortiert werden.
15. Aufgaben können über die API erstellt werden.
16. Aufgaben können über die API abgeschlossen werden.
17. Beim Erstellen einer Aufgabe wird ein Webhook ausgelöst.
18. Beim Abschließen einer Aufgabe wird ein Webhook ausgelöst.
19. Webhook-Payloads enthalten alle relevanten Aufgabendaten.
20. Webhook-Fehler werden protokolliert und erneut versucht.
21. Eine Aufgabe kann über die Weboberfläche kommentiert werden; Kommentare werden mit Markdown gerendert und neueste zuerst angezeigt.
22. Kommentare können über die API gelesen und erstellt werden.
23. In der Listenansicht erscheint bei kommentierten Aufgaben eine Sprechblase mit der Kommentaranzahl.
24. Eine abgeschlossene Aufgabe kann über UI und API wieder geöffnet werden.
25. Zustandsänderungen einer Aufgabe (Erstellen, Abschließen, Wieder­öffnen, Löschen, Datums- und Projektänderungen) werden im Aktivitätsprotokoll festgehalten und sind nach Aufgabe und Zeitraum abrufbar.
26. Die Oberfläche folgt automatisch dem hellen/dunklen Systemfarbschema.

---

## 10. Explizit gestrichene Anforderungen gegenüber Todoist

Folgende Todoist-nahe Funktionen werden bewusst nicht übernommen:

* Kollaboration
* Teams
* Gäste
* Datei-Uploads
* Zuweisungen
* Labels, sofern nicht später gewünscht
* Board-Ansicht
* Kalenderansicht
* Mobile Apps
* Desktop Apps
* E-Mail-Add-ons
* Erinnerungen
* Push Notifications
* Location Reminders
* Analytics
* Produktivitätsstatistiken
* Gamification
* Single Sign-on
* Rollenmodell
* Admin-Konsole
* KI-Assistenz
* Templates
* komplexe Custom Filter Syntax

> Hinweis: Kommentare wurden bewusst aus dieser Streichliste herausgenommen und
> in Abschnitt 4.8 / 5.12 aufgenommen. Gemeint sind hier ausschließlich
> persönliche Notizen an einer Aufgabe (Single-User), nicht die
> Kollaborations-Kommentare aus Todoist (Mentions, Zuweisungen, Benachrichtigungen).

---

## 11. Kurzbeschreibung des Zielprodukts

Das Zielprodukt ist eine **schlanke, persönliche, webbasierte Todo-Anwendung mit API-first-Charakter**.

Der Kern ist nicht Kollaboration, sondern:

> Aufgaben sichtbar machen, wenn sie relevant werden; Deadlines separat verfolgen; wiederkehrende Aufgaben zuverlässig neu erzeugen; und alles über API und Webhooks automatisierbar machen.

Damit ist das Produkt eher ein **persönlicher Aufgaben-Kernel mit Web-UI und Automatisierungsschnittstellen** als eine klassische Team-Produktivitätsplattform.
