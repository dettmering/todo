# Lastenheft: Webbasierte Todo-Liste mit API und Webhooks

**Ziel:** Schlanke Aufgabenmanagement-Anwendung nach ausgewählten Prinzipien etablierter Task-Manager
**Scope:** Single-User, webbasiert, Listenansicht, API-first
**Nicht-Ziel:** Kollaboration, Mobile/Desktop-Apps, SSO, Analytics, Alarme, Board-/Kanban-Ansicht

> **Hinweis zum Dokumentstatus:** Dieses Lastenheft beschreibt die Anwendung im
> Produktivbetrieb und ist mit dem Implementierungsstand abgeglichen — es
> enthält keine Absichtserklärungen, sondern nur, was das System tatsächlich
> tut. Maßgeblich für Details ist der Code; siehe `CLAUDE.md` für die
> Architektur.

---

## 1. Zielsetzung

Ziel ist die Entwicklung einer schlanken webbasierten Aufgabenmanagement-Anwendung für persönliche Aufgabenverwaltung. Frontend: Ein modernes Web-Frontend. Backend: Backend in Go. Datenbank: SQLite.

Die Anwendung soll insbesondere folgende Fähigkeiten bieten:

* Aufgaben in Projekten organisieren
* Aufgaben in einer Listenansicht anzeigen
* Aufgaben mit Priorität versehen
* Aufgaben mit einem Erscheinungsdatum versehen
* Aufgaben mit einer Deadline versehen
* natürlichsprachige Eingabe für Termine unterstützen
* wiederkehrende Aufgaben verwalten
* eine API zur Automatisierung bereitstellen
* Webhooks zur Ereignisweitergabe bereitstellen

Die Anwendung soll kein etabliertes Produkt vollständig nachbauen, sondern nur die Funktionen abbilden, die für den konkreten Nutzungsfall relevant sind.

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
* Epics als langlaufende Initiativen zwischen Projekt und Aufgabe (siehe 4.11 und 5.15)
* Abwesenheiten und daraus abgeleitete Terminlogik (siehe 4.10 und 5.13)
* Kommentare und Aktivitätsprotokoll (siehe 4.8, 4.9 und 5.12)
* iCal-Feed, anonymisierte Share-Seite und MCP-Server (siehe 5.14)

### Out of Scope

* Mehrbenutzerfähigkeit
* Kollaboration
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

Da keine Kollaboration vorgesehen ist, gibt es nur eine Rolle.

| Rolle  | Beschreibung                                                                                |
| ------ | ------------------------------------------------------------------------------------------- |
| Nutzer | Verwaltet eigene Aufgaben, Projekte, Prioritäten, Termine, Deadlines und Automatisierungen. |

Die Anwendung selbst führt keine Benutzeranmeldung. Sie wird lokal, in einem geschützten Netzwerk oder hinter einem externen Schutzmechanismus betrieben (siehe 5.10).

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

Slug und Farbe sind gegenüber dem ursprünglichen Entwurf hinzugekommen und im
Datenmodell (Abschnitt 7) sowie der API enthalten.

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

### Bezugspunkt der Wiederholung

Es gibt zwei Arten wiederkehrender Aufgaben, die sich darin unterscheiden,
wovon aus der nächste Termin berechnet wird. Die Aufgabe muss zwischen beiden
wählen lassen (`recurrence_mode`):

* **Fester Termin** (`fixed`, Default) — der nächste Termin ergibt sich aus dem
  **Serientermin** (`recurrence_anchor`), einem vom Erscheinungsdatum
  getrennten Feld. Verspätetes Abhaken verschiebt die Serie nicht; verpasste
  Termine werden übersprungen. Für kalendarisch gebundene Aufgaben:
  Geburtstage, Miete, Quartalsabschluss.
* **Ab Erledigung** (`after_completion`) — der nächste Termin ergibt sich aus
  dem Tag des Abhakens; das Intervall startet bei jeder Erledigung neu. Für
  Wartungsintervalle: Kaffeemaschine reinigen, Filter wechseln.

Serientermin und Erscheinungsdatum müssen getrennt bleiben: Das Verschieben
einer einzelnen Instanz („dieses Jahr erst am Mittwoch") darf die Serie **nicht**
mitverschieben. Ein Geburtstag am 1.8., der auf den 19.8. verschoben und dort
abgehakt wird, muss im Folgejahr wieder am 1.8. erscheinen. Das Verlegen der
Serie selbst muss als eigene, ausdrückliche Handlung möglich sein.

„Ab Erledigung" ist auf Regeln ohne Kalender-Anker beschränkt (nur `FREQ` und
`INTERVAL`); „jeden Montag, gerechnet ab dem Abhaken" hat keine sinnvolle
Lesart und wird abgelehnt.

### Umsetzungsmodell

Zwei Modelle waren möglich: dieselbe Aufgabe auf das nächste Erscheinungsdatum
weiterschieben (**A**) oder sie abschließen und eine neue Instanz für den
nächsten Termin erzeugen (**B**).

