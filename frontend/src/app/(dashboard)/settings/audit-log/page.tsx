"use client";

import { useEffect, useState, useCallback } from "react";
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
import { ChevronLeft, ChevronRight } from "lucide-react";

const PAGE_SIZE = 25;

const methodColors: Record<string, string> = {
  GET: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
  POST: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  PUT: "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300",
  DELETE: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
  PATCH: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
};

function statusColor(code: number): string {
  if (code >= 200 && code < 300) return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300";
  if (code >= 400 && code < 500) return "bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-300";
  if (code >= 500) return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300";
  return "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300";
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

export default function AuditLogPage() {
  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [offset, setOffset] = useState(0);
  const [methodFilter, setMethodFilter] = useState<string>("ALL");
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

  const filteredEntries =
    methodFilter === "ALL"
      ? entries
      : entries.filter((e) => e.method === methodFilter);

  const page = Math.floor(offset / PAGE_SIZE) + 1;

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Audit-Log</h2>
        <p className="text-sm text-muted-foreground">
          Protokoll aller API-Anfragen an das Gateway.
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
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Zeitpunkt</TableHead>
              <TableHead>Methode</TableHead>
              <TableHead>Pfad</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>IP-Adresse</TableHead>
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
                  Keine Eintraege vorhanden.
                </TableCell>
              </TableRow>
            ) : (
              filteredEntries.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="text-muted-foreground whitespace-nowrap">
                    {formatTimestamp(entry.created_at)}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="secondary"
                      className={methodColors[entry.method] || ""}
                    >
                      {entry.method}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm max-w-[300px] truncate">
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
                  <TableCell className="text-muted-foreground">
                    {entry.ip_address || "-"}
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
            Zurueck
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
