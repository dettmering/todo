# Vereinfachtes Lastenheft: Webbasierte Todo-Liste mit API und Webhooks

**Ziel:** Minimal nutzbare Aufgabenmanagement-Anwendung nach ausgewählten Todoist-Prinzipien
**Scope:** Single-User, webbasiert, Listenansicht, API-first
**Nicht-Ziel:** Kollaboration, Mobile/Desktop-Apps, SSO, Analytics, Alarme, Kalenderintegration, Board-/Kanban-Ansicht

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

| Attribut           | Beschreibung                   |
| ------------------ | ------------------------------ |
| ID                 | Eindeutige technische Kennung  |
| Name               | Projektname                    |
| Beschreibung       | Optional                       |
| Sortierreihenfolge | Reihenfolge in der UI          |
| Aktiv/Inaktiv      | Projekt kann archiviert werden |
| Erstellt am        | Erstellzeitpunkt               |
| Aktualisiert am    | Letzte Änderung                |

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

### Beispiele

| Eingabe                      | Erwartetes Ergebnis                        |
| ---------------------------- | ------------------------------------------ |
| `morgen`                     | Erscheinungsdatum morgen                   |
| `nächsten Montag`            | Erscheinungsdatum nächster Montag          |
| `jeden Montag`               | Wiederholung wöchentlich montags           |
| `alle 2 Wochen`              | Wiederholung alle 2 Wochen                 |
| `jeden ersten des Monats`    | monatliche Wiederholung                    |
| `am 31.12.`                  | konkretes Datum                            |
| `in 3 Tagen`                 | relatives Erscheinungsdatum                |
| `KW41` | Am Montag der 41. Kalenderwoche |

### Anforderungen

* natürliche Sprache mindestens auf Deutsch
* optional Englisch
* Parser muss Datum, Deadline und Wiederholung unterscheiden können
* erkannte Werte müssen vor dem Speichern sichtbar oder prüfbar sein
* fehlerhafte oder mehrdeutige Eingaben dürfen nicht stillschweigend falsch interpretiert werden

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
* bei wiederkehrender Aufgabe zusätzlich `task.created` oder `task.rescheduled`, je nach Datenmodell

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

---

## 5.6 Sortierung

### LA-F-006 Aufgaben sortieren

Das System muss Aufgaben sortieren können.

**Muss-Kriterien:**

Sortierung nach:

* Priorität
* Deadline
* Erscheinungsdatum
* Projekt
* Erstellzeitpunkt
* manuelle Reihenfolge

Empfohlene Standardsortierung:

1. überfällige Deadlines zuerst
2. hohe Priorität zuerst
3. frühere Deadline zuerst
4. ältere Aufgaben zuerst

---

## 5.7 Filter

### LA-F-007 Aufgaben filtern

Das System muss einfache Filter unterstützen.

**Muss-Kriterien:**

Filter nach:

* Projekt
* Priorität
* Status
* Erscheinungsdatum
* Deadline
* wiederkehrend / nicht wiederkehrend

Beispiele:

* alle offenen Aufgaben
* alle Aufgaben im Projekt „Homelab“
* alle P1-Aufgaben
* alle Aufgaben mit Deadline diese Woche
* alle wiederkehrenden Aufgaben

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

* Aufgaben erstellen
* Aufgaben lesen
* Aufgaben aktualisieren
* Aufgaben abschließen
* Aufgaben löschen
* Projekte erstellen
* Projekte lesen
* Projekte aktualisieren
* Projekte löschen oder archivieren
* Prioritäten lesen
* Wiederholungsregeln lesen
* natürlichsprachige Eingabe parsen
* Webhook-Abonnements verwalten

### Beispielhafte API-Endpunkte

```http
GET    /api/tasks
POST   /api/tasks
GET    /api/tasks/{id}
PATCH  /api/tasks/{id}
POST   /api/tasks/{id}/complete
DELETE /api/tasks/{id}

GET    /api/projects
POST   /api/projects
GET    /api/projects/{id}
PATCH  /api/projects/{id}
DELETE /api/projects/{id}

POST   /api/parse
GET    /api/webhooks
POST   /api/webhooks
DELETE /api/webhooks/{id}
```

