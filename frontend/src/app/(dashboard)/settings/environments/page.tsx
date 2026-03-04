"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { EnvironmentList } from "@/components/environments/environment-list";
import { EnvironmentForm } from "@/components/environments/environment-form";
import { useEnvironmentStore } from "@/stores/environment-store";

export default function EnvironmentsPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const { fetchEnvironments } = useEnvironmentStore();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Umgebungen</h2>
          <p className="text-sm text-muted-foreground">
            Umgebungen erstellen und Nodes zuweisen.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Umgebung erstellen
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <EnvironmentList />
        </CardContent>
      </Card>

      <EnvironmentForm
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchEnvironments}
      />
    </div>
  );
}
