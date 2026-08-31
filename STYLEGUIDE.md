# Styleguide

Dieses Dokument hält fest, wie unser Frontend aussieht und warum. Es ist
**maßgeblich**: eine Farbe, ein Radius oder eine Schriftrolle, die hier nicht
steht, gehört nicht ins Stylesheet, bis sie hier steht.

> **Gestalterische Grundlage: die Weboberfläche von `md2pdf`**
> (`../md2pdf`, `src/webui.js` und `assets/template.typ`). Von dort kommen die
> Haltung zur Typografie (eine Lesende, eine Ablesende), die Druckerei-Palette
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

1. **Zwei Schriften, zwei Aufgaben.** Inter setzt alles, was man
   *liest*. JetBrains Mono setzt alles, was man *abliest* — Zahlen, Daten,
   Kürzel, Rubriken, Marginalien. Mono ist immer klein, versal, gesperrt und
   hat Tabellenziffern.
2. **Zwei Signalfarben, sonst Graustufen.** Prozess-Cyan heißt „jetzt"
   (heute fällig, laufende Woche, aktive Auswahl, Marke).
   Schnittmarken-Rot heißt „überfällig / löschen". Priorität ist **keine
   Ampel, sondern Farbdichte**: Rot, Ocker, Graphit, blass.
3. **Bedienelemente sind kantig, Marken sind rund.** Alles, was man *drückt*
   oder was etwas *aufnimmt* — Schaltfläche, Karte, Dialog, Eingabefeld,
   Menü, Nav-Zeile —, teilt sich eine Kante (`--radius`, 4 px). Eine Pille
   (`--radius-pill`) ist nur, was etwas **benennt** und selbst keine Aktion
   ist: Abwesenheits-Badge, Kürzel im Quick-Add, Balken der Zeitleiste.
   Kreise bleiben Kreise: Checkbox, Projektpunkt, Farbwähler.

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
| Marginalie (Daten, Zähler, Rubriken) | `--text-muted` | `#767d87` | `#8d95a0` |
| Haarstrich | `--border` | `rgba(21,23,28,.12)` | `rgba(232,233,236,.14)` |
| Kräftiger Strich (Rubrikzeile, Feldrand) | `--border-strong` | `rgba(21,23,28,.22)` | `rgba(232,233,236,.26)` |

`--text-muted` trägt die Datums- und Zählerzeilen und muss deshalb die
WCAG-Schwelle halten: `#767d87` liegt bei 4,6:1 auf Papier. Ein blasserer Wert
sieht auf einem Screenshot ruhiger aus und ist im Betrieb nicht mehr lesbar.

Linien sind **Alphawerte, keine Hex-Grautöne**. Dadurch stimmen sie über
Papier, Grund und Sidebar hinweg, ohne dass es drei Strichfarben braucht.

### 2.3 Akzent — die Prozessfarbe

| Zweck | Token | Hell | Dunkel |
| --- | --- | --- | --- |
| Akzent | `--accent` | `#0b6e88` | `#56c5e2` |
| Akzent Hover | `--accent-hover` | `#095a6f` | `#7ad4ec` |
| Akzentlinie (Registermarke, Leiter) | `--accent-line` | `#0e88a6` | `#3fb4d4` |
| Akzent weich (Auswahl in der Sidebar) | `--accent-soft` | `#e1edf0` | `#16323c` |
| Textauswahl im Quick-Add (halbtransparent) | `--qa-selection` | `rgba(14,136,166,.28)` | `rgba(63,180,212,.32)` |
| Schrift auf Akzentfläche | `--on-accent` | `#ffffff` | `#10141a` |

In md2pdf markiert Cyan den Bereich, der **druckt** — das, was gerade zählt.
Bei uns heißt es dasselbe: „jetzt". Es trägt daher **beide** Rollen, Marke und
Fälligkeit, und das ist Absicht, keine Kollision: die laufende Woche, das
heutige Datum, das ausgewählte Nav-Item und das Logo sind alle „jetzt".

