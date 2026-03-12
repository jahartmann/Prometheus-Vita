"use client";

import { useEffect } from "react";
import { HardDrive, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { useVMCockpitStore } from "@/stores/vm-cockpit-store";
import { CockpitError } from "./cockpit-error";

export function SystemDisk() {
  const { disks, isLoadingDisk, fetchDisk, diskError } = useVMCockpitStore();

  useEffect(() => {
    fetchDisk();
  }, [fetchDisk]);

  if (diskError) {
    return <CockpitError {...diskError} onRetry={fetchDisk} />;
  }

  if (isLoadingDisk && disks.length === 0) {
    return (
      <div className="grid gap-4 md:grid-cols-2">
        <Skeleton className="h-40" />
        <Skeleton className="h-40" />
      </div>
    );
  }

  const getProgressColor = (percentStr: string) => {
    const value = parseInt(percentStr, 10);
    if (value > 90) return "bg-red-500";
    if (value > 70) return "bg-orange-500";
    return "bg-primary";
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-end">
        <Button variant="outline" size="sm" onClick={fetchDisk}>
          <RefreshCw className="mr-2 h-3 w-3" />
          Aktualisieren
        </Button>
      </div>

      {disks.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed py-12">
          <p className="text-muted-foreground">
            Keine Speicherinformationen verfuegbar
          </p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {disks.map((disk) => {
            const percent = parseInt(disk.percent, 10) || 0;
            const colorClass = getProgressColor(disk.percent);

            return (
              <Card key={disk.target}>
                <CardHeader className="pb-2">
                  <CardTitle className="flex items-center gap-2 text-sm">
                    <HardDrive className="h-4 w-4" />
                    {disk.target}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="relative">
                    <Progress value={percent} className="h-3" />
                    <div
                      className={`absolute inset-0 h-3 rounded-full ${colorClass} transition-all`}
                      style={{ width: `${Math.min(100, percent)}%` }}
                    />
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">
                      {disk.used} / {disk.size}
                    </span>
                    <span
                      className={`font-bold ${
                        percent > 90
                          ? "text-red-500"
                          : percent > 70
                          ? "text-orange-500"
                          : ""
                      }`}
                    >
                      {disk.percent}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {disk.avail} verfuegbar
                  </p>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
