# UI Function Rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first phase of the Prometheus UI and function rework: a modern card-based operations cockpit, visible toolchain readiness, and working status/error/action flows for Dashboard, Notifications/Telegram, Network Scans, Logs, Security, and Task-Center.

**Architecture:** Keep the existing Next.js App Router, Zustand stores, shadcn/Radix components, Echo handlers, and service/repository layering. Add small shared UI components and a backend node tool-preflight API, then refactor each feature page to consume the same status/error/action patterns.

**Tech Stack:** Go 1.23, Echo v4, pgx, Redis, Next.js 15, React 19, TypeScript, Zustand, Tailwind CSS v4, shadcn/Radix, lucide-react, Docker Alpine.

---

## Scope Check

This plan intentionally covers one cohesive Phase 1 because the features share the same design system, status language, and API error model. Work must still be committed task-by-task. If a worker cannot complete a feature task in one focused pass, split that task into a follow-up plan before editing unrelated areas.

## File Map

- Modify: `Dockerfile.backend` to include backend runtime tools.
- Create: `backend/internal/service/node/tool_preflight.go` for node-side tool checks through existing SSH execution.
- Create: `backend/internal/service/node/tool_preflight_test.go` for command construction and output parsing.
- Modify: `backend/internal/api/handler/node_handler.go` to expose tool preflight data.
- Modify: `backend/internal/api/router.go` to register the node-scoped preflight route.
- Modify: `frontend/src/types/api.ts` to add tool preflight and feature status types.
- Modify: `frontend/src/lib/api.ts` to add node tool-preflight API methods and a shared error helper.
- Modify: `frontend/src/app/globals.css` to tune tokens, shadows, and status utilities.
- Create: `frontend/src/components/ui/status-badge.tsx`.
- Create: `frontend/src/components/ui/action-card.tsx`.
- Create: `frontend/src/components/ui/feature-status-card.tsx`.
- Create: `frontend/src/components/layout/page-shell.tsx`.
- Modify: `frontend/src/components/ui/kpi-card.tsx`.
- Modify: `frontend/src/app/(dashboard)/page.tsx`.
- Modify: `frontend/src/components/dashboard/dashboard-overview.tsx`.
- Modify: `frontend/src/components/dashboard/node-card.tsx`.
- Modify: `frontend/src/components/dashboard/attention-banner.tsx`.
- Modify: `frontend/src/app/(dashboard)/settings/notifications/page.tsx`.
- Modify: `frontend/src/components/notifications/telegram-link-card.tsx`.
- Modify: `frontend/src/components/notifications/smtp-config-card.tsx`.
- Modify: `frontend/src/stores/notification-store.ts`.
- Modify: `frontend/src/stores/network-store.ts`.
- Modify: `frontend/src/app/(dashboard)/network/page.tsx`.
- Modify: `frontend/src/components/network/scan-status-bar.tsx`.
- Modify: `frontend/src/app/(dashboard)/logs/page.tsx`.
- Modify: `frontend/src/stores/log-store.ts`.
- Modify: `frontend/src/app/(dashboard)/security/page.tsx`.
- Modify: `frontend/src/app/(dashboard)/task-center/page.tsx`.

---

### Task 1: Backend Runtime Tools And Node Tool Preflight

**Files:**
- Modify: `Dockerfile.backend`
- Create: `backend/internal/service/node/tool_preflight.go`
- Create: `backend/internal/service/node/tool_preflight_test.go`
- Modify: `backend/internal/api/handler/node_handler.go`
- Modify: `backend/internal/api/router.go`
- Modify: `frontend/src/types/api.ts`
- Modify: `frontend/src/lib/api.ts`

- [ ] **Step 1: Update backend runtime packages**

Modify `Dockerfile.backend` runtime stage package installation:

```dockerfile
RUN apk add --no-cache ca-certificates tzdata nmap openssh-client iproute2
```

Expected result: the backend image has local diagnostics and SSH client tooling without changing the non-root runtime user.

- [ ] **Step 2: Write the failing preflight unit test**

Create `backend/internal/service/node/tool_preflight_test.go`:

```go
package node

import (
	"context"
	"strings"
	"testing"

	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

type preflightRunner struct {
	commands []string
	output   string
	err      error
}

func (r *preflightRunner) RunSSHCommand(ctx context.Context, nodeID uuid.UUID, command string) (*ssh.CommandResult, error) {
	r.commands = append(r.commands, command)
	return &ssh.CommandResult{Stdout: r.output, ExitCode: 0}, r.err
}

func TestParseToolPreflightOutput(t *testing.T) {
	output := "nmap|/usr/bin/nmap\nss|/usr/sbin/ss\njournalctl|\npct|/usr/sbin/pct\nqm|/usr/sbin/qm\n"

	checks := parseToolPreflightOutput(output, []ToolDefinition{
		{Name: "nmap", Command: "command -v nmap"},
		{Name: "ss", Command: "command -v ss"},
		{Name: "journalctl", Command: "command -v journalctl"},
		{Name: "pct", Command: "command -v pct"},
		{Name: "qm", Command: "command -v qm"},
	})

	if len(checks) != 5 {
		t.Fatalf("expected 5 checks, got %d", len(checks))
	}
	if !checks[0].Available || checks[0].Path != "/usr/bin/nmap" {
		t.Fatalf("expected nmap to be available with path, got %+v", checks[0])
	}
	if checks[2].Available {
		t.Fatalf("expected journalctl to be unavailable, got %+v", checks[2])
	}
}

func TestRunToolPreflightUsesSingleShellCommand(t *testing.T) {
	runner := &preflightRunner{output: "nmap|/usr/bin/nmap\nss|/usr/sbin/ss\njournalctl|/usr/bin/journalctl\npct|/usr/sbin/pct\nqm|/usr/sbin/qm\n"}
	nodeID := uuid.New()

	result, err := RunToolPreflight(context.Background(), runner, nodeID)
	if err != nil {
		t.Fatalf("RunToolPreflight returned error: %v", err)
	}
	if result.NodeID != nodeID {
		t.Fatalf("expected node id %s, got %s", nodeID, result.NodeID)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("expected one SSH command, got %d", len(runner.commands))
	}
	if !strings.Contains(runner.commands[0], "command -v nmap") {
		t.Fatalf("expected nmap check in command, got %q", runner.commands[0])
	}
	if len(result.Tools) != 5 {
		t.Fatalf("expected five tools, got %d", len(result.Tools))
	}
}
```

