# Netzwerk-Security-UI MVP Design

## Ziel

Prometheus bekommt einen ersten belastbaren Ausbau in Richtung Business-/Enterprise-Sicherheitstool. Der MVP soll sichtbar professioneller wirken und vorhandene Netzwerkfunktionen korrekt nutzbar machen, ohne direkt ein komplettes Vulnerability-Management-System zu versprechen.

Der Fokus liegt auf drei Ergebnissen:

- Die Sidebar ist weniger voll und priorisiert operative Kernbereiche.
- Karten und Statusflächen wirken klarer, kräftiger und hochwertiger.
- Die Netzwerk-Analyse zeigt die realen Backend-Scan-Ergebnisse korrekt an und bündelt sie in einem Security-Cockpit.

## Nicht-Ziele

Dieser MVP enthält noch keine CVE-Datenbank, keine externe Threat-Intelligence, kein IDS/IPS, keine Compliance-Engine und keine aggressiven Subnetz- oder Internet-Scans. Diese Themen bleiben Phase 2, wenn die interne Datenbasis und UX stabil sind.

## Aktueller Befund

### Sidebar

Die Navigation ist fachlich reich, aber zu dicht. Viele Bereiche sind gleich prominent, obwohl sie unterschiedliche Nutzungsfrequenzen haben. Dadurch wirkt die Anwendung schwerer als nötig.

### Cards und Theme

Das bestehende Theme ist stark zinc/grau-basiert. Viele Cards nutzen blasse Hintergründe und wenig visuelle Hierarchie. Statusfarben existieren, werden aber nicht konsequent als Produkt-Sprache eingesetzt.

### Netzwerk-Analyse

Backend-Funktionalität existiert bereits:

- Quick Scan via `ss` liefert `listening_tcp`, `listening_udp` und `established`.
- Full Scan via `nmap` liefert `ports` mit `service_name` und `service_version`.
- Geräte, Ports, Scans, Baselines und Anomalien haben eigene Repositories und Routen.
- Bandbreiten- und Trafficdaten existieren über Metrics-Endpunkte für Cluster, Nodes und VMs.

Die aktuelle Frontend-Porttabelle erwartet teilweise andere Ergebnisformen wie `nmap_results`, `vm_ports`, `service` und `version`. Dadurch können Scan-Ergebnisse fehlen oder falsch dargestellt werden.

## Ansatz

Empfohlen wird ein inkrementeller MVP:

1. Bestehende Scan-Daten korrekt anzeigen.
2. Netzwerkseite zu einem Security-Cockpit verdichten.
3. Sidebar und Cards produktweit aufwerten, aber ohne großen Design-System-Bruch.
4. VM-/Service-Port-Analyse vorbereiten, indem vorhandene VM-Port- und VM-Trafficdaten sichtbar verknüpft werden.

Dieser Ansatz liefert schnell einen sichtbaren Qualitätssprung und schafft die Basis für spätere Enterprise-Funktionen.

## UX-Design

### Sidebar

Die Sidebar bleibt gruppiert, aber sichtbare Einträge werden priorisiert:

- Cockpit: Dashboard, Monitoring, Alerts, Task-Center, Logs.
- Infrastruktur: Cluster, Netzwerk, Speicher, Backups, Migration, Disaster Recovery.
- Sicherheit & KI: Sicherheit, Netzwerk-Security, Drift, VM-Gesundheit, KI-Chat.
- Automation/Wissen: Topologie, Reflex, Tags; seltenere Analysebereiche können in sekundäre Unterpunkte oder über die Suche erreicht werden.
- Einstellungen: Übersicht, Nodes, Benutzer/Rollen, API-Tokens, Audit, Benachrichtigungen.

Die Node-Unterpunkte bleiben erhalten, werden aber kompakter dargestellt. Aktive Bereiche müssen deutlicher erkennbar sein.

### Cards

Cards bekommen eine ruhigere, professionellere Oberfläche:

- Radius maximal 8px.
- Kräftigere Statusakzente für Online, Warning, Critical und Security.
- Weniger blasse Icon-Kacheln.
- Konsistente Card-Struktur mit Header, Content und klarer KPI-Hierarchie.
- Keine verschachtelten Cards für normale Seitenabschnitte.

Das Zielbild ist ein dichtes Operations-/Security-Tool, keine Marketingoberfläche.

### Netzwerk-Cockpit

Die Seite `/network` bekommt oben eine kompakte Lagezeile:

- Letzter Quick Scan.
- Letzter Full Scan.
- Offene Ports gesamt.
- Riskante oder unbekannte Ports.
- Erkannte Geräte.
- Unbestätigte Netzwerk-Anomalien.
- Optional: Cluster- oder Node-Bandbreite aus vorhandenen Metrics.

Darunter bleiben Tabs, werden aber klarer auf Security-Arbeit geschnitten:

- Ports & Dienste.
- Geräte.
- VM-/Service-Analyse.
- Anomalien.
- Historie & Baselines.

## Datenmodell und Datenfluss

### Scan-Ergebnis-Normalisierung

Das Frontend erhält eine kleine Normalisierungsschicht, die verschiedene Backend-Ergebnisformen in ein gemeinsames `PortEntry`-Format bringt:

