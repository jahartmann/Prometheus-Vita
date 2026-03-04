"use client";

import { useEffect, useState } from "react";
import { approvalApi } from "@/lib/api";
import type { AgentPendingApproval } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function ToolApprovalCard() {
  const [approvals, setApprovals] = useState<AgentPendingApproval[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const fetchApprovals = () => {
    setIsLoading(true);
    approvalApi
      .listPending()
      .then((data) => setApprovals((data || []) as AgentPendingApproval[]))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    fetchApprovals();
    const interval = setInterval(fetchApprovals, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleApprove = async (id: string) => {
    await approvalApi.approve(id);
    fetchApprovals();
  };

  const handleReject = async (id: string) => {
    await approvalApi.reject(id);
    fetchApprovals();
  };

  if (isLoading && approvals.length === 0) {
    return null;
  }

  if (approvals.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-muted-foreground">
        Ausstehende Genehmigungen
      </h3>
      {approvals.map((a) => (
        <Card key={a.id} className="border-amber-500/50">
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium">
                {a.tool_name}
              </CardTitle>
              <Badge variant="warning">Ausstehend</Badge>
            </div>
          </CardHeader>
          <CardContent>
            <pre className="text-xs bg-muted p-2 rounded mb-3 overflow-x-auto">
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