- [ ] **Step 3: Run the failing backend test**

Run:

```powershell
go test ./internal/service/node -run ToolPreflight
```

Expected: FAIL because `RunToolPreflight`, `ToolDefinition`, and `parseToolPreflightOutput` do not exist.

- [ ] **Step 4: Implement node tool preflight service**

Create `backend/internal/service/node/tool_preflight.go`:

```go
package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/google/uuid"
)

type SSHCommandRunner interface {
	RunSSHCommand(ctx context.Context, nodeID uuid.UUID, command string) (*ssh.CommandResult, error)
}

type ToolDefinition struct {
	Name    string
	Command string
}

type ToolCheck struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
}

type ToolPreflightResult struct {
	NodeID uuid.UUID   `json:"node_id"`
	Tools  []ToolCheck `json:"tools"`
}

var defaultToolDefinitions = []ToolDefinition{
	{Name: "nmap", Command: "command -v nmap"},
	{Name: "ss", Command: "command -v ss"},
	{Name: "journalctl", Command: "command -v journalctl"},
	{Name: "pct", Command: "command -v pct"},
	{Name: "qm", Command: "command -v qm"},
}

func RunToolPreflight(ctx context.Context, runner SSHCommandRunner, nodeID uuid.UUID) (*ToolPreflightResult, error) {
	var parts []string
	for _, tool := range defaultToolDefinitions {
		parts = append(parts, fmt.Sprintf("printf '%s|'; %s 2>/dev/null || true", tool.Name, tool.Command))
	}
	command := strings.Join(parts, "; printf '\\n'; ")
	result, err := runner.RunSSHCommand(ctx, nodeID, command)
	if err != nil {
		return nil, fmt.Errorf("run tool preflight: %w", err)
	}
	return &ToolPreflightResult{
		NodeID: nodeID,
		Tools:  parseToolPreflightOutput(result.Stdout, defaultToolDefinitions),
	}, nil
}

func parseToolPreflightOutput(output string, definitions []ToolDefinition) []ToolCheck {
	paths := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, path, ok := strings.Cut(line, "|")
		if !ok {
			continue
		}
		paths[name] = strings.TrimSpace(path)
	}

	checks := make([]ToolCheck, 0, len(definitions))
	for _, definition := range definitions {
		path := paths[definition.Name]
		checks = append(checks, ToolCheck{
			Name:      definition.Name,
			Available: path != "",
			Path:      path,
		})
	}
	return checks
}
```

- [ ] **Step 5: Add handler method**

In `backend/internal/api/handler/node_handler.go`, add imports if missing:

```go
nodeSvc "github.com/antigravity/prometheus/internal/service/node"
```

Add method near other node diagnostic handlers:

```go
func (h *NodeHandler) GetToolPreflight(c echo.Context) error {
	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "invalid node id")
	}

	result, err := nodeSvc.RunToolPreflight(c.Request().Context(), h.service, nodeID)
	if err != nil {
		slog.Warn("node tool preflight failed", slog.String("node_id", nodeID.String()), slog.Any("error", err))
		return response.InternalError(c, "failed to check node tools")
	}
	return response.Success(c, result)
}
```

- [ ] **Step 6: Register route**

In `backend/internal/api/router.go`, register after `nodes.GET("/:id/diagnose", ...)`:

```go
nodes.GET("/:id/tools/preflight", h.Node.GetToolPreflight, middleware.RequirePermission(model.PermissionNodesRead))
```

- [ ] **Step 7: Add frontend types and API method**

Append to `frontend/src/types/api.ts` near node types:

```ts
export interface ToolCheck {
  name: "nmap" | "ss" | "journalctl" | "pct" | "qm" | string;
  available: boolean;
  path?: string;
}

export interface ToolPreflightResult {
  node_id: string;
  tools: ToolCheck[];
}
```

Modify `frontend/src/lib/api.ts`:

```ts
export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === "object" && "response" in error) {
    const response = (error as { response?: { data?: { message?: string; error?: string } } }).response;
    return response?.data?.message ?? response?.data?.error ?? fallback;
  }
  if (error instanceof Error) return error.message;
  return fallback;
}
```

Add to `nodeApi`:

```ts
getToolPreflight: (nodeId: string) => api.get(`/nodes/${nodeId}/tools/preflight`),
```

- [ ] **Step 8: Verify backend task**

Run:

```powershell
cd backend
go fmt ./...
go test ./internal/service/node -run ToolPreflight
```

Expected: PASS for `ToolPreflight` tests.

- [ ] **Step 9: Commit backend preflight**

```powershell
git add Dockerfile.backend backend/internal/service/node/tool_preflight.go backend/internal/service/node/tool_preflight_test.go backend/internal/api/handler/node_handler.go backend/internal/api/router.go frontend/src/types/api.ts frontend/src/lib/api.ts
git commit -m "feat: add node tool preflight"
```

---

### Task 2: Shared Modern UI Components And Tokens

**Files:**
- Modify: `frontend/src/app/globals.css`
- Modify: `frontend/src/components/ui/kpi-card.tsx`
- Create: `frontend/src/components/ui/status-badge.tsx`
- Create: `frontend/src/components/ui/action-card.tsx`
- Create: `frontend/src/components/ui/feature-status-card.tsx`
- Create: `frontend/src/components/layout/page-shell.tsx`

- [ ] **Step 1: Tune global tokens and utility classes**

In `frontend/src/app/globals.css`, adjust root variables to keep a clean neutral base and add reusable utilities:

```css
@layer utilities {
  .surface-panel {
    @apply rounded-lg border bg-card text-card-foreground shadow-sm;
    box-shadow: 0 1px 2px oklch(0 0 0 / 0.04), 0 12px 28px oklch(0 0 0 / 0.05);
  }

  .surface-panel-strong {
    @apply rounded-lg border bg-card text-card-foreground;
    box-shadow: 0 18px 44px oklch(0 0 0 / 0.08);
  }

  .status-accent-ok {
    @apply border-l-4 border-l-emerald-500;
  }

  .status-accent-warning {
    @apply border-l-4 border-l-amber-500;
  }

  .status-accent-critical {
    @apply border-l-4 border-l-red-500;
  }

  .status-accent-info {
    @apply border-l-4 border-l-sky-500;
  }
}
```

Keep existing `.card-hover`; update hover shadow to be softer:

```css
.card-hover:hover {
  transform: translateY(-1px);
  box-shadow: 0 14px 32px oklch(0 0 0 / 0.08);
}
```

- [ ] **Step 2: Add `StatusBadge`**

Create `frontend/src/components/ui/status-badge.tsx`:

```tsx
import { AlertTriangle, CheckCircle2, Clock, Info, XCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export type StatusTone = "ok" | "warning" | "critical" | "info" | "muted";

const toneClasses: Record<StatusTone, string> = {
  ok: "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/25 dark:text-emerald-300",
  warning: "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/25 dark:text-amber-300",
  critical: "border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-300",
  info: "border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/25 dark:text-sky-300",
  muted: "border-border bg-muted text-muted-foreground",
};

const toneIcons = {
  ok: CheckCircle2,
  warning: AlertTriangle,
  critical: XCircle,
  info: Info,
  muted: Clock,
};

interface StatusBadgeProps {
  tone: StatusTone;
  children: React.ReactNode;
  className?: string;
  withIcon?: boolean;
}

export function StatusBadge({ tone, children, className, withIcon = true }: StatusBadgeProps) {
  const Icon = toneIcons[tone];
  return (
    <Badge variant="outline" className={cn("gap-1.5 rounded-full px-2.5 py-1 font-medium", toneClasses[tone], className)}>
      {withIcon && <Icon className="h-3.5 w-3.5" />}
      {children}
    </Badge>
  );
}
```

- [ ] **Step 3: Add `ActionCard`**

Create `frontend/src/components/ui/action-card.tsx`:

```tsx
import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { StatusTone } from "@/components/ui/status-badge";
import { StatusBadge } from "@/components/ui/status-badge";

interface ActionCardProps {
  tone: StatusTone;
  icon: LucideIcon;
  title: string;
  description: string;
  badge: string;
  href: string;
  actionLabel: string;
}

const accentClass: Record<StatusTone, string> = {
  ok: "status-accent-ok",
  warning: "status-accent-warning",
  critical: "status-accent-critical",
  info: "status-accent-info",
  muted: "border-l-4 border-l-border",
};

export function ActionCard({ tone, icon: Icon, title, description, badge, href, actionLabel }: ActionCardProps) {
  return (
    <Card hover className={cn("overflow-hidden", accentClass[tone])}>
      <CardContent className="flex h-full flex-col gap-4 p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            <div className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
              <Icon className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <h3 className="text-sm font-semibold">{title}</h3>
              <p className="mt-1 text-sm text-muted-foreground">{description}</p>
            </div>
          </div>
          <StatusBadge tone={tone}>{badge}</StatusBadge>
        </div>
        <Button asChild variant="outline" size="sm" className="mt-auto w-fit">
          <Link href={href}>
            {actionLabel}
            <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 4: Add `FeatureStatusCard`**

Create `frontend/src/components/ui/feature-status-card.tsx`:

```tsx
import type { LucideIcon } from "lucide-react";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusBadge, type StatusTone } from "@/components/ui/status-badge";

interface FeatureStatusCardProps {
  title: string;
  description: string;
  icon: LucideIcon;
  tone: StatusTone;
  status: string;
  details?: React.ReactNode;
  actionLabel?: string;
  onAction?: () => void;
  isActionPending?: boolean;
  error?: string | null;
}