### 2.4 Priorität als Farbdichte

| Stufe | Token | Hell | Dunkel | Lesart |
| --- | --- | --- | --- | --- |
| **P1** | `--p1` | `#b5342a` | `#e86a5e` | Schnittmarken-Rot |
| **P2** | `--p2` | `#bf8420` | `#d79a4a` | Ocker |
| **P3** | `--p3` | `#5b6472` | `#98a1ae` | Graphit |
| **P4** | `--p4` | `#a9aeb6` | `#5c646f` | blass |

P2 stand im Hellen zunächst auf `#a2620f`. Das lag mit L\* 47,6 nur 5,6
Punkte über P1 (41,9) und im selben warmen Winkel — zwei dunkle, satte Töne
nebeneinander, die man am 18-px-Ring nicht auseinanderhält. `#bf8420` hebt
den Abstand auf 17,7 Punkte, ohne den Ockerton zu verlassen.

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

Eine Ausnahme: „Morgen" und die Deadline-Flagge stehen im Hellen weiter auf
`#a2620f`, dem ursprünglichen Ocker, während `--p2` auf `#bf8420` aufgehellt
wurde (§2.4). Der Grund für die Aufhellung — die Verwechslung mit dem Rot am
Prioritätsring — greift hier nicht: Datumsangaben stehen als Text, nicht als
Ring, und „überfällig" trägt zusätzlich den Mono-700-Schnitt.

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

**Inter** für `--font-sans`, **JetBrains Mono** für `--font-mono`.

Vorher standen hier IBM Plex Sans und IBM Plex Mono — die Schnitte, die
md2pdf selbst ausliefert. Sie waren die naheliegende Wahl, solange es um die
Herkunft der Gestaltung ging, aber nicht die richtige für diese Oberfläche:
Plex hat eine vergleichsweise kleine x-Höhe und enge Punzen, und das rächt
sich genau bei den Größen, aus denen diese App besteht — 12 px Marginalien,
13 px Rubrikzeile, 15 px Fließtext. Inter und JetBrains Mono sind für den
Bildschirm und für kleine Grade gezeichnet: größere x-Höhe, offenere Punzen.
Die **Rollen** aus Regel 1 ändern sich dadurch nicht, nur die Schnitte.

Beide Familien kommen über **Google Fonts**. `index.html` trägt dazu zwei
`preconnect` und einen Stylesheet-Link auf `fonts.googleapis.com`; die
Schriftdateien selbst holt der Browser von `fonts.gstatic.com`. Fällt Google
Fonts aus, greift der Fallback-Stack in `--font-sans` / `--font-mono`.

Vorher lagen die Schnitte als WOFF2-Subsets unter
`internal/web/static/fonts/` und gingen über `go:embed` mit ins Binary. Das
waren sechs zusätzliche Anfragen an den eigenen Host bei jedem Aufruf. Der
Reverse Proxy vor der App begrenzt die Zahl der Anfragen, und mit den
`preload`-Links standen die Schriften vor den Skripten in der Warteschlange:
`app.js` kam als `429` zurück und die Seite blieb leer. Von einem fremden
Host zählen die Schriften nicht gegen dieses Limit.

| Schnitt | Familie | Verwendung |
| --- | --- | --- |
| Sans 400 | `--font-sans` | Regular |
| Sans 400 kursiv | `--font-sans` | Regular kursiv |
| Sans 500 | `--font-sans` | Medium |
| Sans 600 | `--font-sans` | SemiBold |
| Mono 400 | `--font-mono` | Regular |
| Mono 700 | `--font-mono` | Bold |

Genau diese sechs stehen in der Google-Fonts-URL. Wer einen weiteren Schnitt
braucht, muss ihn dort eintragen — sonst synthetisiert der Browser ihn.

