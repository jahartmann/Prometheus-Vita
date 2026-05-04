# Prometheus SRE-Cockpit Redesign

Datum: 2026-05-04

## Ausgangspunkt

Die aktuelle Anwendung wirkt zu voll und visuell unruhig. Prometheus soll sich wie ein professionelles Proxmox-Infrastruktur-Lagezentrum anfühlen: ruhig, schnell scannbar und eindeutig operational. Die UI bleibt deutschsprachig und ersetzt die Proxmox WebGUI nicht, sondern fokussiert auf Übersicht, Intelligenz und Drill-down.

Gewählte Richtung: **Dunkles SRE-Cockpit**.

## Recherche-Leitplanken

- Grafana empfiehlt Dashboards, die eine klare Frage beantworten, eine logische Progression von Übersicht zu Detail haben und kognitive Last reduzieren: https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/best-practices/
- Carbon/IBM behandelt Statusindikatoren als gezielte Aufmerksamkeitssignale und empfiehlt Farbe, Form, Symbol und Text bewusst zu kombinieren: https://carbondesignsystem.com/patterns/status-indicator-pattern/
- Proxmox Datacenter Manager nutzt eine linke Sidebar, ein Hauptpanel und ein Dashboard mit Remotes, Ressourcen, laufenden Tasks, CPU, Memory, SDN und Subscription-Status: https://pdm.proxmox.com/docs/web-ui.html

## Designprinzipien

1. **Erst Aufmerksamkeit, dann Details.** Jede Hauptseite beantwortet zuerst: Was braucht gerade Aufmerksamkeit?
2. **Dunkel, aber nicht schwarz.** Graphit-Flächen, klare Kontraste, keine Cyberpunk- oder Neon-Anmutung.
3. **Orange ist Marke, nicht Alarm.** Prometheus-Orange markiert Brand, aktive Bereiche und primäre Aktionen. Warnungen bleiben Gelb/Orange, Kritisch Rot, Gesund Grün, Info Blau.
4. **Weniger Karten, bessere Gruppen.** Keine Karten in Karten. Wiederholte Items dürfen Karten sein; Seitenbereiche sind flächige Layoutzonen.
5. **Drill-down statt Informationswand.** Dashboard zeigt Zusammenfassung, Risiken und die wichtigsten Einstiege. Tiefe Tabellen, Logs und Diagramme bleiben auf Detailseiten.
6. **Schnelles Scannen im Betrieb.** Kompakte Typografie, tabellarische Zahlen, klare Statuspunkte, stabile Abstände und keine dekorativen Flächen ohne Funktion.

## Visuelles System

### Farben

Der Standardmodus wird dunkel:

- Hintergrund: sehr dunkles Graphit, nicht reines Schwarz.
- Sidebar: minimal dunkler als der Content-Hintergrund.
- Panels: angehobenes Graphit mit subtiler Border.
- Muted-Flächen: leicht helleres Graphit für Reihen, Filterleisten und sekundäre Gruppen.
- Brand-Akzent: gedämpftes Prometheus-Orange für Logo, aktive Hauptnavigation, primäre Aktionen und ausgewählte Zustände.
- Status: Grün, Gelb, Rot, Blau nach Bedeutung. Statusfarben werden sparsam eingesetzt und immer mit Text oder Symbol ergänzt.

Light Mode bleibt erhalten, wird aber aus demselben System abgeleitet: neutrale helle Flächen, schwarze Typografie, Orange als Brand-Akzent, gleiche Statuslogik.

### Typografie

- System-/Inter-nahe Sans-Serif bleibt passend.
- Zahlen bekommen konsequent tabular figures.
- Seitenüberschriften werden kleiner und produktiver als aktuell übliche Hero-Größen.
- Sekundärtexte werden gekürzt. Wo UI-Text nur Funktionen erklärt, wird er entfernt oder in Tooltips/Empty States verschoben.

### Flächen und Radius

