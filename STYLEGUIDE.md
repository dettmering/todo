# Styleguide

Dieses Dokument hält fest, wie unser Frontend sich an einem etablierten
Task-Manager-Webclient orientiert. Grundlage war eine aus dem Browser
gespeicherte Seite dieses Referenz-Clients samt zugehörigen Assets.
Die dortigen CSS-Klassen sind minifiziert (`.xPiLGH0`, `.JczMOAc`, …),
weshalb dieser Guide nicht die Originalklassen kopiert, sondern die
**Design-Tokens und Strukturmuster** extrahiert und in unsere eigene,
lesbare Vanilla-CSS-Implementierung überträgt.

> Referenz-Theme: **„Moonstone"** (helle, leicht warme Variante,
> Default-Theme des Referenz-Clients). Erkennbar im HTML an
> `<html class="theme_moonstone …">`.

---

## 1. Designsystem-Schichten

Der Referenz-Client verwendet ein dreischichtiges CSS-Variablensystem:

| Schicht | Beispiel | Bedeutung |
| --- | --- | --- |
| **Produktbibliothek** | `--product-library-background-base-primary: #fff;` | Reine Tokens (rohe Hex-Werte), themeneutral definiert. |
| **Reactist** | `--reactist-spacing-large: 16px;` | Komponenten-Tokens des Reactist-Designsystems, oft auf Produktbibliothek-Tokens gemappt. |
| **Theme-Layer** | `--theme-card-sidebar-color: #fcf9f8;` | Spezifische Werte pro Theme (Moonstone, Reef, Snow Pine, …). |

Wir spiegeln das vereinfacht in einer einzigen Token-Ebene (`:root` in
`internal/web/static/style.css`), weil wir mit nur einem Theme arbeiten.
Wenn weitere Themes kommen, kann diese Schicht problemlos um
`html.theme-dark { … }` oder Ähnliches erweitert werden.

---

## 2. Farbpalette (Moonstone)

### 2.1 Flächen und Hintergründe

| Token (Referenz) | Wert | Verwendung | Unser Mapping |
| --- | --- | --- | --- |
| `--product-library-background-base-primary` | `#fff` | Hauptinhaltsbereich | `--main-bg` |
| `--product-library-background-base-secondary` | `#fcfbf8` | Leicht abgesetzte Flächen | (frei) |
| `--theme-card-sidebar-color` (Moonstone) | `#fcf9f8` | Sidebar-Hintergrund | `--sidebar-bg` |
| `--product-library-background-overlay` | `rgba(0,0,0,.4)` | Dialog-Backdrop | `dialog::backdrop` |
| `--theme-card-sidebar-hover-color` (Moonstone) | `#f2f2f2` | Hover auf Sidebar-Items | `--hover` |
| `--theme-card-sidebar-selected-color` (Moonstone) | `#fee6e3` | Aktive Auswahl in Sidebar | `--accent-soft` |

### 2.2 Akzentfarbe & Brand

