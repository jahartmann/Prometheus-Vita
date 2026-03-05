"use client";

import Link from "next/link";
import { Home } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export default function DashboardNotFound() {
  return (
    <div className="flex items-center justify-center p-12">
      <Card className="max-w-md w-full">
        <CardContent className="flex flex-col items-center gap-4 py-8">
          <span className="text-5xl font-bold text-muted-foreground">404</span>
          <div className="text-center">
            <h2 className="text-lg font-semibold">Seite nicht gefunden</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Die angeforderte Seite existiert nicht oder wurde verschoben.
            </p>
          </div>
          <Button asChild variant="outline">
            <Link href="/">
              <Home className="mr-2 h-4 w-4" />
              Zum Dashboard
            </Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