- Standardradius: 8px.
- Toolbars, Buttons, Panels und Tabellen bleiben kompakt.
- Schatten sehr sparsam; Trennung primär über Border, Fläche und Abstand.
- Keine dekorativen Gradients, Glows, Orbs oder großen Illustrationen.

## Layout-Architektur

### App Shell

Die bestehende Struktur `AppLayout -> Sidebar + Header + Main` bleibt erhalten.

Änderungen:

- Sidebar wird ruhiger und weniger textlastig.
- Header wird stärker als Arbeitsleiste behandelt: Suche, Breadcrumbs, Agent-Status, Notifications, User.
- Main erhält ein konsistentes Layout mit maximaler Breite nur dort, wo Lesbarkeit davon profitiert. Monitoring- und Tabellenbereiche dürfen die Breite nutzen.
- Mobile Sidebar bleibt Drawer; Content-Panels brechen in klare vertikale Abschnitte um.

### Sidebar

Die Navigation bleibt links, aber die visuelle Hierarchie wird straffer:

- Brand oben kompakt: Logo, Prometheus, kurzer Kontext.
- Suche in der Sidebar bleibt, aber flacher und weniger prominent.
- Gruppen bleiben sinnvoll: Übersicht, Infrastruktur, Intelligenz, Verwaltung, Einstellungen.
- Aktive Items erhalten einen orangefarbenen Indikator plus Flächenzustand.
- Node-Tree wird dichter und lesbarer, mit kleinen Statuspunkten und klaren Untereinträgen.

### Header

Der Header wird optisch leiser:

- Keine pillige Überdekorierung.
- Agent-Status als kleiner, eindeutiger Indikator mit Text nur auf breiten Viewports.
- Notification- und User-Aktionen bleiben rechts.
- Breadcrumbs werden besser als Orientierung genutzt, nicht als Deko.

## Dashboard-Redesign

Das Dashboard wird vom Kartenraster zum Lagezentrum.

### Obere Zone: Lage

Eine kompakte Lagezeile zeigt:

- Gesamtzustand des Clusters.
- Nodes online/offline.
- Workloads aktiv/gesamt.
- Durchschnittliche CPU/Memory-Auslastung.
- Offene kritische Aufmerksamkeitspunkte.

Diese Werte werden nicht als vier gleich laute bunte Cards dargestellt. Stattdessen entsteht eine klare Statusleiste mit einem dominanten Gesundheitszustand und sekundären Kennzahlen.

### Mittlere Zone: Aufmerksamkeit

Eine breite operative Fläche zeigt priorisierte Ereignisse:

- Kritische Alerts.
- Offline Nodes.
- Anomalien.
- Fehlgeschlagene Backups/Migrationen.
- Agent-Empfehlungen.

Ziel: Der Nutzer sieht ohne Scrollen, ob er handeln muss.

### Rechte Zone: Agent und Tasks

Eine schlanke rechte Spalte zeigt:

- Agent-Status und letzte Aktivität.
- Laufende Tasks.
- Nächste geplante Backup-/Scan-/Reflex-Aktionen.

Sie ersetzt nicht den Chat, sondern gibt Betriebsvertrauen.

### Untere Zone: Infrastruktur

Die Server-Flotte bleibt sichtbar, aber kompakter:

- Nodes als scanbare Zeilen oder kompakte Tiles.
- Status, CPU, Memory, Storage, VM/CT Count.
- Klick führt zur Node-Detailseite.

Funktionsbereiche werden reduziert. Statt vier großen Karten nur gezielte Schnellaktionen oder Links, abhängig vom Zustand.

## Komponenten

### Übergreifende Komponenten

Neue oder angepasste Komponenten:

- `OpsStatusBar`: kompakte Lagezeile für Zustand und Kernmetriken.
- `AttentionQueue`: priorisierte Liste aus Alerts, Anomalien, Backups, Tasks und Node-Zuständen.
- `OpsPanel`: allgemeiner Panel-Stil für Dashboard- und Detailbereiche.
- `StatusDot`/`StatusIndicator`: einheitliche Statusanzeige mit Farbe, Label und optionalem Icon.
- `MetricCell`: tabellarische Metrikdarstellung mit Label, Wert, optionalem Trend.

