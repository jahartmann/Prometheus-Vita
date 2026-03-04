"use client";

import { useEffect, useState } from "react";
import { userApi } from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

const autonomyLabels: Record<number, { label: string; description: string }> = {
  0: {
    label: "Nur Lesen",
    description: "Der Agent darf nur lesende Tools verwenden. Schreibende Aktionen werden blockiert.",
  },
  1: {
    label: "Mit Bestaetigung",
    description: "Schreibende Aktionen erfordern eine manuelle Genehmigung vor der Ausfuehrung.",
  },
  2: {
    label: "Voll-Automatisch",
    description: "Der Agent fuehrt alle Aktionen sofort aus, ohne Bestaetigung.",
  },
};

export default function AgentSettingsPage() {
  const user = useAuthStore((s) => s.user);
  const [autonomyLevel, setAutonomyLevel] = useState<number>(1);
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (user?.id) {
      userApi
        .getById(user.id)
        .then((r) => setAutonomyLevel(r.data?.data?.autonomy_level ?? 1));
    }
  }, [user?.id]);

  const handleSave = async (level: number) => {
    if (!user?.id) return;
    setIsSaving(true);
    setSaved(false);
    try {
      await userApi.update(user.id, { autonomy_level: level });
      setAutonomyLevel(level);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">Agent-Einstellungen</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Konfiguriere das Autonomie-Level des KI-Agenten fuer dein Konto.
        </p>
      </div>

      <div className="grid gap-4">
        {[0, 1, 2].map((level) => {
          const info = autonomyLabels[level];
          const isActive = autonomyLevel === level;

          return (
            <Card
              key={level}
              className={`cursor-pointer transition-colors ${
                isActive ? "border-primary ring-1 ring-primary" : "hover:border-muted-foreground/50"
              }`}
              onClick={() => handleSave(level)}
            >
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm font-medium">
                    {info.label}
                  </CardTitle>
                  {isActive && <Badge>Aktiv</Badge>}
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  {info.description}
                </p>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {saved && (
        <p className="text-sm text-green-600">
          Autonomie-Level erfolgreich gespeichert.
        </p>
      )}

      {isSaving && (
        <p className="text-sm text-muted-foreground">Speichere...</p>
      )}
    </div>
  );
}
