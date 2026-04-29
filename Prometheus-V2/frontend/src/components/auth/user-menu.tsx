import { LogOut } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/lib/auth/store";
import { logoutRequest } from "@/lib/auth/client";

export function UserMenu() {
  const user = useAuthStore((s) => s.user);
  const clearSession = useAuthStore((s) => s.clearSession);
  const navigate = useNavigate();

  async function handleLogout() {
    try {
      await logoutRequest();
    } finally {
      clearSession();
      navigate({ to: "/login" });
    }
  }

  if (!user) return null;
  return (
    <div className="flex items-center gap-3">
      <div className="text-right hidden sm:block">
        <p className="text-sm font-medium leading-tight">{user.name}</p>
        <p className="text-xs uppercase tracking-wide text-muted-foreground">{user.role}</p>
      </div>
      <Button variant="ghost" size="icon" onClick={handleLogout} aria-label="Abmelden">
        <LogOut className="h-4 w-4" />
      </Button>
    </div>
  );
}