Umgesetzt ist **Modell B**. Es kostet etwas mehr Daten, liefert dafür eine
echte Historie einzelner Erledigungen und ist API- und Webhook-freundlich: Jede
Instanz hat ihre eigene ID, ihr eigenes `completed_at` und löst eigene Events
aus (`task.completed` für die erledigte, `task.recurrence_generated` für die
neue).

---

## 4.7 Natürlichsprachige Datumseingabe

Das System muss natürlichsprachige Datumseingaben unterstützen.

### Datumsphrasen

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

### Wiederholungen

Die natürlichsprachige Eingabe deckt **Datumsphrasen** ab. Wiederholungen
werden **strukturiert** gesetzt, nicht aus freiem Text erkannt:

* im Frontend über ein Auswahlfeld mit Presets (Täglich, Werktäglich,
  Wöchentlich, Alle 2 Wochen, Monatlich, Alle 3 Monate, Jährlich, eigene RRULE)
  sowie einen Umschalter für den Bezugspunkt (Fester Termin / Ab Erledigung)
* über die API direkt als RRULE-String im Feld `recurrence_rule`, der
  Bezugspunkt in `recurrence_mode`

Siehe Abschnitt 4.6 für den unterstützten RRULE-Umfang.

### Anforderungen

* natürliche Sprache auf Deutsch
* der Parser unterscheidet Erscheinungsdatum und Deadline anhand des Feldes,
  in das die Phrase geschrieben wird
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

* Kommentare sind append-only — kein Bearbeiten, kein Löschen
* der Inhalt unterstützt Markdown; die Darstellung rendert das Markup
* Kommentare werden absteigend nach Aktualität (neueste zuerst) angezeigt
* das Setzen von Kommentaren muss auch automatisiert über die API möglich sein
* Kommentare können an Aufgaben in jedem Status gehängt werden (auch erledigte)

---

## 4.9 Aktivitätsprotokoll (Audit-Log)

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

## 4.10 Abwesenheit

