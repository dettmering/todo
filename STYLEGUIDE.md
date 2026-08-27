# Styleguide

Dieses Dokument hält fest, wie unser Frontend aussieht und warum. Es ist
**maßgeblich**: eine Farbe, ein Radius oder eine Schriftrolle, die hier nicht
steht, gehört nicht ins Stylesheet, bis sie hier steht.

> **Gestalterische Grundlage: die Weboberfläche von `md2pdf`**
> (`../md2pdf`, `src/webui.js` und `assets/template.typ`). Von dort kommen die
> beiden Schriften (IBM Plex Sans / IBM Plex Mono), die Druckerei-Palette
> (Papier, Tusche, Graphit, Prozess-Cyan, Schnittmarken-Rot) und die Haltung:
> **ein lautes Element, alles andere leise.** Übernommen ist nicht das Layout —
> md2pdf konfiguriert einen Satzspiegel, wir führen eine Aufgabenliste —,
> sondern das Vokabular.

> **Vorgänger:** Bis Commit `2215809` orientierte sich diese Datei an einem
> etablierten Task-Manager-Webclient („Moonstone"-Theme, Salbei-Akzent). Diese
> Fassung ersetzt sie vollständig. Wer alte Werte sucht, findet sie in der
> Git-Historie, nicht hier.

---

## 1. Die drei Regeln

Alles Weitere folgt aus diesen drei Sätzen. Sie stehen als Kommentar auch am
Kopf von `internal/web/static/style.css`.

1. **Zwei Schriften, zwei Aufgaben.** IBM Plex Sans setzt alles, was man
   *liest*. IBM Plex Mono setzt alles, was man *abliest* — Zahlen, Daten,
   Kürzel, Rubriken, Marginalien. Mono ist immer klein, versal, gesperrt und
   hat Tabellenziffern.
2. **Zwei Signalfarben, sonst Graustufen.** Prozess-Cyan heißt „jetzt"
   (heute fällig, laufende Woche, aktive Auswahl, Marke).
   Schnittmarken-Rot heißt „überfällig / löschen". Priorität ist **keine
   Ampel, sondern Farbdichte**: Rot, Ocker, Graphit, blass.
3. **Flächen sind kantig, Marken sind rund.** Was man *drückt* oder was etwas
   *benennt*, ist eine Pille (`--radius-pill`). Was etwas *trägt* oder
   *aufnimmt* — Karte, Dialog, Eingabefeld, Menü —, hat eine Kante
   (`--radius`, 4 px). Kreise bleiben Kreise: Checkbox, Projektpunkt,
   Farbwähler.

---

## 2. Farbpalette

Alle Farben stehen als Token in `:root` und werden im
`@media (prefers-color-scheme: dark)`-Block vollständig neu belegt. **Ein hart
notierter Hex überlebt den Themenwechsel nicht** — Farben gehören immer in ein
Token.

### 2.1 Flächen

| Rolle | Token | Hell | Dunkel |
| --- | --- | --- | --- |
| Papier (Inhaltsspalte, Dialog) | `--main-bg` | `#ffffff` | `#12151a` |
| Grund (abgesetzte Flächen, Quick-Add, Modal-Spalte) | `--surface` | `#fbfaf7` | `#191d23` |
| Sidebar | `--sidebar-bg` | `#f3f1ec` | `#0e1116` |
| Hover | `--hover` | `rgba(21,23,28,.05)` | `rgba(232,233,236,.07)` |

Die Neutraltöne im Hellmodus sind **warm** (Papierweiß, nicht Reinweiß) — das
ist der Grund, auf dem in md2pdf das Blatt liegt. Im Dunkelmodus ist die
Sidebar **dunkler** als der Inhalt, nicht heller: der Inhalt bleibt die Fläche,
auf die man schaut.

### 2.2 Schrift und Linien

| Rolle | Token | Hell | Dunkel |
| --- | --- | --- | --- |
| Tusche (Titel, Aufgabentitel) | `--text` | `#15171c` | `#e8e9ec` |
| Graphit (Fließtext, Nav idle) | `--text-secondary` | `#666c76` | `#9aa1ab` |
| Marginalie (Daten, Zähler, Rubriken) | `--text-muted` | `#8a9099` | `#7b838e` |
| Haarstrich | `--border` | `rgba(21,23,28,.12)` | `rgba(232,233,236,.14)` |
| Kräftiger Strich (Rubrikzeile, Feldrand) | `--border-strong` | `rgba(21,23,28,.22)` | `rgba(232,233,236,.26)` |

Linien sind **Alphawerte, keine Hex-Grautöne**. Dadurch stimmen sie über
Papier, Grund und Sidebar hinweg, ohne dass es drei Strichfarben braucht.

### 2.3 Akzent — die Prozessfarbe

| Zweck | Token | Hell | Dunkel |
| --- | --- | --- | --- |
| Akzent | `--accent` | `#0b6e88` | `#56c5e2` |
| Akzent Hover | `--accent-hover` | `#095a6f` | `#7ad4ec` |
| Akzentlinie (Registermarke, Leiter) | `--accent-line` | `#0e88a6` | `#3fb4d4` |
| Akzent weich (Auswahl in der Sidebar) | `--accent-soft` | `#e1edf0` | `#16323c` |
| Schrift auf Akzentfläche | `--on-accent` | `#ffffff` | `#10141a` |

In md2pdf markiert Cyan den Bereich, der **druckt** — das, was gerade zählt.
Bei uns heißt es dasselbe: „jetzt". Es trägt daher **beide** Rollen, Marke und
Fälligkeit, und das ist Absicht, keine Kollision: die laufende Woche, das
heutige Datum, das ausgewählte Nav-Item und das Logo sind alle „jetzt".

### 2.4 Priorität als Farbdichte

| Stufe | Token | Hell | Dunkel | Lesart |
| --- | --- | --- | --- | --- |
| **P1** | `--p1` | `#b5342a` | `#e86a5e` | Schnittmarken-Rot |
| **P2** | `--p2` | `#a2620f` | `#d79a4a` | Ocker |
| **P3** | `--p3` | `#5b6472` | `#98a1ae` | Graphit |
| **P4** | `--p4` | `#a9aeb6` | `#5c646f` | blass |

Die Leiter ist bewusst **kein Regenbogen**. Eine vierstufige Skala aus vier
Farbtönen zwingt zum Auswendiglernen; eine Skala aus einer Signalfarbe und
abnehmender Tinte liest man ohne Legende. Blau ist damit als Prioritätsfarbe
frei geworden — es steht jetzt der Akzentfarbe und den Projektfarben zur
Verfügung.

Die Farben tragen den Rand der hohlen Checkbox, mit einem 10 %-Tint als
Innenfläche (`color-mix(in srgb, var(--pN) 10%, transparent)`, damit der Tint
im Dunkelmodus automatisch mitzieht).

> **Sonderfall Beschriftung:** Als Text auf einem Schalter (`.td-prio.p4`) wäre
> `--p4` nicht mehr lesbar. Dort steht `--text-secondary`. Blass ist P4 überall
> dort, wo es als *Fläche* auftritt.

### 2.5 Fälligkeit

| Status | Token | Hell | Dunkel |
| --- | --- | --- | --- |
| Überfällig | `--overdue` | `#b5342a` | `#e86a5e` |
| Heute | `--due-today` | `#0b6e88` | `#56c5e2` |
| Morgen | `--due-tomorrow` | `#a2620f` | `#d79a4a` |
| Später | `--due-future` | `#666c76` | `#9aa1ab` |
| Deadline (Flagge) | `--deadline-color` | `#a2620f` | `#d79a4a` |

Dieselbe Leiter wie bei der Priorität, nur mit Cyan an der Stelle von „heute".
Überfällig steht zusätzlich in Mono **700**.

### 2.6 Abwesenheitsfarben

Vier **Kategorien**, keine Rangfolge — der einzige Ort im Produkt, an dem
mehrere Farbtöne nebeneinander stehen dürfen, weil die Zeitleiste sie
unterscheidbar braucht. Sie sind auf dieselbe Helligkeit und Sättigung gedämpft
wie der Rest.

| Typ | Hell (`-bg` / `-fg`) | Dunkel (`-bg` / `-fg`) |
| --- | --- | --- |
| `vacation` | `#e4efe8` / `#2c7351` | `#16321f` / `#6ecb8c` |
| `holiday` | `#ebe6f4` / `#6a4ca8` | `#251f36` / `#b49ae8` |
| `business_trip` | `#e5e8f4` / `#3a4fa0` | `#1b2340` / `#8b9ce8` |
| `sick` | `#f7e4e2` / `#b5342a` | `#3a201d` / `#e86a5e` |
| `other` | `--surface` / `--text-secondary` | dito (Token-basiert) |

`sick` teilt sich die Farbe mit `--p1`: „krank" und „überfällig" sind beide
Zustände, die etwas anhalten.

---

## 3. Typografie

Beide Schnitte liegen als Subset (Latin + Latin-Ext) im WOFF2-Format unter
`internal/web/static/fonts/` und werden über `go:embed` mit ausgeliefert —
kein CDN, keine Netzwerkabhängigkeit. Die TTF-Quellen stammen aus
`../md2pdf/assets/fonts` (IBM Plex, SIL Open Font License; `LICENSE.txt` liegt
im selben Ordner).

| Datei | Familie | Schnitt |
| --- | --- | --- |
| `plex-sans-400.woff2` | `--font-sans` | Regular |
| `plex-sans-400i.woff2` | `--font-sans` | Regular kursiv |
| `plex-sans-500.woff2` | `--font-sans` | Medium |
| `plex-sans-600.woff2` | `--font-sans` | SemiBold |
| `plex-mono-400.woff2` | `--font-mono` | Regular |
| `plex-mono-700.woff2` | `--font-mono` | Bold |

**Es gibt kein 700 in der Sans.** `strong`/`b` sind global auf `600` gesetzt,
damit gerendertes Markdown keinen synthetischen Fettschnitt erzeugt.

### 3.1 Die Rollen

| Rolle | Wert | Verwendung |
| --- | --- | --- |
| Body | Sans 400 / `14px` / `1.5` / `--text-secondary` | alles Gelesene |
| Seitentitel (H1) | Sans 600 / `28px` / `-0.022em` | Ansichtsüberschrift |
| Dialogtitel (H2) | Sans 600 / `18px` / `-0.018em` | |
| Aufgabentitel im Modal | Sans 600 / `21px` / `-0.018em` | |
| Aufgabentitel in der Liste | Sans 400 / `14px` / `1.45` / `--text` | |
| **Marginalie** | Mono 400 / `11px` / `0.13em` / versal / `--text-muted` | Rubriken, Feldrubriken, Zähler, Rubrikzeilen |
| **Ablesewert** | Mono 400 / `11px` / `0.04–0.07em` / `tabular-nums` | Daten, Uhrzeiten, Zeiträume, Kalenderwochen |

Die Utility-Klassen `.mono` und `.eyebrow` in `style.css` fassen die letzten
beiden Zeilen zusammen; Komponenten dürfen die Werte auch direkt setzen, wenn
sie zusätzlich etwas anderes brauchen.

**Faustregel für die Entscheidung Sans/Mono:** Steht dort ein Wort, das jemand
frei gewählt hat (Aufgabentitel, Projektname, Abwesenheitslabel wie „Kreta")?
→ Sans. Steht dort etwas, das das System ausrechnet oder benennt (Datum,
Anzahl, „KW 35", „P2", „MO")? → Mono.

Antialiasing ist explizit aktiviert (`-webkit-font-smoothing: antialiased;
-moz-osx-font-smoothing: grayscale`).

---

## 4. Spacing-Skala

Unverändert eine 4er-Skala: `4 / 8 / 12 / 16 / 24 / 32 px`, nicht zu Tokens
erhoben — bei unserer Komponentenmenge wäre das Overengineering.

Eine Ausnahme mit Bedeutung: der Abstand **über** einer Rubrikzeile ist
`30px` (Aufgabenliste) bzw. `22px` (Aktivität) und damit deutlich größer als
der Abstand darunter (`7px` bis zur Linie). Die Rubrik gehört zu dem, was
folgt, nicht zu dem, was davor stand.

---

## 5. Radien und Schatten

| Element | Radius |
| --- | --- |
| Karte, Dialog, Eingabefeld, Textarea, Select, Menü, Codeblock | `var(--radius)` = `4px` |
| **Jeder Button**, Chip, Badge, Pille, Nav-Item, Icon-Button, Zeitleistenbalken | `var(--radius-pill)` = `999px` |
| Checkbox, Projektpunkt, Farbwähler, „Heute"-Punkt | `50%` |

Die Trennung ist die dritte Grundregel aus §1 und der auffälligste Unterschied
zur Vorgängerfassung, die alles zwischen 4 und 12 px gerundet hat. Sie ist
funktional: Wenn ein Element rund ist, kann man es drücken oder es benennt
etwas.

Schatten sind zurückhaltend und nur an zwei Stellen:

* Dialoge: `0 16px 48px var(--shadow-dialog)`
* Popup-Menü (Projektauswahl im Quick-Add): `0 8px 24px var(--shadow-menu)`

Aufgabenzeilen, Karten und Buttons haben **keinen** Schatten — sie sitzen auf
dem Papier, sie schweben nicht darüber.

---

## 6. Layoutgerüst

```
┌─────────────┬──────────────────────────────────────────────┐
│             │  Heute                                       │
│             │  DONNERSTAG, 27. AUGUST                      │
│  Sidebar    │  ─────────────────────────────────────────   │
│  280 px     │  ┌ Quick-Add ────────────────────────────┐   │
│  --sidebar- │  └───────────────────────────────────────┘   │
│    bg       │  ▪ KW 35  24.08.–30.08. ─────────────────  5 │
│             │  ◯ Aufgabe                        Projekt •  │
│             │  ◯ Aufgabe                        Projekt •  │
│             │    KW 36  31.08.–06.09. ──────  [Messe]   3  │
└─────────────┴──────────────────────────────────────────────┘
```

* Shell: zweispaltiges Grid `280px 1fr`, volle Höhe, beide Spalten mit eigenem
  Scrollbereich. Tablet hochkant: `242px`. Unter `721px` wird die Sidebar zum
  Drawer.
* Inhaltsspalte `max-width: 680px`, zentriert über
  `padding: 32px max(60px, (100% - 680px) / 2) 96px`.
* Der Kopf trägt unten einen `--border-strong`-Haarstrich; die Zeile darunter
  („DONNERSTAG, 27. AUGUST") ist eine Marginalie. In der Ansicht „Heute"
  wechselt der Strich auf `--accent-line` und die Marginalie trägt die
  Registermarke (7.1).

---

## 7. Komponenten

### 7.1 Die Rubrikzeile — das eine laute Element

```
▪ KW 35  24.08.–30.08. ─────────────────────────  [Kreta]   5
──────────────────────────────────────────────────────────────
```

Sie ist die **Maßlinie über dem Satzspiegel**: links die Rubrik, rechts am
Rand die Zahl, dazwischen ein ausgezogener Haarstrich. Genau die Geste, mit der
md2pdf den druckenden Bereich eines Blattes bemaßt.

* Aufbau als Flexzeile mit expliziter `order`: Registermarke (0), Label (1),
  Zeitraum (2), **Leiter** `::after` (3), Abwesenheits-Badges (4), Zähler (5).
  Der Leiter ist ein `flex: 1 1 auto` hohes 1-px-Element — kein Markup nötig.
* Grundzustand: Mono 11 px versal, Label in `--text`, Rest in `--text-muted`,
  Unterlinie `--border-strong`.
* **`.current` (laufende Woche):** Label, Zeitraum und Zähler in `--accent`,
  Unterlinie und Leiter in `--accent-line`, plus eine **Registermarke** —
  ein 7 × 7 px großes gefülltes Quadrat vor dem Label.
  **Genau eine Registermarke pro Bildschirm.** Sie ist die einzige Stelle,
  an der die Oberfläche laut wird; sie zweimal zu vergeben, macht sie
  wertlos. In der Ansicht „Heute" gibt es keine Gruppenköpfe — dort ist die
  ganze Seite der laufende Bereich, und die Marke sitzt in der Kopfzeile
  (`.main-header.is-now`, gesetzt in `refresh()`), zusammen mit einer
  Unterlinie in `--accent-line`.
* **`.past`:** dasselbe in `--overdue`, aber **ohne** Marke.
* Dieselbe Komponente trägt die Tagesköpfe der Aktivitätsansicht und die
  Abschnitte „Bevorstehend" / „Vergangen" der Abwesenheiten.

### 7.2 Sidebar

* Item-Layout `[icon 24×24] [label, flex:1] [count]`, Padding `6px 10px`,
  **Radius `--radius-pill`**.
* Idle transparent, Hover `--hover`, Aktiv `--accent-soft` mit `--accent` für
  Icon und Label (`font-weight: 500`).
* Zähler in Mono 11 px, `tabular-nums`.
* Der Abschnittskopf („PROJEKTE") ist eine Marginalie mit `--border`-Unterlinie
  — dieselbe Rubrikgeste wie in 7.1, nur ohne Leiter, weil hier keine Zahl am
  rechten Rand steht.
* Das **„Heute"-Icon** trägt weiterhin dynamisch die heutige Tageszahl
  (`todayIconSVG()`).

### 7.3 Aufgabenzeile

```
◯   Titel der Aufgabe  ⟳
    📅 HEUTE · ⚑ SAMSTAG · 💬 1                    Projekt •
```

* **Checkbox:** Hohlkreis 18 × 18 px, Rand = Prioritätsfarbe, Innenfläche
  10 %-Tint (2.4). Hover blendet ein halbtransparentes Häkchen ein.
* **Titel:** Sans 14 px / `1.45` in `--text`.
* **Datumszeile:** Mono 11 px, **versal**, `0.07em` gesperrt, `tabular-nums`,
  Farbe nach 2.5. Das ist der Faden, der alle Ansichten zusammenhält.
* **Projekttag** rechtsbündig in Sans (ein Projektname ist ein Wort, kein
  Ablesewert), in der Projektansicht ausgeblendet.
* Trennlinie `1px solid var(--border)`, Mindesthöhe `40px`.

### 7.4 Quick-Add

* Karte auf `--surface` mit `--border-strong` und `--radius`. Bei Fokus
  wechselt die Fläche auf `--main-bg` und der Rand auf `--accent` — das Feld
  wird zu Papier, sobald man hineinschreibt.
* Die Overlay-Marken (`.qa-mark-*`) sind Pillen: Datum in `--accent`, Projekt
  in `--text` (Tusche auf Papier), Prioritäten in ihrer Stufenfarbe.
  **Ihre Geometrie darf sich nicht ändern** — das Overlay liegt deckungsgleich
  über dem unsichtbaren Text des Inputs; Padding oder Margin zu verschieben
  bringt beide auseinander.

### 7.5 Dialoge

* `--radius`, 1 px `--border`, Backdrop `rgba(21,23,28,.42)`.
* **Feldrubriken sind Marginalien** — `dialog label`, `.dialog-label` und
  `.td-field-label` sind Mono 11 px versal. Eingabefelder darin setzen die
  Sans wieder (`dialog label input, dialog label select`).
* Buttons: primär gefüllt in `--accent` mit `--on-accent`, sekundär
  `--main-bg` mit `--border-strong`, destruktiv als Textbutton in `--p1` links
  (`margin-right: auto`). Alle drei sind Pillen.
* Fehler inline als Streifen in `--danger-bg` / `--p1`, nie global.

### 7.6 Aufgaben-Modal

* Zweispaltig `1fr 320px`, Kopfleiste und rechte Spalte auf `--surface`,
  Trennkanten in `--border-strong`.
* **Prioritätsschalter** `P1…P4`: Mono 700, 11 px, `0.1em`, Pillen. Ausgewählt
  = 2 px Rand in der Stufenfarbe plus 10 %-Tint.
* **Wochentagsschalter** `MO…FR`: Mono 11 px versal, Pillen.
* **Datumsfelder** in Mono mit `tabular-nums`; das Deadline-Feld nimmt
  Freitext („morgen, +5, ende juni") und bleibt deshalb Sans.
* Kommentare: Karte auf `--surface` mit `--radius`, Zeitstempel in Mono,
  Abschnittskopf „KOMMENTARE" als Marginalie mit Unterlinie.
* Code in gerenderten Beschreibungen: `--font-mono` auf `--surface` mit
  1 px `--border` — dasselbe Bild wie ein Codeblock in einem md2pdf-Dokument.

### 7.7 Farbwähler im Projektdialog

* 13 Kreise à 22 px. Ausgewählt = Rand in `--text` plus 1 px weißer Innenring.
* Hex-Feld daneben in Mono.

### 7.8 Abwesenheitszeile

* Layout `[Typ-Badge] [Label] [Zeitraum + Meta] [Edit] [Delete]`.
* Typ-Badge: Pille in den Tokens aus 2.6, **Sans** 11 px / 500 (der Typ ist
  ein Wort).
* Zeitraum in Mono 12 px, Metazeile („12 TAGE · BEGINNT IN EINER WOCHE") in
  Mono 11 px versal.
* Die laufende Abwesenheit liegt auf `--hover` mit `--radius` — neutral, nicht
  im Akzent: sie ist eine Ortsmarke, kein Signal.

### 7.9 Zeitleiste (12 Wochen)

Das einzige Element im Produkt, das Zeit **räumlich** darstellt — und damit die
direkte Entsprechung zum Blatt Papier in md2pdf.

* `.tl-track`: Höhe `10px`, Pille, Grundfläche `--surface`. Die Wochengrenzen
  sind kein Markup, sondern ein `repeating-linear-gradient` in
  `--border-strong` mit zwölf gleich breiten Feldern.
* `.tl-span`: ein Balken je Abwesenheit in der `-fg`-Variante der Typfarbe.
* `.tl-today`: Punkt in `--accent` mit Ring in `--main-bg`.
* `.tl-weeks` / `.tl-months`: Mono 10 px, `tabular-nums`; die laufende Woche
  trägt `.cur` und steht in `--accent` / 700. Monatskürzel versal.
* Hover auf einer Listenzeile hebt den zugehörigen Balken (`scaleY(1.6)`).

> **Achtung, Zwilling:** Dieselben `.tl-*`-Klassen existieren ein zweites Mal
> als Inline-`<style>` in `internal/httpapi/share.go`. Geometrieänderungen
> hier und dort nachziehen.

### 7.10 Die Share-Seite (`GET /api/absences.html`)

Ein eigenständiges Dokument, kein Fenster in die App — und die Stelle, an der
md2pdfs Umkehrung wörtlich gilt: **das Blatt bleibt auch im Dunkelmodus
Papier** (`--paper: #e9e7e1`), nur der Grund darunter wird dunkel. Im
Dark-Block werden deshalb ausschließlich `--bg`, `--paper`, `--edge`,
`--shade` und `--offpaper` (die Fußzeile *neben* dem Blatt) neu belegt.

* Die Überschrift „Availability" ist die **Rubrik** (Marginalie), die
  Statuszeile ist die **Nachricht** (Sans 600, bis `1.7rem`) — die Seite
  beantwortet eine Frage, also steht die Antwort oben und groß.
* Verfügbar = `--ok` (Prozess-Cyan, „jetzt erreichbar"), abwesend =
  Schnittmarken-Rot, Dienstreise = Indigo (dasselbe wie `business_trip`).
* Die Seite lädt die Schriften aus `/static/fonts/` — derselbe Server, keine
  externe Abhängigkeit.

---

## 8. Icons

Inline-SVG im Lucide-Stil, keine Emojis. Alle Icons:

* `viewBox="0 0 24 24"`, `fill="none"`, `stroke="currentColor"`
* `stroke-width: 1.8` für Nav-Icons, `2.0` für stark betonte Elemente
* `stroke-linecap: round; stroke-linejoin: round`
* Größen: `12px` (inline), `16px` (Action-Buttons), `22px` (Nav, Logo)

Inventar in `app.js`:

| Token | Verwendung |
| --- | --- |
| `ICON.calendar` | Erscheinungsdatum vor Datumslabel |
| `ICON.flag` | Deadline vor Datumslabel |
| `ICON.repeat` | Wiederholungs-Indikator am Aufgabentitel |
| `ICON.edit` | Edit-Action |
| `ICON.trash` | Delete-Action |
| `ICON.hashtag` | (reserviert) Projekt-Indikator |
| `todayIconSVG()` | Kalender mit eingelegter Tageszahl für die „Heute"-Nav |

---

## 9. Datumsdarstellung

`smartDateLabel` setzt relative Label; die Darstellung ist immer Mono versal
(7.3):

| Eingabe (Heute = `T`) | Label |
| --- | --- |
| `T` | `HEUTE` |
| `T + 1` | `MORGEN` |
| `T - 1` | `GESTERN` |
| `T + 2 … T + 6` | voller Wochentag (`MITTWOCH`) |
| `T - 2 … T - 6` | voller Wochentag |
| selbes Jahr | `DD.MM.` |
| anderes Jahr | `DD.MM.YYYY` |

Die Farbklasse kommt aus `dateColorClass` (2.5); überfällig setzt zusätzlich
`font-weight: 700`.

---

## 10. Projektfarben

Dreizehn Voreinstellungen als **Druckfarben** — dieselbe Helligkeit und
Sättigung wie der Rest der Oberfläche, damit ein Projektpunkt neben einem
Prioritätshaken nicht wie ein Signal aussieht:

Zinnober `#b5342a` · Ocker `#a2620f` · Safran `#b08b12` · Reseda `#6b8f3a` ·
Tannengrün `#2c7351` · Grünspan `#2b8d84` · Prozesscyan `#0b6e88` ·
Stahlblau `#2f6f9e` · Preußischblau `#3a4fa0` · Indigo `#6a4ca8` ·
Aubergine `#8e3d86` · Krapp `#a83c62` · Graphit `#666c76`

Frei vergebener Hex bleibt erlaubt. Ohne gespeicherte Farbe fällt ein Projekt
auf Graphit zurück, die Inbox auf Preußischblau. **Bestehende Projekte behalten
ihren gespeicherten Hex** — die Liste gilt für neue.

---

## 11. Interaktionsmuster

| Muster | Verhalten |
| --- | --- |
| **Hover-Reveal** | Aktions-Icons werden bei Hover sichtbar, der Platz ist permanent reserviert (`visibility`, nicht `display`). |
| **Selected vor Hover** | Der Auswahl-Hintergrund bleibt, auch wenn der Cursor das Item verlässt. |
| **Eine primäre Aktion** | Pro Dialog genau ein gefüllter Button; alles andere ist Outline oder Textbutton. |
| **Optimistic refresh** | Nach einer Mutation ruft das Frontend `refresh()` — kein clientseitiges Diffing. |
| **Fehler inline** | Fehler stehen im jeweiligen Dialog, nicht global. |
| **Fokus sichtbar** | `:focus-visible` setzt global `2px solid var(--accent)` mit `2px` Offset. |

---

## 12. Bewusst nicht gemacht

* **Kein Theme-Switcher.** Der Dunkelmodus folgt der Systemeinstellung über
  `@media (prefers-color-scheme: dark)`. Deshalb ist jede Farbe ein Token.
* **Kein Build-Schritt.** `index.html`, `style.css`, `app.js` und die sechs
  WOFF2-Dateien liegen als Quelltext in `internal/web/static/` und werden
  per `go:embed` eingebettet. Kein Bundler, kein PostCSS, keine
  Tailwind-Klassen im Markup.
* **Keine zweite laute Stelle.** Wenn eine neue Komponente Aufmerksamkeit
  braucht, ist die richtige Frage, ob die Registermarke (7.1) noch am
  richtigen Ort sitzt — nicht, welche zweite Farbe man einführt.
* Drag & Drop, Filter/Labels, Subtasks, Kalender- und Board-Ansicht
  (siehe Lastenheft, „Out of Scope").

---

## 13. Anwenden bei Änderungen

* **Neue Komponente?** Erst die Tokens aus dieser Datei nutzen. Neue Variable
  nur, wenn sich keine bestehende ehrlich verwenden lässt — dann hier
  eintragen.
* **Neue Farbe?** Gegen §2 prüfen. Signalfarben sind vergeben (Cyan = jetzt,
  Rot = Stopp); alles andere ist Graustufe, bis das Gegenteil begründet ist.
* **Text hinzugefügt?** Sans oder Mono nach der Faustregel in §3.1 —
  gewähltes Wort gegen ausgerechneten Wert.
* **Rundung gebraucht?** §1, Regel 3: Pille, wenn man es drückt oder es etwas
  benennt; `--radius`, wenn es etwas trägt.
* **Neues Icon?** Lucide-Stil, ins `ICON`-Objekt, `currentColor` benutzen.
* **Schrift geändert?** Die WOFF2-Subsets werden aus
  `../md2pdf/assets/fonts/*.ttf` mit `pyftsubset` erzeugt (Latin + Latin-Ext,
  `kern,liga,calt,tnum,frac`, `--flavor=woff2`). Die Lizenz muss mitwandern.