---

## 5.10 API-Authentifizierung

Auch wenn kein Sign-on vorgesehen ist, sollte die API nicht ungeschützt sein, sobald sie außerhalb eines lokalen Netzwerks erreichbar ist.

### Minimalanforderung

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

**Muss-Kriterien:**

Webhook-Ereignisse bei:

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

* Ziel-URL konfigurierbar
* Events pro Webhook auswählbar
* Secret pro Webhook
* Signatur im Header
* Retry bei Fehlern
* Timeout-Konfiguration
* Webhook-Logs mindestens für die letzten Zustellversuche
* Deaktivierung fehlerhafter Webhooks nach wiederholtem Fehlschlag

Empfohlener Signaturheader:

```http
X-Webhook-Signature: sha256=<signature>
```

---

## 6. Nicht-funktionale Anforderungen

## 6.1 Web-only

Das System wird ausschließlich als Webanwendung bereitgestellt.

**Muss-Kriterien:**

* responsives Web-UI
* nutzbar auf Desktop-Browsern
* nutzbar auf mobilen Browsern
* keine native App erforderlich
* keine Desktop-App erforderlich

---

## 6.2 Einfachheit der Bedienung

**Muss-Kriterien:**

* neue Aufgabe schnell erfassbar
* Listenansicht steht im Zentrum
* keine überladenen Projektmanagementfunktionen
* natürlichsprachige Eingabe prominent verfügbar
* Tastaturbedienung für zentrale Aktionen

Empfohlene Shortcuts:

| Shortcut | Funktion         |
| -------- | ---------------- |
| `n`      | neue Aufgabe     |
| `Enter`  | speichern        |
| `e`      | bearbeiten       |
| `d`      | erledigen        |
| `p`      | Priorität ändern |
| `/`      | Suche/Filter     |

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

### Project

```json
{
  "id": "project_garden",
  "name": "Garten",
  "description": "Aufgaben rund um Kleingarten und Hütte",
  "sort_order": 20,
  "archived": false,
  "created_at": "2026-05-20T20:00:00Z",
  "updated_at": "2026-05-20T20:00:00Z"
}
```

### Webhook

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
* löschen/archivieren
* anzeigen

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

* API
* Webhooks
* API-Key

### Web-UI

* eine Listenansicht
* Aufgabenformular
* Projektverwaltung
* Filter-/Sortierleiste

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
10. Eine wiederkehrende Aufgabe erzeugt beim Abschließen den nächsten Termin.
11. Natürlichsprachliche Eingaben wie `morgen`, `jeden Montag`, `alle 2 Wochen` werden korrekt interpretiert.
12. Aufgaben können nach Projekt gefiltert werden.
13. Aufgaben können nach Priorität sortiert werden.
14. Aufgaben können nach Deadline sortiert werden.
15. Aufgaben können über die API erstellt werden.
16. Aufgaben können über die API abgeschlossen werden.
17. Beim Erstellen einer Aufgabe wird ein Webhook ausgelöst.
18. Beim Abschließen einer Aufgabe wird ein Webhook ausgelöst.
19. Webhook-Payloads enthalten alle relevanten Aufgabendaten.
20. Webhook-Fehler werden protokolliert und erneut versucht.

---

## 10. Explizit gestrichene Anforderungen gegenüber Todoist

Folgende Todoist-nahe Funktionen werden bewusst nicht übernommen:

* Kollaboration
* Teams
* Gäste
* Kommentare
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

---

## 11. Kurzbeschreibung des Zielprodukts

Das Zielprodukt ist eine **schlanke, persönliche, webbasierte Todo-Anwendung mit API-first-Charakter**.

Der Kern ist nicht Kollaboration, sondern:

> Aufgaben sichtbar machen, wenn sie relevant werden; Deadlines separat verfolgen; wiederkehrende Aufgaben zuverlässig neu erzeugen; und alles über API und Webhooks automatisierbar machen.

Damit ist das Produkt eher ein **persönlicher Aufgaben-Kernel mit Web-UI und Automatisierungsschnittstellen** als eine klassische Team-Produktivitätsplattform.