**Es gibt kein 700 in der Sans.** `strong`/`b` sind global auf `600` gesetzt,
damit gerendertes Markdown keinen synthetischen Fettschnitt erzeugt.

### 3.1 Die Rollen

| Rolle | Wert | Verwendung |
| --- | --- | --- |
| Body | Sans 400 / `15px` / `1.55` / `--text-secondary` | alles Gelesene |
| Seitentitel (H1) | Sans 600 / `30px` / `-0.022em` | Ansichtsüberschrift |
| Dialogtitel (H2) | Sans 600 / `18px` / `-0.018em` | |
| Aufgabentitel im Modal | Sans 600 / `22px` / `-0.018em` | |
| Aufgabentitel in der Liste | Sans 400 / `15px` / `1.45` / `--text` | |
| **Rubrikzeile** | Mono / `13px` / `0.06em` / versal; Rubrik 700, Rest 400 | Wochen-, Tages- und Abschnittsköpfe (7.1) |
| **Marginalie** | Mono 400 / `12px` / `0.09em` / versal / `--text-muted` | Feldrubriken, Abschnittsköpfe der Sidebar, Hinweise |
| **Ablesewert** | Mono 400 / `12px` / `0.01–0.03em` / gemischt / `tabular-nums` | Daten, Uhrzeiten, Zeiträume, Zähler |

Die Utility-Klassen `.mono` und `.eyebrow` in `style.css` fassen Marginalie und
Ablesewert zusammen; Komponenten dürfen die Werte auch direkt setzen, wenn sie
zusätzlich etwas anderes brauchen.

**Versalien nur für Rubriken, nicht für Werte.** Eine Rubrik ist ein kurzes,
festes Wort und darf ihre Wortform verlieren — bei einem Wert liest man genau
sie: `Mittwoch` ist auf einen Blick erfassbar, `MITTWOCH` muss man
buchstabieren. Ablesewerte stehen deshalb gemischt und nur leicht gesperrt;
die Zugehörigkeit zur Nebenschrift tragen Mono und `--text-muted`.

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
| **Jede Schaltfläche**, Nav-Zeile, Icon-Button, Karte, Dialog, Eingabefeld, Textarea, Select, Menü, Codeblock | `var(--radius)` = `4px` |
| Abwesenheits-Badge, Kürzel im Quick-Add, Schiene und Balken der Zeitleiste | `var(--radius-pill)` = `999px` |
| Checkbox, Projektpunkt, Farbwähler | `50%` |
| „Heute" auf der Zeitleiste | Quadrat, um 45° gedreht (7.9) |

Die Trennung ist die dritte Grundregel aus §1: rund ist, was etwas *benennt*,
kantig ist, was man *bedient*.

