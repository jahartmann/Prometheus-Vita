# Prometheus UI- und Funktions-Rework Design

Datum: 2026-04-27

## Ziel

Prometheus soll als modernes, klares Operations-Cockpit wirken, ohne bestehende Funktionen zu verlieren. Das Rework konzentriert sich auf vorhandene Bereiche: Dashboard, Notifications inklusive Telegram, Security, Netzwerk-Scans, Logs und Task-Center. Neue grosse Feature-Saeulen sind nicht Teil dieser Phase.

Das Ergebnis soll nicht grau oder starr wirken. Die Oberflaeche bleibt card-basiert, mit hochwertigen Schatten, klaren Flaechen, gezielter Farbe und moderner Typografie. Der Unterschied zur aktuellen Oberflaeche ist die Hierarchie: Cards haben eine Aufgabe, Farben haben eine Bedeutung, und wichtige Aktionen stehen vor dekorativen Kennzahlen.

## Leitprinzipien

- Funktionen bleiben erhalten und werden in echte Workflows eingebettet.
- Cards bleiben ein zentrales Gestaltungsmittel, aber weniger redundant und mit besserer visueller Rangordnung.
- Farben werden bewusst eingesetzt: Akzentfarben fuer Produktcharakter, Statusfarben fuer echte Zustaende.
- Listen und Tabellen werden dort genutzt, wo Vergleichbarkeit wichtiger ist als Praesentation; Card-Grids bleiben fuer Uebersichten und Drilldown erhalten.
- Jede relevante Aktion zeigt Loading-, Success-, Empty- und Error-Zustaende.
- Stille Fehler werden sichtbar gemacht. `catch {}` ohne Nutzerfeedback wird in den betroffenen Flows ersetzt.
- Das Dashboard aggregiert, dupliziert aber keine kompletten Fachseiten.

## Visuelle Richtung

Die gewaehlte Richtung ist "Ruhiges Lagezentrum" in einer modernen, card-basierten Auspraegung.

Die Basis ist neutral und sauber: helle Flaechen, feine Borders, weiche Schatten, 8px Radius, klare Abstaende. Dazu kommen gezielte Akzente, zum Beispiel dunkles Anthrazit fuer Struktur, ein frischer Gruen/Cyan-Akzent fuer operative Signale und Amber/Rot fuer Warnung und Risiko.

Statusfarben werden nicht als grosse, laute Card-Hintergruende verwendet. Stattdessen erscheinen sie als Akzentlinie, Badge, Icon-Flaeche, Progress-Bar oder dezente Toenung innerhalb einer Card. Dadurch wirkt die UI lebendig, aber nicht ueberladen.

## Dashboard-Aufbau

### 1. Hero-Lagekarte

Die erste Card ist ein breites Operations-Panel. Sie zeigt:

- Clusterzustand
- Node-Verfuegbarkeit
- wichtigste Prioritaet
- zwei bis drei Kernmetriken
- direkte Aktion zur wichtigsten Stelle

Wenn keine Probleme existieren, zeigt die Card einen ruhigen, positiven Zustand. Wenn Probleme existieren, nennt sie maximal eine Hauptprioritaet und verlinkt zur passenden Detailseite.

### 2. Priorisierte Action-Cards

Unter der Lagekarte erscheinen maximal drei Action-Cards. Beispiele:

- RAM auf einem Node pruefen
- Telegram-Kanal testen
- Netzwerk-Anomalie bestaetigen
- Log-Analyse starten
- fehlgeschlagene Task oeffnen

Jede Action-Card beantwortet:

- Was ist los?
- Warum ist es wichtig?
- Was kann ich jetzt tun?

### 3. Kompakte KPI-Cards

KPI-Cards bleiben, werden aber konsolidiert. Es gibt keine doppelte Kombination aus Lagebild, KPI-Reihe und sehr aehnlichen Statuskarten. Die KPIs zeigen nur die wichtigsten Betriebswerte.

### 4. Server-Flotte

Die Server-Flotte bleibt visuell als moderne Node-Card-Ansicht erhalten. Die Cards werden ruhiger, mit besseren Progressbars, weniger Text und klareren Schnellaktionen. Eine spaetere Grid/Liste-Umschaltung ist moeglich, aber nicht Voraussetzung fuer Phase 1.

### 5. Funktionsmodule

Dashboard-Module fuer Notifications, Security, Netzwerk, Logs und Tasks zeigen verdichtete Zustaende und fuehren zur jeweiligen Fachseite. Sie ersetzen nicht die Fachseiten.

## Phase-1-Funktionsumfang

### Notifications, Telegram und SMTP

Die Notifications-Seite wird zu einem klaren Setup- und Betriebsflow:

- Status fuer Telegram: verbunden, nicht verbunden, Fehler, letzter Test.
- Status fuer SMTP: konfiguriert, nicht konfiguriert, letzter Test, letzter Fehler.
- Channel-Liste mit echter Testaktion und sichtbarer Rueckmeldung.
- Alert-Regeln, Reflexe, Eskalationen, Incidents und Verlauf bleiben vorhanden, werden aber visuell klarer gruppiert.
- Nach create/update/delete/test werden Daten neu geladen.
- API-Fehler erscheinen als Toast und, wo relevant, inline in der Card.

### Security

Security wird zu einem Bewertungs- und Acknowledge-Workflow:

- Security-Modus bleibt steuerbar.
- Befunde werden als moderne Event-Cards dargestellt.
- Severity, Kategorie, Node, Zeit, Impact, Empfehlung und Metriken bleiben sichtbar.
- Acknowledge zeigt Loading und aktualisiert Event sowie Stats.
- Filter bleiben vorhanden, wirken aber leichter und klarer.

### Netzwerk-Scans

Netzwerk-Scans werden als zusammenhaengender Workflow dargestellt:

- Node-Auswahl bleibt erhalten.
- Quick- und Full-Scan werden klar unterschieden.
- Laufender Scan, letzter erfolgreicher Scan, Scan-Fehler und fehlende Voraussetzungen werden sichtbar.
- Ports, Devices, Anomalien, Services, Historie und Baseline bleiben erreichbar.
- Baseline-Verwaltung wird weniger versteckt, aber nicht dominant.
- Scan-Ergebnisse werden korrekt node-spezifisch geladen und angezeigt.

### Logs

Logs werden vom isolierten Terminal-Look zu einer Operations-Ansicht:

- Node-Auswahl, Log-Quelle, Filter, Auto-Refresh und Aktualisieren bleiben erhalten.
- Severity-Zusammenfassung wird visuell ruhiger.
- Ladefehler pro Node werden sichtbar, statt still ignoriert.
- Analyse, Export, Bookmarks und Quellenverwaltung werden als echte Aktionen erkennbar, soweit die vorhandenen APIs das erlauben.
- Der Log-Viewer bleibt monospaced und gut lesbar, wirkt aber eingebettet statt angeklebt.

### Task-Center

Das Task-Center bleibt die aggregierte Arbeitsliste fuer lange Operationen, Incidents und fehlgeschlagene Benachrichtigungen:

- Status, Quelle, Fortschritt, Detail, Ziel-Link und Zeit bleiben sichtbar.
- Empty-State erklaert, dass keine laufenden oder auffaelligen Operationen offen sind.
- Fehler beim Laden werden sichtbar.
- Schreibende Aktionen bleiben auf Detailseiten, solange es keine sichere zentrale API dafuer gibt.

## Toolchain und Runtime-Voraussetzungen

Prometheus soll die fuer seine bestehenden Funktionen notwendigen Systemtools nicht still voraussetzen.

### Backend-Container

Das Backend-Docker-Image soll die lokalen Diagnose- und Netzwerktools enthalten, die fuer Backend-nahe Checks sinnvoll sind. Dazu gehoeren mindestens:

- `nmap`
- `openssh-client`
- `iproute2`
- `ca-certificates`
- `tzdata`

Falls Alpine-Pakete andere Namen verwenden, wird die Dockerfile-Installation entsprechend angepasst. Der Runtime-Container bleibt nicht-root; Installation geschieht im Image-Build vor `USER prometheus`.

### Proxmox-/Node-Seite

Die vorhandenen Full-Scans laufen per SSH auf dem Ziel-Node und benoetigen dort `nmap`. Quick-Scans nutzen unter anderem `ss`, das normalerweise ueber `iproute2` verfuegbar ist.

Das Rework fuehrt deshalb eine Tool-Preflight-Sicht ein:

- pro Node wird angezeigt, ob `nmap`, `ss`, `journalctl` und relevante Proxmox-Kommandos verfuegbar sind.
- wenn `nmap` fehlt, zeigt die Netzwerk-Seite den Full-Scan nicht als kaputte Funktion, sondern als fehlende Voraussetzung.
- fuer Admin/Operator kann spaeter eine bestaetigungspflichtige Installationsaktion angeboten werden. Diese muss transparent zeigen, welcher Befehl auf welchem Node ausgefuehrt wird.

Automatische Installation auf Proxmox-Nodes erfolgt nicht heimlich. Sie braucht eine explizite Nutzeraktion und passende Berechtigung.

### Optionale Tools

Tools wie `masscan` werden in Phase 1 nicht automatisch eingefuehrt. Sie sind leistungsfaehig, aber riskanter und brauchen eigene Sicherheits- und Rate-Limit-Regeln. Phase 1 konzentriert sich auf `nmap` und vorhandene SSH-/Proxmox-Funktionen.

