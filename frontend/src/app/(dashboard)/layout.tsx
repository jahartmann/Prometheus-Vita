"use client";

import { AppLayout } from "@/components/layout/app-layout";
import { AnomalyToastListener } from "@/components/layout/anomaly-toast-listener";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout>
      <AnomalyToastListener />
      {children}
    </AppLayout>
  );
}