**Korrigiert.** Die erste Fassung hatte es umgekehrt — Pillen für alles
Drückbare, Kanten für alles Tragende. Das las sich als Regel gut und ist an
einer konkreten Stelle gescheitert: die Aktionszeile im Aufgaben-Modal
(„Löschen | Abbrechen | Speichern") braucht in Pillenform rund 300 px und
hatte 284. Pillen kosten waagerechte Polsterung, die eine 320-px-Spalte nicht
hat. Dazu kam, dass eine Pille neben einem 4-px-Eingabefeld im selben Feld
steht und sich nicht auf dessen Raster einlässt. Beides zusammen war teurer
als die Merkregel wert war.

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
* **`align-items: baseline`, nicht `center`.** Die Zeile mischt zwei Schnitte;
  `center` legt nur die Zeilenkästen übereinander, nicht die Glyphen. Solange
  beide Schnitte aus derselben Schrift kommen, fällt das nicht auf — sobald
  einer davon noch lädt oder auf einen Systemfont zurückfällt (auf iOS
  beobachtet), steht die Rubrik sichtbar höher als das Datum daneben. Was
  keine eigene Grundlinie beiträgt — Registermarke, Leiter, Badge-Gruppe —
  bekommt `align-self: center` und ist damit aus der Ausrichtung heraus.
  **Dieselbe Regel gilt für jede Zeile, die Schnitte oder Größen mischt**:
  `.activity-day` und `.task .dates` (dort trägt zusätzlich das Icon
  `align-self: center`, damit die Zeile ihre Grundlinie vom Text bekommt).
* Grundzustand: Mono 13 px versal, `0.06em`; die Rubrik selbst in **700** und
  `--text`, Zeitraum und Zähler in 400 und `--text-muted`; Unterlinie
  `--border-strong`. Der Fettschnitt liegt nur auf der Rubrik — läge er auf
  der ganzen Zeile, würde sie bei dieser Größe zur Wand.
* **`.current` (laufende Woche):** Label, Zeitraum und Zähler in `--accent`,
  Unterlinie und Leiter in `--accent-line`, plus eine **Registermarke** —
  ein 8 × 8 px großes gefülltes Quadrat vor dem Label.
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
* **Titel:** Sans 15 px / `1.45` in `--text`.
* **Datumszeile:** Mono 12 px, **gemischt** (nicht versal), `0.01em`,
  `tabular-nums`, Farbe nach 2.5. Das ist der Faden, der alle Ansichten
  zusammenhält — und die Stelle, an der Versalien am meisten gekostet haben
  (siehe 3.1).
* **Projekttag** rechtsbündig in Sans (ein Projektname ist ein Wort, kein
  Ablesewert), in der Projektansicht ausgeblendet.
* Trennlinie `1px solid var(--border)`, Mindesthöhe `40px`.

### 7.4 Quick-Add

* Karte auf `--surface` mit `--border-strong` und `--radius`. Bei Fokus
  wechselt die Fläche auf `--main-bg` und der Rand auf `--accent` — das Feld
  wird zu Papier, sobald man hineinschreibt.
* Die Overlay-Marken (`.qa-mark-*`) sind Pillen: Datum in `--accent`, Projekt
  in `--text` (Tusche auf Papier), Prioritäten in ihrer Stufenfarbe.
* **Keine Marke darf Breite kosten.** Das Overlay liegt deckungsgleich über dem
  unsichtbaren Text des Inputs; jedes Pixel seitliches `padding`, `margin` oder
  `border` an einer Marke schiebt den sichtbaren Text gegen den Cursor — und
  zwar aufaddiert über alle Marken davor, bis Korrigieren unmöglich wird. Die
  Höhe der Pille kommt aus senkrechtem `padding` (an einem inline-Element ohne
  Wirkung aufs Layout), ihr schmaler Rand aus `box-shadow`-Spread.
* **Input und Overlay teilen einen Metrikblock.** Schnitt, Größe, Fettgrad,
  Laufweite, Wortabstand, Zeilenhöhe, Texteinzug und waagerechtes Padding
  stehen in `style.css` genau einmal, für beide Selektoren zugleich. Ein
  `<input>` erbt `font-family` **nicht** vom `body` — sie muss dort
  ausgeschrieben stehen, sonst setzt das Feld in der System-Schrift und das
  Overlay in Inter.
* Die Textauswahl im Feld nutzt `--qa-selection`, halbtransparent: eine deckende
  Fläche würde den Text des Overlays darunter zudecken, und die Markierung sähe
  aus wie ein leerer Balken.

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
* Typ-Badge: Pille in den Tokens aus 2.6, **Sans** 12 px / 500 (der Typ ist
  ein Wort). Die Spalte ist 132 px breit, damit „Dienstreise" nicht umbricht.
* Zeitraum in Mono 13 px, Metazeile („12 Tage · beginnt in einer Woche") in
  Mono 12 px, gemischt.
* Die laufende Abwesenheit liegt auf `--hover` mit `--radius` — neutral, nicht
  im Akzent: sie ist eine Ortsmarke, kein Signal.

### 7.9 Zeitleiste (12 Wochen)

Das einzige Element im Produkt, das Zeit **räumlich** darstellt — und damit die
direkte Entsprechung zum Blatt Papier in md2pdf.

* `.tl-track`: Höhe `10px`, Pille, Grundfläche `--surface`. Die Wochengrenzen
  sind kein Markup, sondern ein `repeating-linear-gradient` in
  `--border-strong` mit zwölf gleich breiten Feldern.
* `.tl-span`: ein Balken je Abwesenheit in der `-fg`-Variante der Typfarbe.
* `.tl-today`: **Raute** in `--accent` mit Ring in `--main-bg` — ein um 45°
  gedrehtes Quadrat, die Meilensteinmarke aus dem Gantt-Diagramm. Sie hebt
  sich damit in der *Form* von den Balken ab und nicht nur in der Farbe.
  9 px Kante ergeben ~12,7 px Diagonale, also dieselbe sichtbare Größe wie
  der Punkt davor. `rotate` steht hinter `translate`, sonst wandert die
  Marke von ihrer Position weg.
* `.tl-weeks` / `.tl-months`: Mono 11 px, `tabular-nums`; die laufende Woche
  trägt `.cur` und steht in `--accent` / 700. Monatskürzel versal.
* Hover auf einer Listenzeile hebt den zugehörigen Balken (`scaleY(1.6)`).

> **Achtung, Zwilling:** Dieselben `.tl-*`-Klassen existieren ein zweites Mal
> als Inline-`<style>` in `internal/httpapi/share.go`. Geometrieänderungen
> hier und dort nachziehen.

### 7.10 Die Share-Seite (`GET /api/absences.html`)

Ein eigenständiges Dokument, kein Fenster in die App: eine Karte auf dem
Grund, mit eigenem, kleinem Tokensatz im Inline-`<style>` von `share.go`.

* **Der Dunkelmodus schaltet die ganze Karte um**, nicht nur den Grund. Die
  Werte im Dark-Block sind dieselben wie in `style.css` (`--paper` = das
  `--surface` der App, `--fg` / `--muted` / `--line` = Tusche, Graphit,
  Haarstrich), damit Karte und Oberfläche nicht auseinanderlaufen.
  `--offpaper` ist der einzige Sonderwert: die Fußzeile steht *neben* der
  Karte auf dem dunklen Grund und braucht dort einen eigenen Ton.
* Die Überschrift „Availability" ist die **Rubrik** (Marginalie), die
  Statuszeile ist die **Nachricht** (Sans 600, bis `1.7rem`) — die Seite
  beantwortet eine Frage, also steht die Antwort oben und groß.
* Verfügbar = `--ok` (Prozess-Cyan, „jetzt erreichbar"), abwesend =
  Schnittmarken-Rot, Dienstreise = Indigo (dasselbe wie `business_trip`).
* **Die Schriften kommen von Google Fonts**, wie in der App auch (§3).
  Für diese Seite war das schon immer zwingend: sie ist die einzige, die
  jemand von außen aufruft, und `/static` liegt hinter der BasicAuth des
  Reverse Proxy — selbst ausgelieferte WOFF2 kämen als 401 zurück. Fällt
  Google Fonts aus, greift der Fallback-Stack in `--sans` / `--mono`.

> **Verworfen:** In md2pdf bleibt das Blatt auch im Dunkelmodus Papier, weil
> dort ein Druckbogen in der Vorschau liegt. Hier war es eine hell
> leuchtende Karte auf dunklem Grund — als Metapher richtig, als Verhalten
> falsch. Wer nachts eine Verfügbarkeit nachschlägt, will nicht geblendet
> werden.

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
* **Schnitt gebraucht, den es noch nicht gibt?** In die Google-Fonts-URL im
  `<head>` von `index.html` eintragen — und in §3 nachtragen. Sonst
  synthetisiert der Browser ihn.