Eine **Abwesenheit** ist ein zusammenhängender Zeitraum, in dem der Nutzer nicht
am Platz ist — Urlaub, Feiertag, Krankheit, Dienstreise oder Sonstiges.
Abwesenheiten hängen an keiner Aufgabe; sie beschreiben die Verfügbarkeit des
Nutzers und speisen die Terminlogik („nächster Arbeitstag"), die Warnungen beim
Anlegen von Aufgaben sowie die nach außen geteilten Verfügbarkeits-Sichten
(iCal-Feed, Share-Seite).

### Attribute einer Abwesenheit

* `id` — eindeutige ID (`absence_*`)
* `type` — einer von `vacation`, `holiday`, `sick`, `business_trip`, `other`
* `label` — optionaler freier Name (z. B. „Kreta"); rein informativ
* `start_date` / `end_date` — **inklusive** Kalendertage (`YYYY-MM-DD`)
* `created_at` / `updated_at` — Zeitstempel

### Typen und ihre Wirkung

| Typ             | Bedeutung   | Blockiert Arbeit? |
| --------------- | ----------- | ----------------- |
| `vacation`      | Urlaub      | ja                |
| `holiday`       | Feiertag    | ja                |
| `sick`          | Krank       | ja                |
| `other`         | Sonstiges (Default) | ja        |
| `business_trip` | Dienstreise | **nein** — zählt als regulärer Arbeitstag |

Die Dienstreise ist der bewusste Sonderfall: Sie ist eine Abwesenheit (erscheint
in Listen, Zeitleisten und geteilten Sichten), gilt aber der Terminlogik als
normaler Arbeitstag und verschiebt daher keine Aufgaben.

### Anforderungen

* Abwesenheiten sind anlegbar, bearbeitbar und löschbar (**harter Delete** —
  anders als Aufgaben und Projekte gibt es hier kein Soft-Delete/Archiv, weil
  eine gelöschte Abwesenheit keine Historie trägt)
* `end_date` muss `>= start_date` sein
* Abwesenheiten dürfen sich überlappen; die Auswertung nimmt jeweils den
  strengeren Fall (eine arbeitsblockierende Abwesenheit schlägt eine
  gleichzeitige Dienstreise)
* Datumsangaben werden hier **nur als ISO-Datum** entgegengenommen — die
  natürlichsprachige Eingabe aus 4.7 gilt für Aufgaben, nicht für Abwesenheiten
* das Anlegen und Pflegen muss auch automatisiert über die API möglich sein

### Abgeleitete Auswertungen

Die folgenden Regeln sind reine Domänenlogik (`internal/domain/absence.go`) und
werden von API, Frontend und den geteilten Sichten gemeinsam genutzt:

| Auswertung             | Regel                                                                 |
| ---------------------- | --------------------------------------------------------------------- |
| **Nächster Arbeitstag** | erster Tag **strikt nach** dem Stichtag, der weder Wochenende noch von einer arbeitsblockierenden Abwesenheit abgedeckt ist |
| **Aktuelle Verfügbarkeit** | „abwesend", wenn heute eine blockierende Abwesenheit greift; „auf Dienstreise", wenn nur eine Dienstreise greift; sonst „verfügbar" |
| **Rückkehrdatum**      | der nächste Arbeitstag — aber nur, solange heute tatsächlich eine blockierende Abwesenheit läuft; sonst gibt es nichts anzukündigen |
| **Ansichts-Auswahl**   | alle laufenden und künftigen Abwesenheiten plus die vergangenen **des laufenden Kalenderjahres** |
| **Zeitraum-Abfrage**   | alle Abwesenheiten, die ein inklusives Fenster `[von, bis]` **überlappen** — auch solche, die davor begonnen haben |

---

## 4.11 Epic

Ein **Epic** ist eine langlaufende Initiative — die Ebene zwischen Projekt und
Aufgabe. Ein Projekt ist ein Fach, das bleibt („Renovierung", „adesso"); ein
Epic ist ein Vorhaben darin, das anfängt, läuft und irgendwann fertig ist
(„Küche renovieren", „ISO-Audit 2027"). Ohne diese Ebene zerfällt ein solches
Vorhaben in zwanzig lose Aufgaben, und niemand sieht, wie weit es ist.

### Attribute eines Epics

| Attribut       | Beschreibung                                                              |
| -------------- | ------------------------------------------------------------------------- |
| `id`           | eindeutige ID (`epic_*`)                                                   |
| `title`        | Titel, Pflicht; wird getrimmt                                              |
| `slug`         | URL-sicherer Bezeichner, aus dem Titel abgeleitet, über alle Epics eindeutig, **stabil über Umbenennungen** |
| `description`  | optionaler Freitext (Markdown)                                             |
| `project_id`   | Pflicht; genau ein Projekt. **Die Inbox ist ausgeschlossen**               |
| `status`       | `active` („laufend") oder `completed` („beendet")                          |
| `start_date`   | optionales ISO-Datum; leer = offener Anfang                                |
| `end_date`     | optionales ISO-Datum; leer = offenes Ende                                  |
| `sort_order`   | manuelle Sortierung                                                        |
| `archived`     | ausgeblendet statt gelöscht (siehe unten)                                  |
| `created_at` / `updated_at` / `completed_at` | Zeitstempel                              |

### Anforderungen

* Ein Epic gehört zu **genau einem Projekt**. Die Inbox scheidet aus: sie ist
  der Eingangskorb für Unsortiertes, kein Vorhaben, das man über Monate führt.
* Start- und Enddatum sind **beide optional** und unabhängig voneinander. Ein
  gesetztes Ende darf nicht vor einem gesetzten Anfang liegen.
* Eine Aufgabe gehört **höchstens einem** Epic (`epic_id`, leer = keins) und
  liegt dann **immer im Projekt des Epics**. Wird einer Aufgabe ein Epic
  zugewiesen, wandert sie in dessen Projekt; wechselt ein Epic das Projekt,
  wandern alle seine Aufgaben mit. Wird umgekehrt nur das Projekt einer
  verknüpften Aufgabe geändert, **löst sich die Verknüpfung und die Aufgabe
  wird verschoben** — das zuletzt genannte Feld gewinnt, und ein Epic in einem
  anderen Projekt als seine Aufgabe ist kein lesbarer Zustand.
* Ein Epic lässt sich **nur beenden, wenn keine seiner Aufgaben mehr offen
  ist.** Erledigte und soft-gelöschte Aufgaben blockieren nicht — was
  weggeworfen wurde, ist keine offene Arbeit.
* Ein **beendetes oder archiviertes** Epic nimmt **keine neuen Aufgaben** an.
* Wird eine Aufgabe eines beendeten Epics wieder geöffnet, geht das **Epic
  automatisch zurück auf `active`**. Die Alternative — das Wiederöffnen zu
  verbieten — machte ein beendetes Epic zur Falle.
* Epics werden **archiviert, nicht gelöscht** (wie Projekte, anders als
  Abwesenheiten): Aufgaben behalten ihre `epic_id`, damit historische
  Beschriftungen weiter auflösen.
* Der Statuswechsel läuft über eigene Aktionen (`complete` / `reopen`), nicht
  über das allgemeine Teilupdate — so hat die Beendigungsregel genau eine Stelle.
* Datumsangaben werden **nur als ISO-Datum** entgegengenommen; die
  natürlichsprachige Eingabe aus 4.7 gilt für Aufgaben.
* Epics erzeugen **keine** eigenen Audit-Einträge; das Protokoll aus 4.9
  beschreibt Aufgaben. Die Zuordnung einer Aufgabe zu einem Epic wird dort
  allerdings als `epic_changed` festgehalten.

### Abgeleitete Auswertungen

| Auswertung        | Regel                                                                       |
| ----------------- | --------------------------------------------------------------------------- |
| **Fortschritt**   | Anzahl erledigter und offener Aufgaben des Epics; soft-gelöschte zählen in keiner der beiden Zahlen |
| **Beendbarkeit**  | keine Aufgabe des Epics hat den Status `offen`                              |
| **Laufende Epics**| weder archiviert noch beendet; das ist die Auswahl für Sidebar und Projektgruppierung |

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
| Projekt     | `/projects/<slug>`   | Aufgaben eines Projekts, unter den laufenden Epics des Projekts gruppiert |
| Epic        | `/epics/<slug>`      | Kopfbereich (Projekt, Status, Zeitraum, Fortschritt) und die Aufgaben eines Epics, erledigte darunter |
| Aktivität   | `/activity`          | Aktivitätsprotokoll (siehe 4.9), nach Tag gruppiert          |
| Abwesenheiten | `/absences`        | Abwesenheiten (siehe 4.10) als 12-Wochen-Zeitleiste + vollständige Liste, laufende/künftige oben, vergangene darunter |

Die gruppierten Ansichten rufen die API mit `scope=all` ab und wenden die
Sichtbarkeitslogik clientseitig an. Aufgabendetails sind direkt verlinkbar
(`/tasks/<id>`).

In der Ansicht **Demnächst** trägt jeder Kalenderwochen-Gruppenkopf zusätzlich
die **Abwesenheits-Badges** der Wochen, die von einer Abwesenheit berührt werden
— damit sichtbar wird, warum eine Woche dünn geplant ist. Das Badge zeigt nur
das Label der Abwesenheit (z. B. „Kreta"), der Typ steckt in der Einfärbung;
ohne Label steht dort der Typname. Der Tooltip nennt Typ und Label im Klartext.

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
* Standard (leer) — siehe unten

Aufgaben haben bewusst **keine manuelle Reihenfolge** und keinen eigenen
Sort-Modus „Projekt": Ein `sort_order` tragen nur Projekte, und Projekte sind
in der UI ohnehin eigene Ansichten.

Standardsortierung (Default `SortDefault`):

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

* Projekt (`project_id`)
* Status (`status` = open/completed/deleted)
* Sichtbarkeit (`scope=all` schaltet die Erscheinungsdatum-Logik ab)
* Erscheinungsdatum als Tag/Bereich/KW (`date` · `from`/`to` · `week=YYYY-Www`)
* Erledigungsdatum als Tag/Bereich/KW (`completed_date` · `completed_from`/`completed_to` · `completed_week`)

Beispiele:

* alle offenen Aufgaben
* alle Aufgaben im Projekt „Homelab“
* alle Aufgaben mit Erscheinungsdatum in einer bestimmten Kalenderwoche
* alle in einem Zeitraum erledigten Aufgaben

Eine komplexe Filter-Syntax, wie sie etablierte Task-Manager bieten, ist bewusst nicht vorgesehen.

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

### API-Endpunkte

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

GET    /api/epics
POST   /api/epics
GET    /api/epics/{id}
PATCH  /api/epics/{id}
POST   /api/epics/{id}/complete
POST   /api/epics/{id}/reopen
DELETE /api/epics/{id}

GET    /api/absences
POST   /api/absences
GET    /api/absences/{id}
PATCH  /api/absences/{id}
DELETE /api/absences/{id}
GET    /api/next-workday

POST   /api/parse
```

Zusätzlich, **nicht Teil der JSON-API**, sondern token-geschützte Sonderrouten
für Konsumenten außerhalb der Anwendung (siehe 5.14):

```http
GET    /api/absences.ics    iCal-Feed der Abwesenheiten   (ICAL_TOKEN)
GET    /api/absences.html   anonymisierte Share-Seite     (SHARE_TOKEN)
POST   /mcp                 MCP-Server (Streamable HTTP)  (MCP_TOKEN)
```

**Abweichungen gegenüber dem ursprünglichen Entwurf:**

* `POST /api/tasks/{id}/reopen`, `GET /api/tasks/{id}/audit` und `GET /api/audit`
  sind neu hinzugekommen.
* Epics (`/api/epics*`) sind ein ganzer Objektbereich, den der ursprüngliche
  Entwurf nicht kannte — siehe 4.11. `GET /api/tasks` kennt dazu den Filter
  `epic_id`, das Task-Objekt die Felder `epic_id` und `epic_title`.
* Abwesenheiten (`/api/absences*`) und die daraus abgeleitete
  Terminlogik (`/api/next-workday`) sind ein ganzer Objektbereich, den der
  ursprüngliche Entwurf nicht kannte — siehe 4.10.
* Eigene Endpunkte zum Lesen von „Prioritäten" und „Wiederholungsregeln" gibt es
  nicht — beide sind feste Wertebereiche (P1–P4 bzw. der RRULE-Subset aus 4.6)
  und werden nicht über die API ausgeliefert.
* **Webhook-Verwaltungsendpunkte (`/api/webhooks`) existieren nicht.** Webhooks
  sind reduziert über eine einzige konfigurierte Ziel-URL umgesetzt (siehe 5.11).

Hinweise zu Listenparametern: `GET /api/tasks` akzeptiert die Filter-/Sortier-
Parameter aus 5.6/5.7. `GET /api/projects` liefert standardmäßig nur aktive
Projekte; `include_archived=true` schließt archivierte ein (das Frontend nutzt
dies, damit historische Projektlabels weiter auflösen). `GET /api/absences`
kennt zwei Modi: ohne Parameter (bzw. mit `today=`) die Ansichts-Auswahl aus
4.10, mit `from=`/`to=` die überlappende Zeitraum-Abfrage. `GET /api/epics`
filtert über `project_id` und `status` und blendet archivierte Epics aus,
solange nicht `include_archived=true` gesetzt ist — dieselbe Regel wie bei
Projekten, und aus demselben Grund.

---

## 5.10 Zugriffsschutz

Die Anwendung kennt keine Benutzeranmeldung. Der Schutz ist zweistufig
aufgeteilt, weil die beiden Gruppen von Konsumenten unterschiedliche
Möglichkeiten haben.

**JSON-API und Weboberfläche** (`/`, `/api/tasks`, `/api/projects`,
`/api/absences`, `/api/audit`, `/api/parse`) tragen keine eigene
Authentifizierung. Sie liegen hinter der **BasicAuth des Reverse-Proxys**; die
Anwendung wird nur lokal, im geschützten Netz oder hinter diesem Proxy
betrieben.

**Die Sonderrouten aus 5.14** (`/api/absences.ics`, `/api/absences.html`,
`/mcp`) müssen von dieser Proxy-Auth ausgenommen sein — Kalender-Clients,
MCP-Clients und Kollegen können sich nicht interaktiv anmelden. Dort ist ein
**Token der einzige Schutz**, und für alle drei gilt dasselbe Muster:

* je Route eine **eigene** Umgebungsvariable (`ICAL_TOKEN`, `SHARE_TOKEN`,
  `MCP_TOKEN`), damit ein Zugang einzeln widerrufbar ist und ein geteilter Link
  nicht mehr freischaltet als gewollt
* **ist die Variable nicht gesetzt, existiert die Route nicht** (404) — eine
  Fehlkonfiguration kann die Route also nie ungeschützt ausliefern
* falscher oder fehlender Token ⇒ 401; der Vergleich läuft in konstanter Zeit
* der MCP-Endpunkt nimmt den Token zusätzlich als
  `Authorization: Bearer <token>`-Header entgegen

### Out of Scope

* OAuth
* User Login
* SSO
* Rollenmodell
* Multi-User-Tokens

---

## 5.11 Webhooks

### LA-F-010 Webhooks bereitstellen

Das System muss Ereignisse an einen externen Empfänger weitergeben.

Umgesetzt ist bewusst die **einfachste tragfähige Form**: Ist die
Umgebungsvariable `WEBHOOK_URL` gesetzt, wird **jedes** Event des internen
Event-Bus als JSON-POST an diese eine URL zugestellt. Für einen
Single-User-Betrieb mit genau einem Empfänger (typischerweise eine
Automatisierungs-Plattform, die selbst filtert) trägt das — eine
Verwaltungs-API mit mehreren Zielen, Event-Filtern und Secrets wäre Aufwand
ohne Nutzen.

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
* Kommentar erstellt
* Epic erstellt, aktualisiert, beendet, archiviert
* Abwesenheit erstellt, aktualisiert, gelöscht

### Event-Typen

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
epic.created
epic.updated
epic.completed
epic.deleted
absence.created
absence.updated
absence.deleted
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

### Zustellung

| Eigenschaft   | Verhalten                                                                 |
| ------------- | ------------------------------------------------------------------------- |
| Ziel          | genau eine URL, konfiguriert über `WEBHOOK_URL`; ohne die Variable ist der Versand aus |
| Auswahl       | keine — es geht **jedes** Event des Busses hinaus, der Empfänger filtert   |
| Ausführung    | asynchron auf der Bus-Goroutine; eine HTTP-Anfrage wartet nie auf die Zustellung |
| Timeout       | fest 10 s                                                                  |
| Wiederholung  | keine — ein Versuch je Event                                               |
| Fehlerbehandlung | Transportfehler und Antworten ab Status 400 landen im Server-Log         |
| Absicherung   | keine Signatur, kein Secret — der Empfänger wird über eine nicht öffentliche URL adressiert |

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
* Kommentare sind append-only — kein Bearbeiten, kein Löschen

---

## 5.13 Abwesenheiten verwalten

### LA-F-012 Abwesenheiten pflegen und auswerten

Das System muss erlauben, Abwesenheitszeiträume (siehe 4.10) zu pflegen und die
Terminlogik daran auszurichten.

**Muss-Kriterien:**

* eine eigene Ansicht `/absences` listet die Abwesenheiten: laufende und
  künftige zuerst (nächste zuerst), vergangene darunter (jüngste zuerst)
* je Eintrag werden Typ-Badge, Label, Zeitraum sowie **Dauer und zeitlicher
  Abstand** gezeigt („12 Tage · beginnt in 3 Wochen", „5 Tage · noch 2 Tage",
  „endet heute") — genau das, was man sich aus zwei nackten Datumsangaben sonst
  selbst ausrechnet; bei vergangenen Einträgen entfällt der Abstand
* der aktuell laufende Eintrag ist hervorgehoben
* über der Liste steht eine **Zeitleiste über die nächsten 12 Wochen**: ein
  farbiger Balken je Abwesenheit auf einer Schiene mit Wochenstrichen,
  Kalenderwochen darüber, Monatslabels darunter und einer Markierung für
  heute. Die Wochenzahlen sind ISO-8601 und stehen ohne Präfix „KW" (über
  einer Kalenderschiene ist die Zahl eindeutig); die laufende Woche ist
  hervorgehoben. Das Fenster beginnt am Montag
  der laufenden Woche, Balken werden an den Fensterrändern abgeschnitten,
  Abwesenheiten außerhalb des Fensters entfallen (sie bleiben in der Liste —
  die interne Ansicht dient der Pflege und zeigt alles).
  Ein Hover auf einer Listenzeile hebt den zugehörigen Balken hervor, sodass
  Leiste und Liste als eine Darstellung lesbar sind
* Anlegen, Bearbeiten und Löschen laufen über einen Dialog
* **Warnung statt Verbot:** Fällt das Erscheinungsdatum oder die Deadline einer
  Aufgabe in eine hinterlegte Abwesenheit, zeigt die Eingabe (Quick-Add wie
  Aufgabendialog) einen nicht blockierenden Hinweis mit Typ und Label. Alle
  Typen lösen ihn aus, auch die Dienstreise
* der Quick-Reschedule-Knopf **„Nächster Arbeitstag"** im Aufgabendialog setzt
  das Erscheinungsdatum auf den nächsten Arbeitstag nach der Regel aus 4.10
  (überspringt Wochenenden und blockierende Abwesenheiten)

**Verhalten:**

* die Terminlogik ist **serverseitig** die Quelle der Wahrheit
  (`GET /api/next-workday`), damit Frontend, API-Clients und MCP-Clients
  dieselbe Antwort bekommen
* Anlegen, Ändern und Löschen lösen die Events `absence.created`,
  `absence.updated` und `absence.deleted` aus (siehe 5.11)
* Abwesenheiten erzeugen **keine** Audit-Einträge — das Protokoll aus 4.9
  beschreibt Aufgaben

---

## 5.14 Geteilte Sichten und Integrationen

### LA-F-013 Verfügbarkeit teilen und die App fernsteuerbar machen

Drei Routen liegen außerhalb der JSON-API und richten sich an Konsumenten
außerhalb der Anwendung. Alle drei sind nach demselben Muster token-geschützt
(siehe 5.10).

| Route | Zweck | Token |
| --- | --- | --- |
| `GET /api/absences.ics` | Abwesenheiten als abonnierbarer iCal-Feed für Kalender-Clients | `ICAL_TOKEN` |
| `GET /api/absences.html` | anonymisierte Verfügbarkeits-Seite zum Teilen mit Kollegen | `SHARE_TOKEN` |
| `/mcp` | MCP-Server (Streamable HTTP), damit LLM-Clients die App als Werkzeuge nutzen | `MCP_TOKEN` |

**iCal-Feed:**

* jede Abwesenheit wird ein ganztägiges `VEVENT` mit stabiler `UID`, sodass ein
  erneuter Abruf Termine aktualisiert statt sie zu duplizieren
* blockierende Abwesenheiten werden als **„Out of Office"** markiert, die
  Dienstreise als „woanders arbeitend" — sie bleibt terminierbar
* Kalendername und Refresh-Intervall werden mitgeliefert; der Kalendername
  folgt `SHARE_NAME`

**Share-Seite:**

* **anonymisiert:** nur Zeitraum und ein grober Zustand („Away" /
  „Business Trip") verlassen den Server. Grund (`type` im Detail) und `label`
  werden nie ausgeliefert
* die Seite wird **serverseitig** als self-contained HTML gerendert (kein
  JS-Fetch), damit die sensiblen Felder gar nicht erst im Browser landen
* sie zeigt: die aktuelle Verfügbarkeit als Satz, bei Abwesenheit das
  Rückkehrdatum („Back in 4 days on Monday, 3 August 2026"), eine
  12-Wochen-Zeitleiste, die Liste der laufenden und künftigen Zeiträume, eine
  Legende und das Alter der letzten Änderung
* **Zeitleiste und Liste teilen sich dasselbe 12-Wochen-Fenster** (Montag der
  laufenden Woche + 12 Wochen): was später beginnt, erscheint weder als Balken
  noch als Listenzeile. Die Seite beantwortet die absehbare Verfügbarkeit, sie
  gibt keine Jahresvorausschau preis
* der Token steht in der URL, also setzt die Seite `Referrer-Policy:
  no-referrer`, damit er nicht über den Referer abfließt

**MCP-Server:**

* eingebettet in dasselbe Binary und auf **derselben Service-Schicht** wie die
  REST-API — gleiche Validierung, ID-Vergabe, Zeitstempel und Events
* deckt Aufgaben, Projekte, Epics, Abwesenheiten, den nächsten Arbeitstag und
  den NL-Parser ab

Die vollständige technische Beschreibung (Felder, Parameter, Clientanbindung)
steht in `README.md`; dieser Abschnitt hält nur die fachliche Anforderung fest.

---

## 5.15 Epics verwalten

### LA-F-014 Langlaufende Initiativen führen

Das System muss erlauben, Epics (siehe 4.11) zu pflegen und Aufgaben darunter
zu bündeln.

**Muss-Kriterien:**

* Epics sind anlegbar, bearbeitbar, beendbar, wieder öffenbar und archivierbar
  — über Oberfläche, REST-API und MCP gleichermaßen
* die Sidebar führt einen eigenen Abschnitt **„Epics"** mit den **laufenden**
  Epics, je Eintrag ein Punkt in der Farbe seines Projekts, der Titel und die
  Zahl der offenen Aufgaben. **Beendete und archivierte Epics stehen nicht in
  der Sidebar** — sie bleiben über ihre URL und die API erreichbar. Eine Liste,
  die nur wächst, ertränkt die Einträge, auf die es ankommt
* Klick öffnet das Epic, Doppelklick den Bearbeiten-Dialog — dieselbe Geste wie
  bei Projekten
* die **Projektansicht** stellt die laufenden Epics des Projekts **über** die
  Aufgaben und gruppiert die zugeordneten Aufgaben **unter** ihrem Epic;
  Aufgaben ohne Epic stehen darunter unter „Ohne Epic". Ein laufendes Epic ohne
  offene Aufgabe bleibt sichtbar — „hier ist gerade nichts zu tun" ist eine
  Aussage. Hat ein Projekt gar keine Epics, bleibt seine Liste unverändert flach
* die **Epic-Ansicht** (`/epics/<slug>`) zeigt einen Kopfbereich mit Projekt,
  Status, Zeitraum und Fortschritt („3 / 8 erledigt"), darunter die offenen
  Aufgaben und, davon abgesetzt, die bereits erledigten — eine Initiative liest
  sich ebenso sehr an dem, was hinter ihr liegt
* der Quick-Add legt in der Epic-Ansicht direkt **in** dieses Epic an; ein
  ausdrückliches `#projekt`-Kürzel schlägt das
* der Aufgabendialog hat ein Feld **Epic** mit den laufenden Epics des
  gewählten Projekts und einem ausdrücklichen „kein Epic"; die Wahl eines Epics
  setzt das Projektfeld passend nach
* in den Ansichten Heute, Demnächst und Inbox trägt eine zugeordnete Aufgabe
  den Epic-Namen als Marke in ihrer Datumszeile — in Projekt- und Epic-Ansicht
  nicht, dort steht das Epic ohnehin darüber
* der Versuch, ein Epic mit offenen Aufgaben zu beenden, wird mit einer
  **sichtbaren Meldung an Ort und Stelle** abgelehnt, nicht stillschweigend

**Verhalten:**

* die Beendigungsregel ist **serverseitig** die Quelle der Wahrheit, damit
  Frontend, API- und MCP-Clients dieselbe Antwort bekommen
* Anlegen, Ändern, Beenden und Archivieren lösen `epic.created`,
  `epic.updated`, `epic.completed` und `epic.deleted` aus (siehe 5.11)
* die Zuordnung einer Aufgabe zu einem Epic erscheint im Aktivitätsprotokoll
  als `epic_changed` (siehe 4.9); das Epic selbst führt kein eigenes Protokoll

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
* **Dark Mode:** folgt automatisch der Systemeinstellung via
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

| Shortcut | Funktion         |
| -------- | ---------------- |
| `n`      | neue Aufgabe (Quick-Add fokussieren) |
| `Enter`  | speichern (Quick-Add, Dialoge), `Strg/Cmd+Enter` für Kommentare |
| `Escape` | Dialog/Sidebar schließen |

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
  "epic_id": "",
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
(aufgelöster Projektname), `epic_title` (aufgelöster Epic-Titel),
`recurrence_label` (deutsche Klartext-Beschreibung der Wiederholung),
`comment_count` sowie die eingebetteten `comments`.

### Epic

```json
{
  "id": "epic_123",
  "title": "Küche renovieren",
  "slug": "kueche-renovieren",
  "description": "",
  "project_id": "project_renovierung",
  "status": "active",
  "start_date": "2026-09-01",
  "end_date": "2027-03-31",
  "sort_order": 0,
  "archived": false,
  "created_at": "2026-09-04T09:57:13Z",
  "updated_at": "2026-09-04T09:57:13Z",
  "completed_at": null
}
```

Die API-Antwort (`epicDTO`) reichert an: `project_name` sowie `open_tasks` und
`done_tasks` — berechnet, nicht gespeichert.

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

### Absence

```json
{
  "id": "absence_123",
  "type": "vacation",
  "label": "Kreta",
  "start_date": "2026-07-10",
  "end_date": "2026-07-24",
  "created_at": "2026-06-08T10:15:00Z",
  "updated_at": "2026-06-08T10:15:00Z"
}
```

`start_date` und `end_date` sind inklusive. Abwesenheiten hängen an keiner
Aufgabe und tragen keinen Fremdschlüssel.

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

Webhooks haben **kein** persistiertes Objekt: Das Ziel ist eine einzelne, per
`WEBHOOK_URL` konfigurierte Adresse (siehe 5.11).

---

## 8. Funktionsumfang

Die Anwendung umfasst:

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

### Epics

* erstellen
* bearbeiten
* beenden und wieder öffnen
* archivieren
* Aufgaben zuordnen (die Aufgabe wandert dabei ins Projekt des Epics)
* Fortschritt ablesen

### Terminlogik

* Erscheinungsdatum
* Deadline
* wiederkehrende Aufgaben
* natürlichsprachige Datumseingabe
* Abwesenheiten (siehe 4.10) inkl. „nächster Arbeitstag" und Warnung, wenn eine
  Aufgabe in eine Abwesenheit fällt

### Organisation

* Prioritäten
* Listenansicht
* einfache Sortierung
* einfache Filter

### Automatisierung

* REST-API
* Webhooks (eine Ziel-URL, siehe 5.11)
* iCal-Feed, anonymisierte Share-Seite und MCP-Server (jeweils token-geschützt —
  siehe 5.14)

### Web-UI

* Ansichten Heute / Demnächst / Inbox / Projekt / Aktivität / Abwesenheiten
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
11. Natürlichsprachliche Datumseingaben wie `morgen`, `Freitag`, `in 3 Tagen`, `KW32` werden korrekt interpretiert; Wiederholungen werden über Auswahlfeld bzw. RRULE gesetzt (siehe 4.7).
12. Aufgaben können nach Projekt gefiltert werden.
13. Aufgaben können nach Priorität sortiert werden.
14. Aufgaben können nach Deadline sortiert werden.
15. Aufgaben können über die API erstellt werden.
16. Aufgaben können über die API abgeschlossen werden.
17. Beim Erstellen einer Aufgabe wird ein Webhook ausgelöst.
18. Beim Abschließen einer Aufgabe wird ein Webhook ausgelöst.
19. Webhook-Payloads enthalten alle relevanten Aufgabendaten.
20. Fehlgeschlagene Webhook-Zustellungen werden im Server-Log protokolliert.
21. Eine Aufgabe kann über die Weboberfläche kommentiert werden; Kommentare werden mit Markdown gerendert und neueste zuerst angezeigt.
22. Kommentare können über die API gelesen und erstellt werden.
23. In der Listenansicht erscheint bei kommentierten Aufgaben eine Sprechblase mit der Kommentaranzahl.
24. Eine abgeschlossene Aufgabe kann über UI und API wieder geöffnet werden.
25. Zustandsänderungen einer Aufgabe (Erstellen, Abschließen, Wieder­öffnen, Löschen, Datums- und Projektänderungen) werden im Aktivitätsprotokoll festgehalten und sind nach Aufgabe und Zeitraum abrufbar.
26. Die Oberfläche folgt automatisch dem hellen/dunklen Systemfarbschema.
27. Eine Abwesenheit kann über UI und API angelegt, bearbeitet und gelöscht werden; die Ansicht „Abwesenheiten" zeigt sie als 12-Wochen-Zeitleiste und Liste mit Dauer und Abstand.
28. Der nächste Arbeitstag überspringt Wochenenden und arbeitsblockierende Abwesenheiten, behandelt eine Dienstreise aber als normalen Arbeitstag.
29. Fällt das Datum einer Aufgabe in eine Abwesenheit, weist die Eingabe nicht blockierend darauf hin.
30. Der iCal-Feed und die Share-Seite sind ohne gesetzten Token nicht erreichbar (404) und liefern mit falschem Token 401; die Share-Seite gibt weder Typ noch Label preis.
31. Ein Epic kann über UI und API angelegt, bearbeitet, beendet, wieder geöffnet und archiviert werden; es gehört immer zu genau einem Projekt und nie zur Inbox.
32. Ein Epic lässt sich nicht beenden, solange eine seiner Aufgaben offen ist; die Oberfläche zeigt die Begründung an Ort und Stelle.
33. Wird eine Aufgabe eines beendeten Epics wieder geöffnet, geht das Epic zurück auf „laufend".
34. Eine Aufgabe mit Epic liegt immer im Projekt ihres Epics — beim Zuordnen wandert sie mit; wird sie in ein anderes Projekt verschoben, löst sich die Epic-Verknüpfung.
35. Die Sidebar führt die laufenden Epics; die Projektansicht gruppiert die Aufgaben unter ihnen, die Epic-Ansicht zeigt Zeitraum, Status und Fortschritt.

---

## 10. Explizit gestrichene Anforderungen gegenüber etablierten Task-Managern

Folgende in solchen Produkten übliche Funktionen werden bewusst nicht übernommen:

* Kollaboration
* Teams
* Gäste
* Datei-Uploads
* Zuweisungen
* Labels
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
> Kollaborations-Kommentare etablierter Task-Manager (Mentions, Zuweisungen, Benachrichtigungen).

---

## 11. Kurzbeschreibung des Produkts

Die Anwendung ist eine **schlanke, persönliche, webbasierte Todo-Anwendung mit API-first-Charakter**.

Der Kern ist nicht Kollaboration, sondern:

> Aufgaben sichtbar machen, wenn sie relevant werden; Deadlines separat verfolgen; wiederkehrende Aufgaben zuverlässig neu erzeugen; Abwesenheiten in die Terminlogik einrechnen; und alles über API, Webhooks und MCP automatisierbar machen.

Damit ist das Produkt eher ein **persönlicher Aufgaben-Kernel mit Web-UI und Automatisierungsschnittstellen** als eine klassische Team-Produktivitätsplattform.
