import * as React from "react";
import { cn } from "@/lib/utils";

interface OpsPanelProps extends React.HTMLAttributes<HTMLDivElement> {
  interactive?: boolean;
}

export const OpsPanel = React.forwardRef<HTMLDivElement, OpsPanelProps>(
  ({ className, interactive, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "ops-panel",
        interactive && "card-hover",
        className
      )}
      {...props}
    />
  )
);
OpsPanel.displayName = "OpsPanel";

export function OpsPanelHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex flex-col gap-1 border-b ops-divider px-4 py-3",
        className
      )}
      {...props}
    />
  );
}

export function OpsPanelTitle({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h2
      className={cn("text-sm font-semibold tracking-tight", className)}
      {...props}
    />
  );
}

export function OpsPanelDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p
      className={cn("text-xs text-muted-foreground", className)}
      {...props}
    />
  );
}

export function OpsPanelContent({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("p-4", className)} {...props} />;
}