Der Referenz-Client hat **rot** (`#dc4c3e`) als Akzent. Diese Farbe wird sehr
sparsam eingesetzt: Brand-Logo, primärer Button („Add task"),
Auswahl-Highlights in der Sidebar (kombiniert mit `#fee6e3`),
Destructive-Aktion (Löschen).

| Zweck | Wert |
| --- | --- |
| Brand / Akzent | `#dc4c3e` (`--accent`) |
| Akzent-Hover | `#b03d32` |
| Akzent weich (Background bei Auswahl) | `#fee6e3` (`--accent-soft`) |
| Destructive (Löschen-Button-Border) | `#dc4c3e` |

### 2.3 Text

| Rolle | Wert | Verwendung |
| --- | --- | --- |
| Primary | `#202020` (`--text`) | Headlines, Task-Titel |
| Secondary | `#4d4d4d` (`--text-secondary`) | Body Copy, Nav-Items im Idle |
| Muted / Tertiary | `#666` (`--text-muted`) | Datumslabel, Projekttag, Zähler |
| Quaternary | `#999` | Sehr schwach betonte Metadaten |
| Display-Disabled | `#b3b3b3` | Disabled-Icons, P4-Border |

### 2.4 Border und Trennlinien

| Rolle | Wert |
| --- | --- |
| Idle | `#e6e6e6` |
| Strong (Inputs, Cards) | `#e5e5e5` (`--border-strong`) |
| Trennlinie sehr dezent | `#ebebeb` (`--border`) — eigene Wahl, etwas softer als die `#e5e5e5` der Referenz |
| Border-Hover | `#ccc` |

### 2.5 Prioritäten

Die Originalwerte der Referenz sind in den Asset-CSS-Dateien stark
verschachtelt, aber öffentlich bekannt:

| Stufe | Hex | Bedeutung |
| --- | --- | --- |
| **P1** | `#d1453b` | Sehr wichtig |
| **P2** | `#eb8909` | Wichtig |
| **P3** | `#246fe0` | Normal |
| **P4** | `#b3b3b3` | Niedrig (entspricht `--product-library-display-quaternary-…`-Stufe) |

Diese vier Farben werden in der hohlen Checkbox als Border verwendet,
mit einem 8 %-igen Tint als Innenfläche (`rgba(... ,0.08)`), damit auch
P4 deutlich von einer normalen Border unterscheidbar bleibt.

### 2.6 Status-Datumsfarben

Die Referenz färbt Datumslabel je nach Fälligkeit. Wir spiegeln das mit
dezenten Akzenten:

| Status | Wert | Aussagekraft |
| --- | --- | --- |
| Heute | `#058527` (`--due-today`) | Grün |
| Morgen | `#ad6200` (`--due-tomorrow`) | Warmgelb/Orange |
| Diese Woche / Zukunft | `#246fe0` (`--due-future`) | Blau |
| Überfällig | `#d1453b` (`--overdue`) | Rot, ggf. fett |

---

## 3. Typografie

| Eigenschaft | Wert |
| --- | --- |
| Body-Font-Family | System-Stack: `-apple-system, BlinkMacSystemFont, "Segoe UI", "Inter", system-ui, sans-serif` |
| Base-Font-Size | `14px` |
| Line-Height für Body | `21px` (die Referenz verwendet am häufigsten `21` und `20`) |
| Body-Color | `#4d4d4d` |
| Headline (H1) | `26px / 700 / letter-spacing -0.02em` |
| Sektionsüberschrift | `14px / 700 / letter-spacing -0.005em` |
| Metadaten / Datum | `12px / 400 / line-height 16px` |
| Nav-Items | `14px / 500 (active) / 400 (idle) / line-height 20px` |

**Antialiasing** ist explizit aktiviert (`-webkit-font-smoothing:
antialiased; -moz-osx-font-smoothing: grayscale;`).

---

## 4. Spacing-Skala

Reactist definiert sechs Stufen, alle in 4er-Schritten:

| Token | Wert |
| --- | --- |
| `--reactist-spacing-xsmall` | `4px` |
| `--reactist-spacing-small` | `8px` |
| `--reactist-spacing-medium` | `12px` |
| `--reactist-spacing-large` | `16px` |
| `--reactist-spacing-xlarge` | `24px` |
| `--reactist-spacing-xxlarge` | `32px` |

Wir verwenden dieselbe 4er-Skala, ohne sie zu Tokens zu erheben — bei
unserer Komponentenmenge wäre das Overengineering.

---

## 5. Radien und Schatten

| Element | Radius |
| --- | --- |
| Inputs, Pills, kleine Buttons | `4 – 6 px` |
| Cards, Quick-Add, Dialoge | `8 – 10 px` |
| Avatare, Color-Swatches, Checkboxen | `50 %` |
| Skip-Pill „Add task" usw. | `24 px` (kapselförmig) |

Schatten sind in der Referenz extrem zurückhaltend. Verwendet werden:

* Dialoge: `0 16px 48px rgba(0,0,0,0.18)` (wir nutzen den gleichen)
* Hover-Cards: leichte `0 2px 8px rgba(0,0,0,0.08)`

---

## 6. Layoutgerüst

```
┌─────────────┬──────────────────────────────────────────────┐
│             │  ┌─ Header (H1 + Subtitle) ───────────────┐  │
│             │  │                                        │  │
│  Sidebar    │  │  Quick-Add (Card, focus-ring akzent)   │  │
│  280 px     │  │                                        │  │
│  (warm bg)  │  │  Group Header                          │  │
│             │  │  Task / Task / Task                    │  │
│             │  │  Group Header                          │  │
│             │  │  Task / Task                           │  │
│             │  └────────────────────────────────────────┘  │
└─────────────┴──────────────────────────────────────────────┘
```

* Shell: zweispaltiges Grid `280px 1fr`, volle Höhe (`100vh`),
  Sidebar mit eigenem Scrollbereich, Main-Pane mit eigenem.
* Main-Pane mittig zentriert mit `max-width: 800px`, das ergibt die
  für solche Clients typische Lesebreite.
* Padding im Main: `32px 60px 96px` (großzügig nach unten für scrollende
  Listen).

---

## 7. Komponenten

### 7.1 Sidebar-Navigation

* Höhe pro Item ≈ `32px` (Padding `6px 8px`, Inhalt `20px` line-height).
* Layout: `[icon 24×24] [label, flex:1] [count]`.
* **Idle:** transparent. **Hover:** `#f2f2f2`. **Active:** `#fee6e3` mit Akzentfarbe `#dc4c3e` für Icon und Label.
* Icons sind 22 × 22 px Inline-SVGs im Lucide-Stil
  (`stroke-width: 1.8`, `stroke-linecap: round`).
* Das **„Heute"-Icon** ist ein Kalender-SVG, in dessen Innenrahmen wir
  dynamisch die heutige Tageszahl positionieren (klein, fett, mittig
  über dem horizontalen Strich). Genau dieses Muster nutzt die Referenz
  auch.

### 7.2 Sidebar-Projektliste

* Project-Item-Markup: `[Farbpunkt 10px] [Name, flex:1] [Edit-Stift bei Hover oder Count]`.
* Hover blendet **Edit** ein und **Count** aus — ein Standard-Pattern
  der Referenz (gleicher Platz wird genutzt, kein Layout-Shift).
* Farbpunkt nutzt die Projektfarbe (Default `#246fe0` für Inbox, sonst
  `#808080` als neutrales Grau).
* Klick auf das Item navigiert; Klick auf den Stift öffnet den
  Projekt-Dialog. Event wird mit `stopPropagation` getrennt.

### 7.3 Task-Zeile

```
[ ◯ ]   Titel der Aufgabe   🔁
        📅 Heute · ⚑ 25.05.                            • Projekt
                                                  [ ✎ ] [ 🗑 ]
```

* **Checkbox:** Hohlkreis 18 × 18 px, Border-Color = Prioritätsfarbe,
  Innenfläche `rgba(priority, 0.08)`. Beim Hover blendet ein
  halbtransparentes Häkchen-SVG ein.
* **Body:** Titel (14 px, primary) und darunter eine Datumszeile (12 px,
  muted) mit Mini-SVG-Icons.
* **Recurrence-Indikator:** kleines Repeat-SVG (12 × 12) direkt am
  Titelende — kein Emoji.
* **Project-Tag:** rechtsbündig, klein gesetzt. In der Projektsicht
  ausgeblendet (sonst redundant — Verhalten wie in der Referenz).
* **Actions:** Bearbeiten/Löschen erscheinen erst bei Hover
  (`visibility: hidden` → `visible`, damit kein Layout-Shift entsteht).
* Trennlinie zwischen Tasks: `1px solid #ebebeb` an `border-bottom`.
* Mindesthöhe `40px`.

### 7.4 Group-Header (Demnächst / Alle offenen)

```
KW 21   18.05.–24.05.                                         3
─────────────────────────────────────────────────────────────────
```

* Inhalte: Label (fett, 14 px), Range (regular, muted), Count
  (rechtsbündig, muted).
* In der Referenz sind die Headers in „Demnächst" tagweise („Montag,
  20. Mai"); wir gruppieren bewusst nach **Kalenderwoche**, weil das
  für unseren Anwendungsfall die richtige Verdichtung ist.

### 7.5 Quick-Add

* Card-Style, 1 px Border `#e5e5e5`, Radius 10 px.
* Beim Fokus wechselt die Border auf die Akzentfarbe — gleiches Muster
  wie in der Referenz (dort animiert es subtiler).
* Optionsleiste unter dem Eingabefeld mit gepunkteter Trennlinie
  (`border-top: 1px dashed`), Selects mit `4px 8px` Padding und
  6 px Radius.
* Submit per `Enter` legt die Aufgabe an, kein expliziter Button.

### 7.6 Dialoge

* Radius 10 px, Backdrop `rgba(0,0,0,0.4)`.
* Form-Layout: vertikaler Flex mit `gap: 12px`, sauberes Padding `20px 24px`.
* Inputs/Selects haben Focus-Ring in Akzentfarbe statt Browser-Default.
* Buttons:
  * **Primary** („Speichern", „Anlegen"): rot gefüllt (`#dc4c3e`),
    weißer Text, Hover `#c44035`.
  * **Secondary** („Abbrechen"): weißer Hintergrund, dezenter Border.
  * **Danger** („Löschen"): outline-Style mit P1-Farbe, links
    angeordnet (per `margin-right: auto`).
* Error-Anzeige als roter Toast-Streifen im Dialog.

### 7.7 Color-Picker im Projektdialog

* 13 vordefinierte Swatches als 22 × 22 px Kreise in einem Grid.
* Selected = Border in Textfarbe + 1 px Inset weißer Ring, damit auch
  bei dunklen Farben sichtbar.
* Hover skaliert leicht (`transform: scale(1.12)`).
* Hex-Feld daneben akzeptiert `#rgb` und `#rrggbb`, ergänzt
  führendes `#` automatisch.

---

## 8. Icons

Wir verwenden **Inline-SVG im Lucide-Stil** statt Emojis. Alle Icons:

* `viewBox="0 0 24 24"`
* `fill="none"`, `stroke="currentColor"`
* `stroke-width: 1.8` für Nav-Icons, `2.0` für stark betonte Elemente
* `stroke-linecap: round; stroke-linejoin: round`
* Größen: `12px` (inline in Texten), `16px` (Action-Buttons),
  `22px` (Sidebar-Nav, Logo).

Inventar in `app.js`:

| Token | Verwendung |
| --- | --- |
| `ICON.calendar` | Erscheinungsdatum vor Datumslabel |
| `ICON.flag` | Deadline vor Datumslabel |
| `ICON.repeat` | Wiederholungs-Indikator am Task-Titel |
| `ICON.edit` | Edit-Action |
| `ICON.trash` | Delete-Action |
| `ICON.hashtag` | (reserviert) Projekt-Indikator |
| `todayIconSVG()` | Kalender mit eingelegter Tageszahl für die „Heute"-Nav |

Die Referenz verwendet teils ein eigenes Icon-Set mit minimal anderen
Endcap-Radien — die Unterschiede sind in der Wahrnehmung minimal, und
Lucide ist Open-Source-freundlich.

---

## 9. Datumsdarstellung

Die Referenz setzt extrem stark auf **relative Datumslabel**. Unsere
Übernahme in `smartDateLabel`:

| Eingabe (Heute = `T`) | Label |
| --- | --- |
| `T`            | `Heute` |
| `T + 1`        | `Morgen` |
| `T - 1`        | `Gestern` |
| `T + 2 … T + 6`| voller Wochentag (z. B. `Mittwoch`) |
| `T - 2 … T - 6`| voller Wochentag |
| selbes Jahr   | `DD.MM.` |
| anderes Jahr  | `DD.MM.YYYY` |

Die zugehörige Farbklasse kommt aus `dateColorClass` (siehe 2.6).
Überfällig setzt zusätzlich `font-weight: 500`.

---

## 10. Projektfarben

Die Referenz bietet eine kuratierte Palette aus etwa 13 Farben (Berry Red,
Red, Orange, Yellow, Olive Green, …). Wir nähern uns mit denselben
Familien an, leicht abweichend benannt (Tomate, Orange, Gelb, Olive,
Grün, Mint, Türkis, Hellblau, Blau, Lila, Magenta, Rosa, Grau).

Frei vergebener Hex bleibt erlaubt (siehe Color-Picker).
Inbox ist farblich auf `#246fe0` gepinnt — entspricht dem
Inbox-Indigo der Referenz.

---

## 11. Interaktionsmuster

| Muster | Verhalten |
| --- | --- |
| **Hover-Reveal**       | Aktions-Icons auf Items werden bei Hover sichtbar, der Platz ist permanent reserviert (`visibility`, kein `display: none`). Verhindert Layout-Sprünge. |
| **Selected vs Hover**  | Selected überschreibt Hover — selected-Background bleibt, selbst wenn der Cursor das Item verlässt. |
| **Single primary action** | Pro Dialog/Form genau ein primärer Button (rot). Sekundäres ist outline. |
| **Optimistic refresh** | Nach einer Mutation (`createTask`, `completeTask` …) ruft das Frontend `refresh()` — wir verzichten bewusst auf clientseitiges Diffing; es ist schnell genug. Bei größeren Datenmengen würden wir auf optimistische Updates wechseln. |
| **Toast-freie Errors** | Fehler werden inline im jeweiligen Dialog angezeigt, nicht global. |

---

## 12. Bewusst nicht übernommen

Aus dem Lastenheft („Out of Scope" und „Explizit gestrichene Anforderungen"):

* Drag & Drop in der Task-Liste — die Referenz nutzt dafür ein eigenes
  Virtuoso-basiertes Virtual-List-Setup (`data-testid="virtuoso-item-list"`),
  bei unserer Größenordnung unnötig.
* Theme-Switcher, Dark Mode.
* Filters/Labels-Bereich, Karma, Activity Log.
* Avatare im User-Switcher (Single-User-Anwendung).
* Subtasks (Modell nicht vorgesehen).
* Kalender- und Board-Ansicht.

Sollten wir später Erweiterungen wie Subtasks oder Filter ergänzen,
bietet die Token-Schicht in `style.css` und das gut isolierte
Icon-Inventar in `app.js` saubere Andockpunkte.

---

## 13. Mapping unserer CSS-Variablen

| Unsere Variable | Wert | Entspricht Referenz-Token |
| --- | --- | --- |
| `--sidebar-bg` | `#fcf9f8` | `--theme-card-sidebar-color` (Moonstone) |
| `--main-bg` | `#ffffff` | `--product-library-background-base-primary` |
| `--accent` | `#dc4c3e` | `--product-library-actionable-destructive-idle-fill` |
| `--accent-soft` | `#fee6e3` | `--theme-card-sidebar-selected-color` (Moonstone) |
| `--text` | `#202020` | `--product-library-display-primary-idle-tint` |
| `--text-secondary` | `#4d4d4d` | `--theme-card-content-color` (Moonstone) |
| `--text-muted` | `#666` | `--product-library-display-secondary-idle-tint` |
| `--border` | `#ebebeb` | (eigene Wahl, etwas softer als das `#e5e5e5` der Referenz) |
| `--border-strong` | `#e5e5e5` | `--product-library-display-quaternary-idle-tint` |
| `--hover` | `#f2f2f2` | `--theme-card-sidebar-hover-color` (Moonstone) |
| `--p1 … --p4` | `#d1453b / #eb8909 / #246fe0 / #b3b3b3` | Prioritäts-Palette der Referenz |

---

## 14. Anwenden bei Änderungen

* **Neue UI-Komponente?** Erst die Tokens aus dieser Datei nutzen, nur
  bei Bedarf eine neue Variable in `:root` ergänzen.
* **Neue Farbe?** Vorher prüfen, ob die Referenz sie im
  `--product-library-*`-Namensraum verwendet — wenn ja, denselben Hex
  übernehmen.
* **Neue Icons?** Lucide-Stil, in `ICON`-Objekt eintragen, `currentColor`
  benutzen, damit das Icon Theme-fähig bleibt.
* **Neue Action auf einem Item?** Hover-Reveal-Muster anwenden, Platz
  über `visibility`/`opacity` reservieren, nicht über `display`.
