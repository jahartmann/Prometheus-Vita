import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatBytes(bytes: number, decimals = 1): string {
  if (bytes == null || isNaN(bytes) || bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(decimals))} ${sizes[i]}`;
}

export function formatUptime(seconds: number): string {
  if (seconds == null || isNaN(seconds)) return "0m";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

export function formatPercentage(value: number, decimals = 1): string {
  if (value == null || isNaN(value)) return "0%";
  return `${value.toFixed(decimals)}%`;
}

// Status color classes for consistent usage across the app
export const statusColors = {
  success: "text-green-500 dark:text-green-400",
  error: "text-red-500 dark:text-red-400",
  warning: "text-amber-500 dark:text-amber-400",
  info: "text-blue-500 dark:text-blue-400",
  neutral: "text-muted-foreground",
} as const;

export const statusBgColors = {
  success: "bg-green-500/10 text-green-600 dark:text-green-400",
  error: "bg-red-500/10 text-red-600 dark:text-red-400",
  warning: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  info: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  neutral: "bg-muted text-muted-foreground",
} as const;

export function getStatusColor(isOnline: boolean): string {
  return isOnline ? "text-green-500" : "text-red-500";
}

export function getUsageColor(percentage: number): string {
  if (percentage >= 90) return "text-red-500";
  if (percentage >= 75) return "text-amber-500";
  return "text-green-500";
}

export function getUsageBgColor(percentage: number): string {
  if (percentage >= 90) return "bg-red-500";
  if (percentage >= 75) return "bg-amber-500";
  return "bg-green-500";
}

/**
 * Format bytes/sec rate into human-readable bandwidth string.
 */
export function formatBandwidth(bytesPerSec: number): string {
  if (bytesPerSec == null || isNaN(bytesPerSec) || bytesPerSec === 0) return "0 B/s";
  const k = 1024;
  const sizes = ["B/s", "KB/s", "MB/s", "GB/s", "TB/s"];
  const i = Math.floor(Math.log(Math.abs(bytesPerSec)) / Math.log(k));
  const idx = Math.min(i, sizes.length - 1);
  return `${parseFloat((bytesPerSec / Math.pow(k, idx)).toFixed(1))} ${sizes[idx]}`;
}

/**
 * Format total byte count into human-readable traffic string.
 */
export function formatTraffic(bytes: number): string {
  if (bytes == null || isNaN(bytes) || bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k));
  const idx = Math.min(i, sizes.length - 1);
  return `${parseFloat((bytes / Math.pow(k, idx)).toFixed(1))} ${sizes[idx]}`;
}

/**
 * Convert bytes/sec to Mbit/s.
 */
export function bytesToMbits(bytesPerSec: number): number {
  return (bytesPerSec * 8) / (1024 * 1024);
}
