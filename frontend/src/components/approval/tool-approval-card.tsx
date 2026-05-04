"use client";

import { useEffect, useState } from "react";
import { approvalApi, getApiErrorMessage } from "@/lib/api";
import type { AgentPendingApproval } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { toast } from "sonner";

export function ToolApprovalCard() {
  const [approvals, setApprovals] = useState<AgentPendingApproval[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const fetchApprovals = () => {
    setIsLoading(true);
    approvalApi
      .listPending()
      .then((data) => {
        setApprovals((data || []) as AgentPendingApproval[]);
      })
      .catch(() => {
        setApprovals([]);
      })
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    fetchApprovals();
    const interval = setInterval(fetchApprovals, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleApprove = async (id: string) => {
    try {
      await approvalApi.approve(id);
      toast.success("Agent-Aktion genehmigt");
      fetchApprovals();
    } catch (err: unknown) {
      toast.error(getApiErrorMessage(err, "Genehmigung fehlgeschlagen"));
    }
  };

  const handleReject = async (id: string) => {
    try {
      await approvalApi.reject(id);
      toast.success("Agent-Aktion abgelehnt");
      fetchApprovals();
    } catch (err: unknown) {
      toast.error(getApiErrorMessage(err, "Ablehnen fehlgeschlagen"));
    }
  };

  if (isLoading && approvals.length === 0) {
    return null;
  }

  if (approvals.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <h3 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        Ausstehende Genehmigungen
      </h3>
      {approvals.map((a) => (
        <Card key={a.id} className="border-amber-500/40 bg-amber-500/5">
          <CardHeader className="pb-2 pt-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">
                {a.tool_name}
              </CardTitle>
              <Badge variant="warning">Ausstehend</Badge>
            </div>
          </CardHeader>
          <CardContent className="pb-3">
            <pre className="mb-3 max-h-40 overflow-auto rounded-md border bg-background/70 p-2 text-xs">
              {JSON.stringify(a.arguments, null, 2)}
            </pre>
            <div className="flex gap-2">
              <Button
                size="sm"
                onClick={() => handleApprove(a.id)}
              >
                Genehmigen
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleReject(a.id)}
              >
                Ablehnen
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
