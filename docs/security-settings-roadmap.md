# Security- und Settings-Roadmap

Diese Datei haelt die offenen Punkte fest, damit die Umsetzung schrittweise und nachvollziehbar weiterlaeuft.

## Phase 1: Basis, bereits begonnen

- [x] Dashboard und Navigation als Operations-Cockpit aufraeumen
- [x] Ollama optional per Docker-Compose-Profil integrieren
- [x] LLM- und Env-Konfiguration dokumentieren
- [x] Erste Validierung fuer lokale Ollama-Discovery haerten

## Phase 2: Rechte- und Sicherheitssystem

- [x] Zentrales Permission-Modell definieren
- [x] Rollen auf Permissions abbilden
- [x] Backend-Helper fuer Permission-Pruefungen ergaenzen
- [x] Riskante Aktionen zusaetzlich ueber konkrete Permissions schuetzen
- [x] API-Key-Scopes an dasselbe Permission-Modell anbinden
- [x] VM-, Node- und Environment-Scopes ausbauen
- [x] Audit-Events fuer alle kritischen Aktionen vereinheitlichen

## Phase 3: Verwaltbare Einstellungen

- [x] Settings-Zentrale mit klaren Kategorien bauen
- [x] Rollen- und Rechte-Matrix in den Einstellungen anzeigen
- [x] Rollenrechte persistieren, in Settings editierbar machen und serverseitig erzwingen
- [x] Benutzerverwaltung weiter ausbauen: Einladungen, Deaktivierung, Session-/Token-Uebersicht
- [x] Agent-, LLM- und Ollama-Konfiguration zentralisieren
- [x] Security-Einstellungen verwaltbar machen
- [x] Benachrichtigungen, API-Tokens, Nodes und Backup/DR einheitlich strukturieren
- [x] Systemstatus und Integrationschecks in den Einstellungen anzeigen

## Phase 4: Agent-Sicherheit

- [x] Agent-Tools nach Risiko und Permission klassifizieren
- [x] Dry-run/Preview fuer Migration, Restore, SSH und File-Write ergaenzen
- [x] Approval-Regeln je Aktionstyp und Risiko-Level konfigurierbar machen
- [x] Vollautomatik nur fuer explizit erlaubte sichere Aktionen zulassen
- [x] Tool-Ausfuehrungen vollstaendig auditieren

## Phase 5: Operative Erweiterungen

- [x] Task-Center fuer lange Operationen bauen
- [x] Infrastructure Flight Recorder ergaenzen
- [x] Root-Cause-Analyse ueber Metriken, Logs, Changes und Topologie ausbauen
- [x] Knowledge Graph fuer Nodes, VMs, Dienste, Ports und Abhaengigkeiten
- [x] Promptbare Reports und Dashboard-Filter ergaenzen

## Phase 6: Backend-Aggregation und Automatisierung

- [x] Einheitliche Task-API fuer Migrationen, Backups, Incidents und Notification-Fehler ergaenzen
- [x] Timeline-/Flight-Recorder-API mit Filterung nach Entity, Severity, Suchtext und Quelle bauen
- [x] RCA-Service mit Evidence, Timeline, Cause-Candidates und optionaler LLM-Zusammenfassung ergaenzen
- [x] Knowledge-Graph serverseitig aus Nodes, VMs, Network-Devices, Ports und VM-Dependencies aggregieren
- [x] Reports ueber Backend-Aggregation erzeugen und optional per lokalem Modell zusammenfassen lassen
- [x] Frontend-Views auf Phase-6-Aggregations-APIs umstellen
- [x] Task-API um Approvals und Scheduled Actions erweitern
- [x] Timeline-Filter um explizite Zeitfenster `from`/`to` ergaenzen
- [x] Task-API um geplante Jobs erweitern
- [x] Globale Queryfilter fuer Audit, Backups, Migrationen, Security, Anomalies und Log-Anomalies vereinheitlichen
- [x] Geplante Reports ausfuehren lassen und Ergebnis-Historie anzeigen
- [x] Ollama-gestuetzte Reports/RCA in der UI mit lokaler Modellwahl verbinden

## Laufende Qualitaetsziele

- [x] Tests fuer neue Backend-Sicherheitslogik ergaenzen
- [x] Frontend-Typecheck vor Abschluss
- [x] Migrations und Env-Dokumentation synchron halten
- [x] Keine kritische Aktion ohne Audit- und Berechtigungspruefung
- [x] Agent-Secrets maskieren und neue LLM-Keys verschluesselt speichern
- [x] Gezieltes Loeschen/Rotieren von LLM-Keys ergaenzen
