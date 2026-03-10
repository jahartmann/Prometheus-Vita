import Link from "next/link";
import { Flame } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4">
      <Flame className="h-16 w-16 text-primary opacity-20" />
      <h1 className="text-4xl font-bold">404</h1>
      <p className="text-muted-foreground">Diese Seite wurde nicht gefunden.</p>
      <Button asChild>
        <Link href="/">Zurueck zum Dashboard</Link>
      </Button>
    </div>
  );
}
