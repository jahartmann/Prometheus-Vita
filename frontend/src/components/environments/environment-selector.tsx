"use client";

import { useEffect } from "react";
import { useEnvironmentStore } from "@/stores/environment-store";

interface EnvironmentSelectorProps {
  value: string;
  onChange: (value: string) => void;
}

export function EnvironmentSelector({ value, onChange }: EnvironmentSelectorProps) {
  const { environments, fetchEnvironments } = useEnvironmentStore();

  useEffect(() => {
    if (environments.length === 0) {
      fetchEnvironments();
    }
  }, [environments.length, fetchEnvironments]);

  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
    >
      <option value="">Alle Umgebungen</option>
      {environments.map((env) => (
        <option key={env.id} value={env.id}>
          {env.name}
        </option>
      ))}
    </select>
  );
}
