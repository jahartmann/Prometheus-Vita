"use client";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { NotificationHistoryEntry, NotificationStatus } from "@/types/api";

interface NotificationHistoryListProps {
  entries: NotificationHistoryEntry[];
  isLoading: boolean;
}

const statusVariant: Record<NotificationStatus, "default" | "secondary" | "destructive" | "outline"> = {
  pending: "outline",
  sent: "default",
  failed: "destructive",
};

const statusLabel: Record<NotificationStatus, string> = {
  pending: "Ausstehend",
  sent: "Gesendet",
  failed: "Fehlgeschlagen",
};

const formatDate = (dateStr?: string | null) => {
  if (!dateStr) return "-";
  return new Date(dateStr).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

export function NotificationHistoryList({
  entries,
  isLoading,
}: NotificationHistoryListProps) {
  return (
    <Card>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Zeitpunkt</TableHead>
              <TableHead>Typ</TableHead>
              <TableHead>Betreff</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Gesendet</TableHead>
              <TableHead>Fehler</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && entries.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  Laden...
                </TableCell>
              </TableRow>
            ) : entries.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  Keine Benachrichtigungen vorhanden.
                </TableCell>
              </TableRow>
            ) : (
              entries.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="text-muted-foreground whitespace-nowrap">
                    {formatDate(entry.created_at)}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{entry.event_type}</Badge>
                  </TableCell>
                  <TableCell className="font-medium max-w-[250px] truncate">
                    {entry.subject}
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant[entry.status]}>
                      {statusLabel[entry.status]}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground whitespace-nowrap">
                    {formatDate(entry.sent_at)}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-xs max-w-[200px] truncate">
                    {entry.error_message || "-"}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
