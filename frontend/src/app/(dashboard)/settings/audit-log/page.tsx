"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { gatewayApi, toArray } from "@/lib/api";
import type { AuditLogEntry } from "@/types/api";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ChevronLeft, ChevronRight, Bot } from "lucide-react";

const PAGE_SIZE = 25;

const methodColors: Record<string, string> = {
  GET: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
  POST: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  PUT: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300",
  DELETE: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
  PATCH: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
  AGENT: "bg-violet-100 text-violet-800 dark:bg-violet-900/30 dark:text-violet-300",
};

function statusColor(code: number): string {
  if (code >= 200 && code < 300) return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300";
  if (code >= 400 && code < 500) return "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300";
  if (code >= 500) return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300";
  return "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300";
}

function riskColor(risk?: string): string {
  if (risk === "high") return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300";
  if (risk === "medium") return "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300";
  return "bg-slate-100 text-slate-800 dark:bg-slate-900 dark:text-slate-300";
}

function formatCategory(category?: string): string {
  if (!category) return "";
  return category.replace(/_/g, " ");
}

function formatTimestamp(dateStr: string): string {
  return new Date(dateStr).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function describeAction(method: string, path: string): string {
  if (method === "AGENT") {
    const toolLabels: Record<string, string> = {
      list_nodes: "KI: Nodes abgefragt",
      node_status: "KI: Node-Status geprüft",
      get_vms: "KI: VMs abgefragt",
      get_metrics: "KI: Metriken abgefragt",
      create_backup: "KI: Backup erstellt",
      start_vm: "KI: VM gestartet",
      stop_vm: "KI: VM gestoppt",
      migrate_vm: "KI: VM migriert",
      get_storage: "KI: Storage abgefragt",
      get_network: "KI: Netzwerk abgefragt",
      get_predictions: "KI: Vorhersagen abgefragt",
      get_anomalies: "KI: Anomalien abgefragt",
      get_briefing: "KI: Briefing abgefragt",
      check_drift: "KI: Drift geprüft",
      check_updates: "KI: Updates geprüft",
      rightsizing: "KI: Rightsizing analysiert",
      save_knowledge: "KI: Wissen gespeichert",
      recall_knowledge: "KI: Wissen abgerufen",
      run_ssh_command: "KI: SSH-Befehl ausgeführt",
    };
    return toolLabels[path] || `KI: ${path}`;
  }

  const clean = path.replace(/^\/api\/v1\//, "");

  const patterns: [RegExp, Record<string, string>][] = [
    [/^auth\/login$/, { POST: "Angemeldet" }],
    [/^auth\/logout$/, { POST: "Abgemeldet" }],
    [/^auth\/refresh$/, { POST: "Token erneuert" }],
    [/^auth\/me$/, { GET: "Profil abgerufen" }],
    [/^users$/, { GET: "Benutzerliste abgerufen", POST: "Benutzer erstellt" }],
    [/^users\/[^/]+$/, { GET: "Benutzer angezeigt", PUT: "Benutzer bearbeitet", DELETE: "Benutzer gelöscht" }],
    [/^users\/[^/]+\/password$/, { POST: "Passwort geändert" }],
    [/^nodes$/, { GET: "Nodes abgerufen", POST: "Node hinzugefügt" }],
    [/^nodes\/[^/]+$/, { GET: "Node angezeigt", PUT: "Node bearbeitet", DELETE: "Node entfernt" }],
    [/^nodes\/[^/]+\/status$/, { GET: "Node-Status abgerufen" }],
    [/^nodes\/[^/]+\/vms$/, { GET: "VMs abgerufen" }],
    [/^nodes\/[^/]+\/vms\/[^/]+\/(start|stop|restart|shutdown)$/, { POST: "VM-Aktion ausgeführt" }],
    [/^nodes\/[^/]+\/backup$/, { POST: "Backup erstellt" }],
    [/^nodes\/[^/]+\/backups$/, { GET: "Backups abgerufen" }],
    [/^backups\/[^/]+\/restore$/, { POST: "Backup wiederhergestellt" }],
    [/^nodes\/[^/]+\/metrics/, { GET: "Metriken abgerufen" }],
    [/^nodes\/[^/]+\/network/, { GET: "Netzwerk abgerufen" }],
    [/^nodes\/[^/]+\/storage/, { GET: "Storage abgerufen" }],
    [/^chat$/, { POST: "Chat-Nachricht gesendet" }],
    [/^chat\/conversations/, { GET: "Chat-Verlauf abgerufen", DELETE: "Chat gelöscht" }],
    [/^reflexes$/, { GET: "Reflex-Regeln abgerufen", POST: "Reflex-Regel erstellt" }],
    [/^reflexes\/[^/]+$/, { GET: "Reflex-Regel angezeigt", PUT: "Reflex-Regel bearbeitet", DELETE: "Reflex-Regel gelöscht" }],
    [/^agent\/config$/, { GET: "KI-Einstellungen abgerufen", PUT: "KI-Einstellungen geändert" }],
    [/^notifications/, { GET: "Benachrichtigungen abgerufen", POST: "Benachrichtigung erstellt" }],
    [/^security/, { GET: "Sicherheitsereignisse abgerufen" }],
    [/^gateway\/tokens/, { GET: "API-Tokens abgerufen", POST: "API-Token erstellt", DELETE: "API-Token gelöscht" }],
    [/^gateway\/audit/, { GET: "Audit-Log abgerufen" }],
    [/^drift/, { GET: "Drift-Check abgerufen", POST: "Drift-Check gestartet" }],
    [/^briefing/, { GET: "Briefing abgerufen" }],
    [/^predictions/, { GET: "Vorhersagen abgerufen" }],
    [/^anomalies/, { GET: "Anomalien abgerufen" }],
    [/^topology/, { GET: "Topologie abgerufen" }],
    [/^ssh-keys/, { GET: "SSH-Keys abgerufen", POST: "SSH-Key erstellt", DELETE: "SSH-Key gelöscht" }],
    [/^environments/, { GET: "Environments abgerufen", POST: "Environment erstellt" }],
    [/^tags/, { GET: "Tags abgerufen", POST: "Tag erstellt" }],
    [/^password-policy/, { GET: "Passwort-Policy abgerufen", PUT: "Passwort-Policy geändert" }],
    [/^migrations/, { GET: "Migrationen abgerufen", POST: "Migration gestartet" }],
  ];

  for (const [pattern, methods] of patterns) {
    if (pattern.test(clean)) {
      return methods[method] || `${method} ${clean}`;
    }
  }

  return `${method} ${clean}`;
}

export default function AuditLogPage() {
  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [offset, setOffset] = useState(0);
  const [methodFilter, setMethodFilter] = useState<string>("ALL");
  const [userFilter, setUserFilter] = useState<string>("ALL");
  const [hasMore, setHasMore] = useState(true);

  const fetchEntries = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await gatewayApi.listAuditLog(PAGE_SIZE + 1, offset);
      const data = toArray<AuditLogEntry>(res.data);
      if (data.length > PAGE_SIZE) {
        setHasMore(true);
        setEntries(data.slice(0, PAGE_SIZE));
      } else {
        setHasMore(false);
        setEntries(data);
      }
    } catch {
      setEntries([]);
    } finally {
      setIsLoading(false);
    }
  }, [offset]);

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries]);

  const uniqueUsers = useMemo(() => {
    const users = new Set<string>();
    for (const e of entries) {
      if (e.username) users.add(e.username);
    }
    return Array.from(users).sort();
  }, [entries]);

  const filteredEntries = useMemo(() => {
    return entries.filter((e) => {
      if (methodFilter !== "ALL" && e.method !== methodFilter) return false;
      if (userFilter !== "ALL" && (e.username || "") !== userFilter) return false;
      return true;
    });
  }, [entries, methodFilter, userFilter]);

  const page = Math.floor(offset / PAGE_SIZE) + 1;

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Audit-Log</h2>
        <p className="text-sm text-muted-foreground">
          Protokoll aller API-Anfragen und KI-Agent-Aktionen.
        </p>
      </div>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Methode:</span>
          <Select value={methodFilter} onValueChange={setMethodFilter}>
            <SelectTrigger className="w-[130px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">Alle</SelectItem>
              <SelectItem value="GET">GET</SelectItem>
              <SelectItem value="POST">POST</SelectItem>
              <SelectItem value="PUT">PUT</SelectItem>
              <SelectItem value="DELETE">DELETE</SelectItem>
              <SelectItem value="AGENT">KI-Agent</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Benutzer:</span>
          <Select value={userFilter} onValueChange={setUserFilter}>
            <SelectTrigger className="w-[160px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">Alle</SelectItem>
              {uniqueUsers.map((u) => (
                <SelectItem key={u} value={u}>
                  {u}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Zeitpunkt</TableHead>
              <TableHead>Benutzer</TableHead>
              <TableHead>Aktion</TableHead>
              <TableHead>Pfad</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Dauer</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  Laden...
                </TableCell>
              </TableRow>
            ) : filteredEntries.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  Keine Einträge vorhanden.
                </TableCell>
              </TableRow>
            ) : (
              filteredEntries.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="text-muted-foreground whitespace-nowrap">
                    {formatTimestamp(entry.created_at)}
                  </TableCell>
                  <TableCell className="whitespace-nowrap">
                    {entry.username || (entry.api_token_id ? "API-Token" : "-")}
                  </TableCell>
                  <TableCell>
                    <span className="flex flex-wrap items-center gap-2">
                      <Badge
                        variant="secondary"
                        className={methodColors[entry.method] || ""}
                      >
                        {entry.method}
                      </Badge>
                      {entry.method === "AGENT" && <Bot className="h-4 w-4 text-violet-500" />}
                      <span className="text-sm">{describeAction(entry.method, entry.path)}</span>
                      {entry.request_body?.risk && (
                        <Badge
                          variant="secondary"
                          className={riskColor(entry.request_body.risk)}
                        >
                          {entry.request_body.critical ? "kritisch" : entry.request_body.risk}
                        </Badge>
                      )}
                      {entry.request_body?.category && (
                        <Badge variant="outline">
                          {formatCategory(entry.request_body.category)}
                        </Badge>
                      )}
                    </span>
                  </TableCell>
                  <TableCell className="font-mono text-sm max-w-[300px] truncate text-muted-foreground">
                    {entry.path}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="secondary"
                      className={statusColor(entry.status_code)}
                    >
                      {entry.status_code}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right text-muted-foreground">
                    {entry.duration_ms} ms
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">Seite {page}</p>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            disabled={offset === 0}
          >
            <ChevronLeft className="mr-1 h-4 w-4" />
            Zurück
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setOffset(offset + PAGE_SIZE)}
            disabled={!hasMore}
          >
            Weiter
            <ChevronRight className="ml-1 h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