Bestehende `Card`, `KpiCard`, `StatusBadge` und Dashboard-Komponenten werden auf das neue Flächensystem abgestimmt, nicht großflächig ersetzt.

### Dashboard-Komponenten

`DashboardOverview` wird neu gegliedert:

- Datenberechnung bleibt in der Komponente oder einem kleinen lokalen Helper.
- Darstellung wird in kleinere, klare Komponenten ausgelagert.
- `NodeGrid` wird entweder dichter gestylt oder in eine neue kompakte Flottenansicht überführt.

## Datenfluss

Keine Backend-Änderungen sind Teil dieses Redesigns.

Das Dashboard nutzt weiterhin:

- `useNodeStore` für Nodes und Node-Status.
- vorhandene APIs für Agent-Aktivität, Alerts, Tasks und weitere Bereiche, wenn bereits im Frontend verfügbar.
- Falls ein Bereich noch keine Daten hat, wird er als ehrlicher leerer Zustand angezeigt, nicht mit dekorativen Platzhaltern gefüllt.

Live- und Polling-Verhalten bleiben unverändert. Das Redesign darf keine zusätzlichen aggressiven Refresh-Intervalle einführen.

## Fehler, Loading und Empty States

- Loading States werden ruhiger: Skeleton-Reihen statt spinnerlastiger UI.
- Fehlerzustände zeigen klare Aktion: erneut laden, zur Detailseite, Einstellungen prüfen.
- Empty States sind knapp und arbeitsorientiert. Keine erklärenden Marketingtexte.
- Kritische Status dürfen visuell stärker sein, aber immer mit konkretem Label und nächstem Schritt.

## Responsives Verhalten

Desktop:

- Sidebar links fest.
- Dashboard als 12-Spalten-Layout: Lage oben, Aufmerksamkeit breit, Agent/Tasks rechts, Flotte darunter.

Tablet:

- Rechte Spalte wandert unter die Aufmerksamkeit.
- Navigation bleibt Sidebar oder Drawer je nach Breite.

Mobile:

- Header kompakt mit Menü, Titel, Suche.
- Dashboard wird vertikal: Lage, Aufmerksamkeit, Agent/Tasks, Flotte.
- Tabellen werden zu kompakten Zeilenlisten, nicht zu großen Marketing-Cards.

## Umsetzungsscope

Phase 1 soll die sichtbare Designwirkung erzeugen:

1. Design Tokens und globale Surface-Klassen für das SRE-Cockpit.
2. App Shell: Sidebar, Header, Main-Fläche.
3. Dashboard: neue Lage-, Aufmerksamkeit- und Flottenstruktur.
4. Reusable Status-/Panel-Komponenten.
5. Lint/Build und Browserprüfung auf Desktop und Mobile.

Nicht Teil von Phase 1:

- Backend-API-Änderungen.
- Vollständiger Umbau jeder Detailseite.
- Neue Branding-Assets außer einer besseren Logo-/Brand-Behandlung mit bestehenden Icons.
- Neue Charts oder neue Datenmodelle.

## Testing und Qualität

- `npm run lint` im Frontend.
- Wenn möglich `npm run build`, sofern lokale Dependencies und Umgebung das zulassen.
- Browserprüfung im In-App-Browser:
  - Dashboard Desktop.
  - Dashboard Mobile.
  - Sidebar geöffnet/geschlossen auf Mobile.
  - Dark Mode als Standardwirkung.
  - Light Mode grob auf Lesbarkeit.
- Visuelle Checks:
  - Keine Textüberläufe in Buttons, Sidebar und KPI/Statusflächen.
  - Keine Karten in Karten.
  - Statusfarben haben Bedeutung und sind nicht rein dekorativ.
  - Primärer Dashboard-Blick beantwortet: Was braucht Aufmerksamkeit?

## Festlegung

Der gewählte Standard ist die dunkle SRE-Cockpit-Anmutung. Light Mode bleibt erhalten, aber Dark Mode soll die primäre Gestaltung tragen.
