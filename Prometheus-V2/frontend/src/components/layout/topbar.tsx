import { ThemeToggle } from "@/components/theme-toggle";
import { StatusBadge } from "@/components/ui/status-badge";

export function Topbar() {
  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-card px-5">
      <div className="flex items-center gap-3">
        <StatusBadge tone="ok" withIcon>Live</StatusBadge>
        <p className="text-sm text-muted-foreground">Skeleton-Build</p>
      </div>
      <div className="flex items-center gap-3">
        <ThemeToggle />
      </div>
    </header>
  );
}
