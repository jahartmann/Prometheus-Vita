"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { TokenList } from "@/components/gateway/token-list";
import { CreateTokenDialog } from "@/components/gateway/create-token-dialog";
import { AuditLog } from "@/components/gateway/audit-log";

export default function APITokensPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">API-Tokens & Gateway</h2>
          <p className="text-sm text-muted-foreground">
            API-Tokens verwalten und Audit-Log einsehen.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Token erstellen
        </Button>
      </div>

      <Card>
        <CardContent className="p-4">
          <TokenList refreshKey={refreshKey} />
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-4">
          <AuditLog />
        </CardContent>
      </Card>

      <CreateTokenDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={() => setRefreshKey((k) => k + 1)}
      />
    </div>
  );
}