export function FeatureStatusCard({
  title,
  description,
  icon: Icon,
  tone,
  status,
  details,
  actionLabel,
  onAction,
  isActionPending,
  error,
}: FeatureStatusCardProps) {
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 pb-3">
        <div className="flex items-start gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted">
            <Icon className="h-5 w-5" />
          </div>
          <div>
            <CardTitle className="text-base">{title}</CardTitle>
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          </div>
        </div>
        <StatusBadge tone={tone}>{status}</StatusBadge>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {details}
        {error && <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-300">{error}</p>}
        {actionLabel && onAction && (
          <Button variant="outline" size="sm" className="w-fit" onClick={onAction} disabled={isActionPending}>
            {isActionPending && <RefreshCw className="mr-2 h-4 w-4 animate-spin" />}
            {actionLabel}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 5: Add `PageShell`**

Create `frontend/src/components/layout/page-shell.tsx`:

```tsx
import { cn } from "@/lib/utils";

interface PageShellProps {
  title: string;
  description?: string;
  eyebrow?: string;
  actions?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}

export function PageShell({ title, description, eyebrow, actions, children, className }: PageShellProps) {
  return (
    <div className={cn("flex flex-col gap-5", className)}>
      <header className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          {eyebrow && <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{eyebrow}</p>}
          <h1 className="text-2xl font-bold tracking-tight">{title}</h1>
          {description && <p className="mt-1 max-w-3xl text-sm text-muted-foreground">{description}</p>}
        </div>
        {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
      </header>
      {children}
    </div>
  );
}
```

- [ ] **Step 6: Update `KpiCard`**

In `frontend/src/components/ui/kpi-card.tsx`, keep the public props but tune icon colors and shadows:

```tsx
const colorClasses: Record<string, string> = {
  blue: "bg-sky-500/12 text-sky-600 dark:text-sky-300",
  green: "bg-emerald-500/12 text-emerald-600 dark:text-emerald-300",
  orange: "bg-amber-500/12 text-amber-700 dark:text-amber-300",
  red: "bg-red-500/12 text-red-600 dark:text-red-300",
  purple: "bg-violet-500/12 text-violet-600 dark:text-violet-300",
  neutral: "bg-muted text-muted-foreground",
};
```

Use `rounded-lg` card default and avoid custom nested card patterns.

- [ ] **Step 7: Verify UI component task**

Run:

```powershell
cd frontend
npm run lint
```

Expected: lint passes or reports existing project lint command issue. If `next lint` is unsupported for Next 15 in this project, record the exact error and use `npx tsc --noEmit` as secondary verification.

- [ ] **Step 8: Commit shared UI**

```powershell
git add frontend/src/app/globals.css frontend/src/components/ui/kpi-card.tsx frontend/src/components/ui/status-badge.tsx frontend/src/components/ui/action-card.tsx frontend/src/components/ui/feature-status-card.tsx frontend/src/components/layout/page-shell.tsx
git commit -m "feat: add operations ui primitives"
```

---

### Task 3: Modern Dashboard Lagezentrum

**Files:**
- Modify: `frontend/src/app/(dashboard)/page.tsx`
- Modify: `frontend/src/components/dashboard/dashboard-overview.tsx`
- Modify: `frontend/src/components/dashboard/node-card.tsx`
- Modify: `frontend/src/components/dashboard/attention-banner.tsx`

- [ ] **Step 1: Replace page header with hero card composition**

In `frontend/src/app/(dashboard)/page.tsx`, remove the bordered header and use a modern hero section with `StatusBadge`. Keep existing `fetchNodes()`.

Use this structure:

```tsx
<section className="surface-panel-strong overflow-hidden p-5">
  <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
    <div>
      <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Prometheus Operations</p>
      <h1 className="mt-1 text-3xl font-bold tracking-tight">Lagezentrum</h1>
      <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
        Prioritaeten, Clusterzustand und naechste Aktionen in einer ruhigen Operations-Ansicht.
      </p>
    </div>
    <div className="flex flex-wrap gap-2">
      <StatusBadge tone={isHealthy ? "ok" : "warning"}>{isHealthy ? "Cluster operativ" : `${offlineNodes} offline`}</StatusBadge>
      <StatusBadge tone="muted" withIcon={false}>{onlineNodes}/{nodes.length} Nodes online</StatusBadge>
    </div>
  </div>
</section>
```

- [ ] **Step 2: Make `AttentionBanner` produce modern action cards**

Refactor `frontend/src/components/dashboard/attention-banner.tsx` to use `ActionCard`.

Map severities:

```ts
const toneBySeverity = {
  critical: "critical",
  warning: "warning",
  info: "info",
} as const;
```

For generated items include a route:

```ts
href: item.severity === "critical" ? "/alerts" : "/monitoring"
```

Render at most three action cards:

```tsx
<section className="grid gap-3 lg:grid-cols-3">
  {sortedItems.slice(0, 3).map((item) => (
    <ActionCard
      key={item.title}
      tone={toneBySeverity[item.severity]}
      icon={item.severity === "info" ? Info : AlertTriangle}
      title={item.title}
      description={item.description}
      badge={severityMeta[item.severity].label}
      href={item.href}
      actionLabel="Oeffnen"
    />
  ))}
</section>
```

For empty state render one calm card, not a full-width alert band.

- [ ] **Step 3: Consolidate `DashboardOverview`**

In `frontend/src/components/dashboard/dashboard-overview.tsx`:

- remove the duplicate `Lagebild` section,
- keep one KPI row,
- keep `NodeGrid`,
- add small feature summary cards linking to `/settings/notifications`, `/network`, `/logs`, `/task-center`.

Use `FeatureStatusCard` for the summaries with read-only details:

```tsx
<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
  <FeatureStatusCard title="Notifications" description="Kanaele und Eskalationen" icon={Bell} tone="info" status="Pruefen" details={<p className="text-sm text-muted-foreground">Telegram, SMTP und Verlauf</p>} />
  <FeatureStatusCard title="Netzwerk" description="Scans und Baselines" icon={Network} tone="info" status="Bereit" details={<p className="text-sm text-muted-foreground">Ports, Devices, Anomalien</p>} />
  <FeatureStatusCard title="Logs" description="Analyse und Live-Sicht" icon={FileText} tone="muted" status="Live" details={<p className="text-sm text-muted-foreground">Filter, Export, Analyse</p>} />
  <FeatureStatusCard title="Tasks" description="Offene Operationen" icon={ListChecks} tone="muted" status="Queue" details={<p className="text-sm text-muted-foreground">Migrationen, Backups, Incidents</p>} />
</div>
```

- [ ] **Step 4: Tune `NodeCard` visual density**

In `frontend/src/components/dashboard/node-card.tsx`:

- remove `space-y-1`,
- use `flex flex-col gap-1`,
- lower icon block visual weight,
- keep CPU/RAM/Disk bars,
- keep VM/CT counts,
- keep link target unchanged.

Use this `UsageBar` outer wrapper:

```tsx
<div className="flex flex-col gap-1">
```

Use softer card header:

```tsx
<CardHeader className="pb-3">
```

Do not remove any data currently visible in the node card.

- [ ] **Step 5: Verify dashboard**

Run:

```powershell
cd frontend
npm run lint
```

Then run the frontend:

```powershell
npm run dev
```

Open `/` in the browser and verify:

- one hero Lagezentrum card is visible,
- no duplicate Lagebild/KPI repetition,
- at most three priority action cards,
- node cards remain clickable,
- mobile viewport does not overlap text.

- [ ] **Step 6: Commit dashboard**

```powershell
git add frontend/src/app/(dashboard)/page.tsx frontend/src/components/dashboard/dashboard-overview.tsx frontend/src/components/dashboard/node-card.tsx frontend/src/components/dashboard/attention-banner.tsx
git commit -m "feat: redesign operations dashboard"
```

---

### Task 4: Notifications, Telegram, And SMTP Flow

**Files:**
- Modify: `frontend/src/stores/notification-store.ts`
- Modify: `frontend/src/app/(dashboard)/settings/notifications/page.tsx`
- Modify: `frontend/src/components/notifications/telegram-link-card.tsx`
- Modify: `frontend/src/components/notifications/smtp-config-card.tsx`

- [ ] **Step 1: Replace generic notification errors**

In `frontend/src/stores/notification-store.ts`, import helper:

```ts
import { notificationApi, alertApi, toArray, getApiErrorMessage } from "@/lib/api";
```

Replace each catch message with:

```ts
const message = getApiErrorMessage(err, "Kanaele konnten nicht geladen werden");
```

Use German ASCII strings:

- `Kanaele konnten nicht geladen werden`
- `Verlauf konnte nicht geladen werden`
- `Alert-Regeln konnten nicht geladen werden`

- [ ] **Step 2: Wrap notifications page in `PageShell`**

In `frontend/src/app/(dashboard)/settings/notifications/page.tsx`, import:

```tsx
import { PageShell } from "@/components/layout/page-shell";
```

Replace the top `<div className="space-y-4">` header with:

```tsx
<PageShell
  title="Benachrichtigungen"
  description="Kanaele, Telegram, SMTP, Alert-Regeln, Reflexe und Eskalationen als Betriebsflow."
  eyebrow="Settings"
>
```

Close with `</PageShell>`.

- [ ] **Step 3: Create status grid above tabs**

Place `SmtpConfigCard` and `TelegramLinkCard` in a responsive grid:

```tsx
<section className="grid gap-4 xl:grid-cols-2">
  <SmtpConfigCard channels={channels} onSaved={() => { fetchChannels(); fetchHistory(); }} />
  <TelegramLinkCard />
</section>
```

Remove their old standalone stacking.

- [ ] **Step 4: Make Telegram card show status, action, loading, and error**

In `frontend/src/components/notifications/telegram-link-card.tsx`:

- use `FeatureStatusCard`,
- load `telegramApi.status()` on mount,
- show `linked`, `is_verified`, `bot_enabled`, username, bot username,
- on link/test/unlink show pending state,
- use `getApiErrorMessage`.

Required behavior:

```tsx
const tone = !status?.bot_enabled ? "warning" : status.linked && status.is_verified ? "ok" : "info";
const label = !status?.bot_enabled ? "Bot inaktiv" : status.linked && status.is_verified ? "Verbunden" : "Nicht verbunden";
```

If `telegramApi.link()` returns `verification_code`, show it inline in a monospace pill and explain `/start <code>` without adding a fake success.

- [ ] **Step 5: Make SMTP card use the same status language**

In `frontend/src/components/notifications/smtp-config-card.tsx`:

- compute `smtpChannel` from props,
- render active/inactive status with `StatusBadge`,
- keep existing form fields,
- after save/test call `onSaved()`,
- display inline error below the controls.

- [ ] **Step 6: Verify notifications flow**

Run:

```powershell
cd frontend
npm run lint
```

Manual checks:

- `/settings/notifications` loads,
- Telegram card shows bot disabled/not linked/linked without placeholder copy,
- channel test button disables while pending,
- failed API call shows a visible error,
- tabs still show channels, rules, reflexes, escalation, incidents, history.

- [ ] **Step 7: Commit notifications**

```powershell
git add frontend/src/stores/notification-store.ts frontend/src/app/(dashboard)/settings/notifications/page.tsx frontend/src/components/notifications/telegram-link-card.tsx frontend/src/components/notifications/smtp-config-card.tsx
git commit -m "feat: modernize notification workflows"
```

---

### Task 5: Network Scans With Tool Preflight

**Files:**
- Modify: `frontend/src/stores/network-store.ts`
- Modify: `frontend/src/app/(dashboard)/network/page.tsx`
- Modify: `frontend/src/components/network/scan-status-bar.tsx`

- [ ] **Step 1: Extend network store state**

In `frontend/src/stores/network-store.ts`, import:

```ts
import { networkApi, nodeApi, getApiErrorMessage } from "@/lib/api";
import type { ToolPreflightResult } from "@/types/api";
```

Add to state:

```ts
toolPreflightByNode: Record<string, ToolPreflightResult | undefined>;
errorsByScope: Record<string, string | undefined>;
fetchToolPreflight: (nodeId: string) => Promise<void>;
```

Initialize:

```ts
toolPreflightByNode: {},
errorsByScope: {},
```

- [ ] **Step 2: Store visible errors**

In each catch block, set a scope-specific error:

```ts
const message = getApiErrorMessage(e, "Netzwerkdaten konnten nicht geladen werden");
set((state) => ({
  errorsByScope: { ...state.errorsByScope, scans: message },
}));
```

Use scopes: `scans`, `devices`, `anomalies`, `baselines`, `trigger`, `tools`.

- [ ] **Step 3: Add `fetchToolPreflight`**

Add implementation:

```ts
fetchToolPreflight: async (nodeId) => {
  try {
    const res = await nodeApi.getToolPreflight(nodeId);
    set((state) => ({
      toolPreflightByNode: { ...state.toolPreflightByNode, [nodeId]: res.data },
      errorsByScope: { ...state.errorsByScope, tools: undefined },
    }));
  } catch (e) {
    const message = getApiErrorMessage(e, "Tool-Preflight konnte nicht geladen werden");
    set((state) => ({
      toolPreflightByNode: { ...state.toolPreflightByNode, [nodeId]: undefined },
      errorsByScope: { ...state.errorsByScope, tools: message },
    }));
  }
},
```

- [ ] **Step 4: Load tool preflight on network page**

In `frontend/src/app/(dashboard)/network/page.tsx`, destructure `fetchToolPreflight`, `toolPreflightByNode`, and `errorsByScope`.

When `selectedNodeId` changes:

```tsx
useEffect(() => {
  if (selectedNodeId) {
    fetchBaselines(selectedNodeId);
    fetchToolPreflight(selectedNodeId);
  }
}, [selectedNodeId, fetchBaselines, fetchToolPreflight]);
```

- [ ] **Step 5: Replace dark hardcoded node selector**

Use neutral semantic styles instead of `bg-zinc-900`:

```tsx
<div className="flex flex-wrap items-center gap-2 rounded-lg border bg-card px-3 py-2">
```

Active node:

```tsx
isActive ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:bg-accent hover:text-foreground"
```

- [ ] **Step 6: Show nmap readiness**

Add below `ScanStatusBar`:

```tsx
const toolPreflight = selectedNodeId ? toolPreflightByNode[selectedNodeId] : undefined;
const nmapAvailable = toolPreflight?.tools.find((tool) => tool.name === "nmap")?.available;
```

Render:

```tsx
<FeatureStatusCard
  title="Tool-Preflight"
  description="Prueft Voraussetzungen fuer Quick- und Full-Scans auf dem Node."
  icon={Wrench}
  tone={nmapAvailable ? "ok" : "warning"}
  status={nmapAvailable ? "Full-Scan bereit" : "nmap fehlt"}
  error={errorsByScope.tools}
  details={
    <div className="flex flex-wrap gap-2">
      {toolPreflight?.tools.map((tool) => (
        <StatusBadge key={tool.name} tone={tool.available ? "ok" : "warning"}>
          {tool.name}
        </StatusBadge>
      ))}
    </div>
  }
/>
```

- [ ] **Step 7: Update `ScanStatusBar`**

In `frontend/src/components/network/scan-status-bar.tsx`, add props:

```ts
interface ScanStatusBarProps {
  nodeId: string;
  fullScanAvailable?: boolean;
  fullScanUnavailableReason?: string;
}
```

Disable the Full Scan button when `fullScanAvailable === false` and show the reason:

```tsx
<Button disabled={scanStatus.isScanning || fullScanAvailable === false}>Full Scan</Button>
{fullScanAvailable === false && <p className="text-xs text-amber-600">{fullScanUnavailableReason ?? "nmap ist auf diesem Node nicht verfuegbar."}</p>}
```

- [ ] **Step 8: Verify network flow**

Run:

```powershell
cd frontend
npm run lint
```

Manual checks:

- `/network` node selector uses semantic styling,
- Tool-Preflight card appears,
- Full Scan is disabled when `nmap` is unavailable,
- Quick Scan can still be triggered,
- scan errors are visible.

- [ ] **Step 9: Commit network scans**

```powershell
git add frontend/src/stores/network-store.ts frontend/src/app/(dashboard)/network/page.tsx frontend/src/components/network/scan-status-bar.tsx
git commit -m "feat: show network scan readiness"
```

---

### Task 6: Logs Operations View

**Files:**
- Modify: `frontend/src/stores/log-store.ts`
- Modify: `frontend/src/app/(dashboard)/logs/page.tsx`

- [ ] **Step 1: Improve log store errors**

In `frontend/src/stores/log-store.ts`, import:

```ts
import { logAnalysisApi, logApi, getApiErrorMessage } from "@/lib/api";
```

Replace generic errors:

```ts
const message = getApiErrorMessage(e, "Logs konnten nicht geladen werden");
set({ error: message });
```

Use concrete fallbacks:

- `Log-Anomalien konnten nicht geladen werden`
- `Bookmarks konnten nicht geladen werden`
- `Log-Quellen konnten nicht geladen werden`
- `Logs konnten nicht geladen werden`
- `Log-Anomalie konnte nicht bestaetigt werden`
- `Log-Analyse konnte nicht ausgefuehrt werden`

- [ ] **Step 2: Use `PageShell` for cluster logs**

In `frontend/src/app/(dashboard)/logs/page.tsx`, import `PageShell`, `StatusBadge`, and `FeatureStatusCard`.

Replace top header with:

```tsx
<PageShell
  title="Logs"
  description="Clusterweite Log-Sicht mit Filter, Auto-Refresh und sichtbaren Ladefehlern."
  eyebrow="Operations"
>
```

- [ ] **Step 3: Replace hardcoded dark node selector**

Use:

```tsx
<div className="flex flex-wrap items-center gap-2 rounded-lg border bg-card px-3 py-2">
```

Active/inactive node button classes:

```tsx
selectedNodeIds.includes(node.id)
  ? "bg-primary text-primary-foreground"
  : "bg-muted text-muted-foreground hover:bg-accent hover:text-foreground"
```

- [ ] **Step 4: Convert KPI row to feature status cards**

Replace four raw KPI cards with `FeatureStatusCard` or tuned `KpiCard`. Keep these values:

- errors
- warnings
- critical
- visible lines

Use red only for actual error/critical counts.

- [ ] **Step 5: Track per-node log load failures**

In `ClusterLogsPage`, add state:

```ts
const [loadErrors, setLoadErrors] = useState<Record<string, string>>({});
```

Inside `Promise.allSettled`, collect rejected results:

```ts
const nextErrors: Record<string, string> = {};
results.forEach((result, index) => {
  if (result.status === "rejected") {
    const nodeId = selectedNodeIds[index];
    const node = nodes.find((n) => n.id === nodeId);
    nextErrors[node?.name ?? nodeId] = result.reason instanceof Error ? result.reason.message : "Logs konnten nicht geladen werden";
  }
});
setLoadErrors(nextErrors);
```

Render errors above the log output:

```tsx
{Object.entries(loadErrors).length > 0 && (
  <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
    {Object.entries(loadErrors).map(([nodeName, message]) => (
      <p key={nodeName}><strong>{nodeName}:</strong> {message}</p>
    ))}
  </div>
)}
```

- [ ] **Step 6: Keep terminal output readable but embedded**

Change log output wrapper to:

```tsx
className="min-h-[300px] flex-1 overflow-auto rounded-lg border bg-zinc-950 p-3 font-mono text-sm shadow-inner"
```

Keep severity text colors for log lines.

- [ ] **Step 7: Verify logs**

Run:

```powershell
cd frontend
npm run lint
```

Manual checks:

- `/logs` loads,
- node buttons are readable in light and dark mode,
- failed node log loads show inline errors,
- Auto-Refresh does not duplicate interval timers,
- filter still works with regex fallback.

- [ ] **Step 8: Commit logs**

```powershell
git add frontend/src/stores/log-store.ts frontend/src/app/(dashboard)/logs/page.tsx
git commit -m "feat: modernize log operations view"
```

---

### Task 7: Security And Task-Center Polish

**Files:**
- Modify: `frontend/src/app/(dashboard)/security/page.tsx`
- Modify: `frontend/src/app/(dashboard)/task-center/page.tsx`

- [ ] **Step 1: Wrap Security in `PageShell`**

Replace the custom security header with:

```tsx
<PageShell
  title="Sicherheit"
  description="Befunde, Analysemodus und Bestaetigungen in einem klaren Bewertungsflow."
  eyebrow="Security"
  actions={...existing mode switcher and refresh button...}
>
```

Keep analysis mode switcher and refresh button.

- [ ] **Step 2: Convert Security status banner to a modern card**

Use `FeatureStatusCard` for the overall security status:

```tsx
<FeatureStatusCard
  title={hasEmergency ? "Notfall erkannt" : hasCritical ? "Kritische Befunde" : hasWarnings ? "Offene Befunde" : "Keine offenen Befunde"}
  description={hasEmergency ? "Mindestens ein Notfall braucht sofortige Pruefung." : hasCritical ? "Kritische Befunde brauchen Aufmerksamkeit." : hasWarnings ? "Befunde warten auf Bestaetigung." : "Aktuell liegen keine offenen Befunde vor."}
  icon={hasEmergency || hasCritical ? ShieldAlert : hasWarnings ? Bell : ShieldCheck}
  tone={hasEmergency || hasCritical ? "critical" : hasWarnings ? "warning" : "ok"}
  status={hasEmergency || hasCritical ? "Handeln" : hasWarnings ? "Pruefen" : "Stabil"}
/>
```

- [ ] **Step 3: Soften Security event cards**

Keep existing Collapsible event behavior. Replace heavy colored icon blocks with `bg-muted` plus `StatusBadge` for severity and category. Keep:

- title,
- category,
- severity,
- node link,
- time,
- affected VMs,
- description,
- impact,
- recommendation,
- metrics,
- acknowledge button.

- [ ] **Step 4: Add visible Security fetch error**

Add state:

```ts
const [error, setError] = useState<string | null>(null);
```

In `fetchData` catch:

```ts
setError("Sicherheitsdaten konnten nicht geladen werden");
```

Render:

```tsx
{error && <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</div>}
```

- [ ] **Step 5: Wrap Task-Center in `PageShell`**

In `frontend/src/app/(dashboard)/task-center/page.tsx`, replace top header with `PageShell` and move refresh button into `actions`.

- [ ] **Step 6: Add Task-Center error state**

Add:

```ts
const [error, setError] = useState<string | null>(null);
```

In `load`:

```ts
setError(null);
try {
  const nextTasks = await operationsApi.listTasks({ limit: 80 }) as OperationTask[];
  setTasks(nextTasks);
} catch (e) {
  setTasks([]);
  setError(getApiErrorMessage(e, "Tasks konnten nicht geladen werden"));
}
```

Import `getApiErrorMessage`.

Render the error above the queue when set.

- [ ] **Step 7: Tune Task-Center KPI cards**

Keep four totals, but use `KpiCard` with tones:

- active: blue
- failed: red
- warning: orange
- done: green

No nested cards inside cards.

- [ ] **Step 8: Verify Security and Task-Center**

Run:

```powershell
cd frontend
npm run lint
```

Manual checks:

- `/security` keeps mode switching and acknowledge behavior,
- fetch failures are visible,
- `/task-center` shows loading, empty, error, and task rows,
- task links still navigate to `task.href`.

- [ ] **Step 9: Commit Security and Task-Center**

```powershell
git add frontend/src/app/(dashboard)/security/page.tsx frontend/src/app/(dashboard)/task-center/page.tsx
git commit -m "feat: refine security and task center"
```

---

### Task 8: Full Verification And Cleanup

**Files:**
- Review changed files from Tasks 1-7.
- Modify only files needed to fix verification failures.

- [ ] **Step 1: Run backend formatting and tests**

Run:

```powershell
cd backend
go fmt ./...
go test ./...
```

Expected: formatting completes and tests pass. If existing unrelated tests fail, capture the exact failing package and error.

- [ ] **Step 2: Run frontend checks**

Run:

```powershell
cd frontend
npm run lint
npx tsc --noEmit
```

Expected: lint and TypeScript pass. If `npm run lint` fails because `next lint` is unavailable, record the exact command error and rely on `npx tsc --noEmit` plus browser verification.

- [ ] **Step 3: Build Docker backend image**

Run:

```powershell
docker build -f Dockerfile.backend -t prometheus-backend:ui-function-rework .
```

Expected: image builds and Alpine packages install successfully.

- [ ] **Step 4: Start app for browser verification**

Run:

```powershell
cd frontend
npm run dev
```

Open the app and verify these routes:

- `/`
- `/settings/notifications`
- `/network`
- `/logs`
- `/security`
- `/task-center`

Check desktop and mobile widths.

- [ ] **Step 5: Verify core workflow behavior**

Use a real or development backend where possible:

- Dashboard shows one hero Lagezentrum and no duplicate KPI blocks.
- Telegram status loads; link/test/unlink buttons show pending and errors.
- SMTP status loads; save/test refreshes status.
- Network preflight loads; missing `nmap` disables Full Scan and explains why.
- Quick Scan can be triggered.
- Logs load for selected nodes; per-node failures appear inline.
- Security acknowledge updates the event state.
- Task-Center displays loading, empty, error, and populated states.

- [ ] **Step 6: Remove temporary artifacts**

Ensure these are not tracked:

```powershell
git status --short
```

Expected: no `.superpowers/`, screenshots, browser logs, `.next`, or `tsconfig.tsbuildinfo` changes are staged.

- [ ] **Step 7: Final commit**

If verification fixes were needed:

```powershell
git add <fixed-files>
git commit -m "fix: stabilize ui function rework"
```

If no fixes were needed, do not create an empty commit.

---

## Self-Review

Spec coverage:

- Modern card-based dashboard: Tasks 2 and 3.
- Colorful but clean visual system: Task 2.
- Notifications and Telegram: Task 4.
- Security: Task 7.
- Network scans and toolchain including `nmap`: Tasks 1 and 5.
- Logs: Task 6.
- Task-Center: Task 7.
- Runtime tools in backend container: Task 1.
- No hidden node-side installs: Task 5 only reports availability.
- Verification: Task 8.

Placeholder scan:

- No placeholder markers or vague implementation steps are used.
- Deferred work is stated as out of scope in the design spec, not as an implementation gap in this plan.

Type consistency:

- `ToolCheck` and `ToolPreflightResult` are defined before use.
- `StatusTone` values are consistent across `StatusBadge`, `ActionCard`, and `FeatureStatusCard`.
- API additions use existing `nodeApi` and `networkApi` patterns.