- Quick Scan:
  - `listening_tcp` -> offene TCP-Ports auf Node.
  - `listening_udp` -> offene UDP-Ports auf Node.
  - `established` -> bestehende Verbindungen, nicht als Listening-Port markieren.
- Full Scan:
  - `ports` -> gescannte Dienste auf Localhost/Node.
  - `service_name` -> `service`.
  - `service_version` -> `version`.
- Zukünftige Formen:
  - `nmap_results` und `vm_ports` bleiben unterstützbar, falls sie später durch Subnetz- oder VM-Scans entstehen.

### Risiko-Einstufung

Der MVP nutzt lokales Heuristik-Scoring im Frontend, später optional Backend-Scoring:

- Hoch: offene Verwaltungs- oder Datenbankports wie SSH, RDP, SMB, PostgreSQL, MySQL, Redis, MongoDB, Proxmox API, Docker.
- Mittel: unbekannte offene Ports oder Dienste ohne Version.
- Niedrig: bekannte Web-/Systemdienste mit erwartbarer Zuordnung.
- Info: etablierte Verbindungen ohne neuen offenen Port.

Diese Einstufung ist bewusst eine Orientierung, kein Vulnerability-Nachweis.

### Bandbreite

Vorhandene Endpunkte werden genutzt:

- Cluster: `/network-summary`.
- Node: `/nodes/:id/network-summary`.
- VM: `/nodes/:id/vms/:vmid/network-summary`.

Im MVP wird daraus eine kompakte Traffic-Übersicht auf `/network`, kein neues Backend-Modul.

### VM-/Service-Analyse

Der MVP verknüpft vorhandene Daten:

- VM-Traffic-Ranking aus Metrics.
- VM-Portdaten aus bestehenden VM-Cockpit-Routen, soweit verfügbar.
- Kritische Dienstnamen und Ports werden sichtbar hervorgehoben.

Wenn die VM-Portdaten nicht clusterweit effizient abrufbar sind, wird die UI zunächst auf ausgewählte Nodes/VMs begrenzt und Phase 2 bekommt einen eigenen Backend-Aggregationsendpunkt.

## Backend-Design

Für den MVP sind nur kleine Backend-Anpassungen vorgesehen:

- Keine neuen Tabellen, falls die vorhandenen Scan-/Port-/Device-Repositories reichen.
- Validierung der `network-scans` Trigger bleibt bestehen.
- Optional: bessere Scan-Status-Rückgabe, falls laufende Scans in der UI nicht zuverlässig erkennbar sind.
- Tests für Scan-Parser und Scheduler-Verhalten, wenn vorhandene Funktionen angepasst werden.

Falls die bestehende Full-Scan-Persistierung nur Localhost-Ports erzeugt, wird das transparent in der UI benannt. Subnetz-Discovery und echte Geräte-Port-Scans bleiben Phase 2.

## Frontend-Design

Betroffene Bereiche:

- `src/components/layout/sidebar.tsx`
- `src/app/(dashboard)/network/page.tsx`
- `src/components/network/port-table.tsx`
- `src/components/network/device-table.tsx`
- `src/components/network/scan-status-bar.tsx`
- `src/stores/network-store.ts`
- `src/lib/api.ts`
- zentrale Card-/KPI-Komponenten und globale CSS-Tokens nach Bedarf

Neue oder geänderte Komponenten:

- `NetworkSecurityOverview`: KPI- und Risikozeile für `/network`.
- `normalizeNetworkScanResults`: Utility für einheitliche Portdaten.
- `ServiceRiskBadge`: konsistente Anzeige für Port-/Dienst-Risiko.
- Optional `VMServiceAnalysis`: MVP-Tab für VM-Traffic und Portbezug.

## Fehler- und Leerzustände

Die Netzwerkseite muss erklären, warum Daten fehlen:

- Kein Node ausgewählt.
- Noch kein Quick Scan.
- Noch kein Full Scan.
- Full Scan nicht verfügbar, weil `nmap` auf dem Node fehlt.
- Scan läuft.
- Scan fehlgeschlagen.

Fehlende Daten dürfen nicht wie "alles sicher" aussehen.

## Tests und Verifikation

Frontend:

- Lint ausführen.
- Normalisierung mit Beispiel-JSON für Quick Scan und Full Scan testen, falls Test-Setup vorhanden ist.
- Manuelle UI-Prüfung im Browser auf Desktop und Mobile.

Backend:

- `go fmt ./...`
- Go-Tests für vorhandene Parser, falls durch Änderungen betroffen.

Manuelle Akzeptanz:

- Quick Scan zeigt Listening TCP/UDP und Established-Verbindungen.
- Full Scan zeigt Ports, Service-Namen und Versionen.
- Riskante Ports werden sichtbar hervorgehoben.
- Sidebar wirkt weniger überladen.
- KPI-Cards wirken kräftiger und bleiben responsiv.

## Phase 2

Nach dem MVP können folgende Enterprise-Funktionen folgen:

- Backend-Aggregationsendpoint für VM-/Service-/Port-Inventar.
- Subnetz-Discovery mit kontrollierten Scan-Profilen.
- CVE- und Versions-Risikoabgleich.
- Policy-basierte Freigaben für erlaubte Ports.
- Reports für Audit, Security Review und Management.
- Zeitreihenbasierte Anomalieerkennung für Bandbreite und Portänderungen.
