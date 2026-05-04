import { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
  variant?: "default" | "loading" | "error";
  className?: string;
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  variant = "default",
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center rounded-md border border-dashed bg-card/40 px-4 py-8 text-center md:py-12",
        variant === "error" && "border-destructive/30 bg-destructive/5",
        className
      )}
    >
      <div
        className={cn(
          "mb-3 rounded-full bg-muted p-3",
          variant === "error" && "bg-destructive/10"
        )}
      >
        <Icon
          className={cn(
            "h-6 w-6 text-muted-foreground",
            variant === "loading" && "animate-spin",
            variant === "error" && "text-destructive"
          )}
        />
      </div>
      <h3 className="text-sm font-medium">{title}</h3>
      {description && (
        <p className="mt-1 text-sm text-muted-foreground max-w-sm">{description}</p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
