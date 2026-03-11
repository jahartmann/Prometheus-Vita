"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { VMPermissionMatrix } from "@/components/settings/vm-permission-matrix";

export default function VMPermissionsPage() {
  const { user } = useAuthStore();
  const router = useRouter();

  useEffect(() => {
    if (user && user.role !== "admin") {
      router.push("/settings/nodes");
    }
  }, [user, router]);

  if (!user || user.role !== "admin") {
    return null;
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">VM-Berechtigungen</h2>
        <p className="text-sm text-muted-foreground">
          Feingranulare Berechtigungen pro Benutzer, VM und Gruppe verwalten.
        </p>
      </div>

      <VMPermissionMatrix />
    </div>
  );
}
