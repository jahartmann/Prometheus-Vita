import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-md px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default: "border border-transparent bg-primary text-primary-foreground shadow-sm",
        secondary: "border border-transparent bg-secondary text-secondary-foreground",
        destructive: "border border-transparent bg-destructive text-destructive-foreground shadow-sm",
        outline: "border text-foreground",
        success: "border border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-500/10 dark:text-green-400",
        warning: "border border-orange-200 bg-orange-50 text-orange-700 dark:border-orange-800 dark:bg-orange-500/10 dark:text-orange-400",
        degraded: "border border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-500/10 dark:text-red-400",
        maintenance: "border border-zinc-200 bg-zinc-100 text-zinc-700 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
