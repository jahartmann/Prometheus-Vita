"use client";

import { ReactFlowProvider } from "@xyflow/react";
import { DependencyGraph } from "@/components/vm-cockpit/dependency-graph";

export default function DependenciesPage() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">VM-Abhaengigkeiten</h2>
        <p className="text-sm text-muted-foreground">
          Visualisierung aller VM-Abhaengigkeiten. Klicken Sie auf einen Knoten, um zum VM-Cockpit zu navigieren.
        </p>
      </div>

      <ReactFlowProvider>
        <DependencyGraph fullPage />
      </ReactFlowProvider>
    </div>
  );
}
