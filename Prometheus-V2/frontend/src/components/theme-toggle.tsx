import { Monitor, Moon, Sun } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTheme } from "./theme-provider";

export function ThemeToggle({ className }: { className?: string }) {
  const { theme, setTheme } = useTheme();
  const options: Array<{ value: "light" | "dark" | "system"; label: string; icon: typeof Sun }> = [
    { value: "light", label: "Hell", icon: Sun },
    { value: "system", label: "System", icon: Monitor },
    { value: "dark", label: "Dunkel", icon: Moon },
  ];

  return (
    <div className={cn("inline-flex items-center gap-1 rounded-full border border-border bg-card p-1", className)}>
      {options.map((opt) => {
        const Icon = opt.icon;
        const active = theme === opt.value;
        return (
          <button
            key={opt.value}
            type="button"
            onClick={() => setTheme(opt.value)}
            aria-label={opt.label}
            className={cn(
              "inline-flex h-7 w-7 items-center justify-center rounded-full transition",
              active
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            )}
          >
            <Icon className="h-3.5 w-3.5" />
          </button>
        );
      })}
    </div>
  );
}
