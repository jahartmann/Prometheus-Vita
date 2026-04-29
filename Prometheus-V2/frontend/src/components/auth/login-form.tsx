import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { ErrorState } from "@/components/ui/error-state";
import { loginRequest } from "@/lib/auth/client";
import { useAuthStore } from "@/lib/auth/store";
import { ApiError } from "@/lib/api/client";

export function LoginForm({ onSuccess }: { onSuccess?: () => void }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const setSession = useAuthStore((s) => s.setSession);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setPending(true);
    try {
      const res = await loginRequest({ email, password });
      setSession(res.access_token, res.user);
      onSuccess?.();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.status === 401 ? "Email oder Passwort falsch." : err.message);
      } else {
        setError("Login fehlgeschlagen.");
      }
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 surface-panel-strong p-6 max-w-sm mx-auto">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Anmeldung</h1>
        <p className="mt-1 text-sm text-muted-foreground">Prometheus V2 Operations Cockpit</p>
      </div>
      <label className="flex flex-col gap-1 text-sm">
        Email
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoComplete="username"
          className="h-9 rounded-md border border-border bg-card px-3 text-foreground"
        />
      </label>
      <label className="flex flex-col gap-1 text-sm">
        Passwort
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          autoComplete="current-password"
          className="h-9 rounded-md border border-border bg-card px-3 text-foreground"
        />
      </label>
      {error && <ErrorState message={error} />}
      <Button type="submit" disabled={pending}>{pending ? "Anmelden..." : "Anmelden"}</Button>
    </form>
  );
}
