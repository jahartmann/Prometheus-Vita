"use client";

import { useEffect, useRef } from "react";
import { useNetworkStore } from "@/stores/network-store";

interface UseNetworkScanOptions {
  nodeId: string;
  enabled?: boolean;
  pollInterval?: number;
}

export function useNetworkScan({
  nodeId,
  enabled = true,
  pollInterval = 10000,
}: UseNetworkScanOptions) {
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const { fetchScans, fetchDevices, fetchAnomalies, scanStatus } = useNetworkStore();

  useEffect(() => {
    if (!enabled || !nodeId) return;

    // Initial fetch
    fetchScans(nodeId);
    fetchDevices(nodeId);
    fetchAnomalies(nodeId);

    // Poll when scanning
    intervalRef.current = setInterval(() => {
      fetchScans(nodeId);
      if (scanStatus.isScanning) {
        fetchDevices(nodeId);
        fetchAnomalies(nodeId);
      }
    }, pollInterval);

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [nodeId, enabled, pollInterval, fetchScans, fetchDevices, fetchAnomalies, scanStatus.isScanning]);

  return { scanStatus };
}