## Komponenten

Neue oder ueberarbeitete Frontend-Bausteine:

- `PageShell` oder konsistente Page-Header-Konvention fuer Fachseiten.
- `StatusBadge` fuer einheitliche Statussprache.
- `ActionCard` fuer priorisierte Aktionen.
- `OperationsHeroCard` fuer das Dashboard.
- ueberarbeitete `KpiCard` mit weniger harten Farbvarianten.
- `FeatureStatusCard` fuer Telegram, SMTP, Netzwerk-Scan, Logs und Security.
- gemeinsame Empty/Error/Loading-Komponenten, statt uneinheitlicher Textzeilen.

Die vorhandenen shadcn/Radix-Komponenten bleiben die Grundlage. Es wird kein neues UI-Framework eingefuehrt.

## Datenfluss

Frontend-Datenzugriffe laufen weiterhin ueber `src/lib/api.ts` und die bestehenden Stores. Das Rework soll keine zweite API-Schicht erfinden.

Pro Fachbereich wird geprueft:

- welche API gelesen wird,
- welche Aktion schreibt,
- ob nach einer Aktion invalidiert oder neu geladen wird,
- ob Response-Envelopes korrekt entpackt werden,
- ob Empty/Error-Zustaende unterschieden werden.

Das Dashboard aggregiert aus bestehenden Stores/APIs. Wenn ein Wert fuer das Dashboard nicht robust verfuegbar ist, wird er als "nicht verfuegbar" dargestellt statt erfunden.

## Fehlerbehandlung

Alle Phase-1-Flows erhalten sichtbare Fehler:

- Toast fuer akute Aktionsfehler.
- Inline-Fehler in der betroffenen Card, wenn ein Bereich nicht geladen werden kann.
- Retry-Aktion bei ladbaren Bereichen.
- Keine stillen `catch`-Bloecke in den betroffenen Seiten.
- Backend-Fehlertexte werden nutzerfreundlich zusammengefasst, ohne sensible Details zu leaken.

## Sicherheit und Berechtigungen

Die bestehenden Backend-Permissions bleiben fuehrend. Das Frontend zeigt Aktionen nur dann prominent, wenn die Rolle sie voraussichtlich nutzen darf; das Backend bleibt die verbindliche Kontrolle.

Besonders fuer Tool-Installation, SSH-Kommandos, Full-Scans und Security-Modus-Aenderungen gilt:

- keine automatische Ausfuehrung ohne bestaetigte Nutzeraktion,
- klare Anzeige von Ziel-Node und Aktion,
- Audit-Log bleibt relevant,
- destructive oder weitreichende Aktionen brauchen explizite Bestaetigung.

## Tests und Verifikation

Frontend:

- `npm run lint`
- manuelle Browserpruefung fuer Dashboard, Notifications, Network, Logs, Security und Task-Center
- Desktop- und Mobile-Viewport

Backend:

- `go fmt ./...`
- gezielte Go-Tests, wenn Handler, Services oder Tool-Preflight-Logik geaendert werden
- Pruefung, dass Docker-Build mit den neuen Paketen funktioniert

Funktionspruefung:

- Telegram-Status laden und Test ausloesen
- Notification-Channel testen
- Netzwerk Quick-Scan starten
- Full-Scan-Zustand bei vorhandenem und fehlendem `nmap` sichtbar machen
- Security-Befund bestaetigen
- Logs laden, filtern, Auto-Refresh testen
- Task-Center laden und leere/fehlerhafte Zustaende pruefen

## Nicht-Ziele dieser Phase

- Kein kompletter Neubau aller Seiten in einem Schritt.
- Keine neue Designbibliothek.
- Keine heimliche Installation von Tools auf Proxmox-Nodes.
- Kein Ersatz fuer die Proxmox-WebGUI.
- Keine neuen riskanten Scan-Engines wie `masscan` ohne eigenes Sicherheitskonzept.

## Akzeptanzkriterien

- Dashboard wirkt modern, card-basiert, farbig akzentuiert und deutlich weniger ueberladen.
- Die Phase-1-Seiten nutzen konsistentere Cards, Statusanzeigen, Fehlerzustaende und Aktionen.
- Telegram, Notifications, Security, Netzwerk-Scans, Logs und Tasks zeigen echte Daten- und Aktionszustaende.
- Fehlende Systemtools wie `nmap` werden klar erkannt und angezeigt.
- Docker-/Runtime-Voraussetzungen sind dokumentiert und, wo passend, im Backend-Image enthalten.
- Linting und relevante Backend-Checks laufen oder verbleibende Blocker sind konkret dokumentiert.
