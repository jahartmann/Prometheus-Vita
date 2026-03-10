"use client";

import { useEffect, useState } from "react";
import { Loader2, Save, Shield } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { passwordPolicyApi } from "@/lib/api";
import type { PasswordPolicy } from "@/types/api";

export default function PasswordPolicyPage() {
  const [policy, setPolicy] = useState<PasswordPolicy | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  // Form state
  const [minLength, setMinLength] = useState(8);
  const [maxLength, setMaxLength] = useState(128);
  const [requireUppercase, setRequireUppercase] = useState(false);
  const [requireLowercase, setRequireLowercase] = useState(false);
  const [requireDigit, setRequireDigit] = useState(false);
  const [requireSpecial, setRequireSpecial] = useState(false);
  const [disallowUsername, setDisallowUsername] = useState(true);

  useEffect(() => {
    loadPolicy();
  }, []);

  const loadPolicy = async () => {
    setIsLoading(true);
    try {
      const data = await passwordPolicyApi.get();
      setPolicy(data);
      setMinLength(data.min_length);
      setMaxLength(data.max_length);
      setRequireUppercase(data.require_uppercase);
      setRequireLowercase(data.require_lowercase);
      setRequireDigit(data.require_digit);
      setRequireSpecial(data.require_special);
      setDisallowUsername(data.disallow_username);
    } catch {
      setError("Richtlinie konnte nicht geladen werden");
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async () => {
    setError("");
    setSuccess("");

    if (minLength > maxLength) {
      setError("Mindestlaenge darf nicht groesser als Maximallaenge sein");
      return;
    }

    setIsSaving(true);
    try {
      const updated = await passwordPolicyApi.update({
        min_length: minLength,
        max_length: maxLength,
        require_uppercase: requireUppercase,
        require_lowercase: requireLowercase,
        require_digit: requireDigit,
        require_special: requireSpecial,
        disallow_username: disallowUsername,
      });
      setPolicy(updated);
      setSuccess("Passwort-Richtlinie gespeichert");
      setTimeout(() => setSuccess(""), 3000);
    } catch {
      setError("Speichern fehlgeschlagen");
    } finally {
      setIsSaving(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Shield className="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle>Passwort-Richtlinie</CardTitle>
              <CardDescription>
                Legen Sie fest, welche Anforderungen Passwoerter erfuellen muessen.
                Diese gelten fuer alle Benutzer bei Erstellung und Passwortaenderung.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Length */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="minLength">Mindestlaenge</Label>
              <Input
                id="minLength"
                type="number"
                min={1}
                max={128}
                value={minLength}
                onChange={(e) => setMinLength(Number(e.target.value))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="maxLength">Maximallaenge</Label>
              <Input
                id="maxLength"
                type="number"
                min={1}
                max={1024}
                value={maxLength}
                onChange={(e) => setMaxLength(Number(e.target.value))}
              />
            </div>
          </div>

          {/* Character requirements */}
          <div className="space-y-3">
            <Label className="text-base">Zeichenanforderungen</Label>
            <div className="space-y-2">
              <ToggleRow
                label="Grossbuchstaben erforderlich (A-Z)"
                checked={requireUppercase}
                onChange={setRequireUppercase}
              />
              <ToggleRow
                label="Kleinbuchstaben erforderlich (a-z)"
                checked={requireLowercase}
                onChange={setRequireLowercase}
              />
              <ToggleRow
                label="Ziffer erforderlich (0-9)"
                checked={requireDigit}
                onChange={setRequireDigit}
              />
              <ToggleRow
                label="Sonderzeichen erforderlich (!@#$...)"
                checked={requireSpecial}
                onChange={setRequireSpecial}
              />
            </div>
          </div>

          {/* Other rules */}
          <div className="space-y-3">
            <Label className="text-base">Weitere Regeln</Label>
            <div className="space-y-2">
              <ToggleRow
                label="Benutzername im Passwort verbieten"
                checked={disallowUsername}
                onChange={setDisallowUsername}
              />
            </div>
          </div>

          {/* Preview */}
          <div className="rounded-lg border bg-muted/30 px-4 py-3">
            <p className="text-xs font-medium text-muted-foreground mb-1">Vorschau der Anforderung:</p>
            <p className="text-sm">
              Mindestens {minLength} Zeichen
              {requireUppercase && ", 1 Grossbuchstabe"}
              {requireLowercase && ", 1 Kleinbuchstabe"}
              {requireDigit && ", 1 Ziffer"}
              {requireSpecial && ", 1 Sonderzeichen"}
              {disallowUsername && ", kein Benutzername"}
              , max. {maxLength} Zeichen
            </p>
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}
          {success && <p className="text-sm text-green-600">{success}</p>}

          <Button onClick={handleSave} disabled={isSaving}>
            {isSaving ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            Speichern
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function ToggleRow({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-center justify-between rounded-lg border px-4 py-3 transition-colors hover:bg-accent/50">
      <span className="text-sm">{label}</span>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full transition-colors ${
          checked ? "bg-primary" : "bg-muted-foreground/30"
        }`}
      >
        <span
          className={`inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
            checked ? "translate-x-[18px]" : "translate-x-[2px]"
          }`}
        />
      </button>
    </label>
  );
}
