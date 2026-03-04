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
