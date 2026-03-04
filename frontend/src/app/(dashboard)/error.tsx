"use client";

import { useEffect } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("Dashboard error:", error);
  }, [error]);

  return (
    <div className="flex items-center justify-center p-12">
      <Card className="max-w-md w-full">
        <CardContent className="flex flex-col items-center gap-4 py-8">
          <AlertTriangle className="h-10 w-10 text-destructive" />
          <div className="text-center">
            <h2 className="text-lg font-semibold">Fehler aufgetreten</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {error.message || "Ein unerwarteter Fehler ist aufgetreten."}
            </p>
          </div>
          <Button onClick={reset} variant="outline">
            <RefreshCw className="mr-2 h-4 w-4" />
            Erneut versuchen
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
