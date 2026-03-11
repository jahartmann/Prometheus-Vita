# VM Cockpit Phase 1: Shell + System Tab

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a VM detail page with integrated browser terminal (xterm.js) and live system introspection (processes, services, ports, disk).

**Architecture:** New WebSocket-based shell handler proxies terminal I/O between browser and Proxmox (pct exec for LXC, qm guest exec for QEMU). System tab executes commands via same mechanism, parses structured output, returns typed JSON. New VM permission model gates access.

**Tech Stack:** xterm.js + xterm-addon-fit + xterm-addon-search (frontend), gorilla/websocket (backend, already installed), pct exec / qm guest exec (Proxmox).

---

## File Structure

### Backend — New Files
| File | Responsibility |
|------|---------------|
| `backend/migrations/044_create_vm_permissions.sql` | VM permissions table |
| `backend/internal/model/vm_permission.go` | VMPermission model + constants |
| `backend/internal/repository/vm_permission_repository.go` | CRUD for vm_permissions |
| `backend/internal/service/vm/permission_service.go` | Permission checking logic |
| `backend/internal/api/handler/vm_cockpit_handler.go` | WebSocket shell + system REST endpoints |
| `backend/internal/api/handler/vm_permission_handler.go` | Permission CRUD REST endpoints |
| `backend/internal/proxmox/exec.go` | Command execution abstraction (pct exec / qm guest exec) |

### Backend — Modified Files
| File | Change |
|------|--------|
| `backend/internal/api/router.go` | Register vm cockpit + permission routes |
| `backend/cmd/server/main.go` | Wire new handlers + services |
| `backend/internal/service/node/node_service.go` | Add ExecVMCommand method |

### Frontend — New Files
| File | Responsibility |
|------|---------------|
| `frontend/src/app/(dashboard)/nodes/[id]/vms/[vmid]/page.tsx` | VM detail page (entry point) |
| `frontend/src/components/vm-cockpit/vm-cockpit.tsx` | Main tabbed cockpit container |
| `frontend/src/components/vm-cockpit/shell-tab.tsx` | xterm.js terminal with multi-session |
| `frontend/src/components/vm-cockpit/system-tab.tsx` | System overview container |
| `frontend/src/components/vm-cockpit/system-processes.tsx` | Process list |
| `frontend/src/components/vm-cockpit/system-services.tsx` | Systemd service list |
| `frontend/src/components/vm-cockpit/system-ports.tsx` | Open ports |
| `frontend/src/components/vm-cockpit/system-disk.tsx` | Disk usage |
| `frontend/src/stores/vm-cockpit-store.ts` | VM cockpit state |
| `frontend/src/hooks/use-vm-shell.ts` | WebSocket shell hook |
| `frontend/src/lib/vm-api.ts` | VM cockpit API calls |

### Frontend — Modified Files
| File | Change |
|------|--------|
| `frontend/src/types/api.ts` | Add VMPermission, VMProcess, VMService, VMPort types |
| `frontend/src/components/nodes/vm-list.tsx` | Add click-through link to VM detail page |

---

## Chunk 1: Database + Permissions Backend

### Task 1: VM Permissions Migration

**Files:**
- Create: `backend/migrations/044_create_vm_permissions.sql`

- [ ] **Step 1: Write migration**

```sql
CREATE TABLE IF NOT EXISTS vm_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(10) NOT NULL CHECK (target_type IN ('vm', 'group')),
    target_id VARCHAR(50) NOT NULL,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vm_permissions_user ON vm_permissions(user_id);
CREATE INDEX idx_vm_permissions_target ON vm_permissions(target_type, target_id, node_id);
CREATE UNIQUE INDEX idx_vm_permissions_unique ON vm_permissions(user_id, target_type, target_id, node_id);
```

- [ ] **Step 2: Commit**

```bash
git add backend/migrations/044_create_vm_permissions.sql
git commit -m "feat(db): add vm_permissions table for fine-grained VM access control"
```

---

### Task 2: VM Permission Model

**Files:**
- Create: `backend/internal/model/vm_permission.go`

- [ ] **Step 1: Write model**

```go
package model

import (
	"time"
	"github.com/google/uuid"
)

const (
	PermVMView           = "vm.view"
	PermVMShell          = "vm.shell"
	PermVMFilesRead      = "vm.files.read"
	PermVMFilesWrite     = "vm.files.write"
	PermVMSystemView     = "vm.system.view"
	PermVMSystemService  = "vm.system.service"
	PermVMSystemKill     = "vm.system.kill"
	PermVMSystemPackages = "vm.system.packages"
	PermVMPower          = "vm.power"
	PermVMSnapshots      = "vm.snapshots"
	PermVMAIProactive    = "vm.ai.proactive"
)

var AllVMPermissions = []string{
	PermVMView, PermVMShell, PermVMFilesRead, PermVMFilesWrite,
	PermVMSystemView, PermVMSystemService, PermVMSystemKill,
	PermVMSystemPackages, PermVMPower, PermVMSnapshots, PermVMAIProactive,
}

type VMPermission struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	NodeID      uuid.UUID `json:"node_id"`
	Permissions []string  `json:"permissions"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/model/vm_permission.go
git commit -m "feat(model): add VMPermission model with permission constants"
```

---

### Task 3: VM Permission Repository

**Files:**
- Create: `backend/internal/repository/vm_permission_repository.go`

- [ ] **Step 1: Write repository**

```go
package repository

import (
	"context"
	"fmt"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VMPermissionRepository interface {
	Create(ctx context.Context, perm *model.VMPermission) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.VMPermission, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.VMPermission, error)
	ListByTarget(ctx context.Context, targetType, targetID string, nodeID uuid.UUID) ([]model.VMPermission, error)
	Update(ctx context.Context, perm *model.VMPermission) error
	Delete(ctx context.Context, id uuid.UUID) error
	HasPermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, vmid string, permission string) (bool, error)
}

type pgVMPermissionRepository struct {
	db *pgxpool.Pool
}

func NewVMPermissionRepository(db *pgxpool.Pool) VMPermissionRepository {
	return &pgVMPermissionRepository{db: db}
}

func (r *pgVMPermissionRepository) Create(ctx context.Context, perm *model.VMPermission) error {
	perm.ID = uuid.New()
	_, err := r.db.Exec(ctx,
		`INSERT INTO vm_permissions (id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
		perm.ID, perm.UserID, perm.TargetType, perm.TargetID, perm.NodeID, perm.Permissions, perm.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create vm permission: %w", err)
	}
	return nil
}

func (r *pgVMPermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.VMPermission, error) {
	var p model.VMPermission
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at
		 FROM vm_permissions WHERE id = $1`, id,
	).Scan(&p.ID, &p.UserID, &p.TargetType, &p.TargetID, &p.NodeID, &p.Permissions, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get vm permission: %w", err)
	}
	return &p, nil
}

func (r *pgVMPermissionRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.VMPermission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at
		 FROM vm_permissions WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list vm permissions by user: %w", err)
	}
	defer rows.Close()
	var perms []model.VMPermission
	for rows.Next() {
		var p model.VMPermission
		if err := rows.Scan(&p.ID, &p.UserID, &p.TargetType, &p.TargetID, &p.NodeID, &p.Permissions, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vm permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *pgVMPermissionRepository) ListByTarget(ctx context.Context, targetType, targetID string, nodeID uuid.UUID) ([]model.VMPermission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, target_type, target_id, node_id, permissions, created_by, created_at, updated_at
		 FROM vm_permissions WHERE target_type = $1 AND target_id = $2 AND node_id = $3 ORDER BY created_at`,
		targetType, targetID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list vm permissions by target: %w", err)
	}
	defer rows.Close()
	var perms []model.VMPermission
	for rows.Next() {
		var p model.VMPermission
		if err := rows.Scan(&p.ID, &p.UserID, &p.TargetType, &p.TargetID, &p.NodeID, &p.Permissions, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vm permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *pgVMPermissionRepository) Update(ctx context.Context, perm *model.VMPermission) error {
	_, err := r.db.Exec(ctx,
		`UPDATE vm_permissions SET permissions = $1, updated_at = NOW() WHERE id = $2`,
		perm.Permissions, perm.ID)
	if err != nil {
		return fmt.Errorf("update vm permission: %w", err)
	}
	return nil
}

func (r *pgVMPermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM vm_permissions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete vm permission: %w", err)
	}
	return nil
}

func (r *pgVMPermissionRepository) HasPermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, vmid string, permission string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM vm_permissions
			WHERE user_id = $1 AND node_id = $2
			  AND ((target_type = 'vm' AND target_id = $3) OR target_type = 'group')
			  AND $4 = ANY(permissions)
		)`, userID, nodeID, vmid, permission,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check vm permission: %w", err)
	}
	return exists, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/repository/vm_permission_repository.go
git commit -m "feat(repo): add VMPermission repository with HasPermission check"
```

---

### Task 4: Permission Service

**Files:**
- Create: `backend/internal/service/vm/permission_service.go`

- [ ] **Step 1: Write service**

```go
package vm

import (
	"context"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/google/uuid"
)

type PermissionService struct {
	repo     repository.VMPermissionRepository
	userRepo repository.UserRepository
}

func NewPermissionService(repo repository.VMPermissionRepository, userRepo repository.UserRepository) *PermissionService {
	return &PermissionService{repo: repo, userRepo: userRepo}
}

// CheckPermission returns true if user has the given permission on the VM.
// Admins always have all permissions.
func (s *PermissionService) CheckPermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, vmid string, perm string) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user.Role == model.RoleAdmin {
		return true, nil
	}
	return s.repo.HasPermission(ctx, userID, nodeID, vmid, perm)
}

func (s *PermissionService) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.VMPermission, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *PermissionService) Grant(ctx context.Context, perm *model.VMPermission) error {
	return s.repo.Create(ctx, perm)
}

func (s *PermissionService) Revoke(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *PermissionService) Update(ctx context.Context, perm *model.VMPermission) error {
	return s.repo.Update(ctx, perm)
}

func (s *PermissionService) ListByTarget(ctx context.Context, targetType, targetID string, nodeID uuid.UUID) ([]model.VMPermission, error) {
	return s.repo.ListByTarget(ctx, targetType, targetID, nodeID)
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/service/vm/permission_service.go
git commit -m "feat(service): add VM permission service with admin bypass"
```

---

## Chunk 2: Proxmox Exec + Cockpit Handler

### Task 5: Proxmox Command Execution Abstraction

**Files:**
- Create: `backend/internal/proxmox/exec.go`

- [ ] **Step 1: Write exec abstraction**

```go
package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ExecResult struct {
	ExitCode int    `json:"exitcode"`
	OutData  string `json:"out-data"`
	ErrData  string `json:"err-data"`
}

// ExecCommand runs a command inside a VM/container via Proxmox API.
func (c *Client) ExecCommand(ctx context.Context, node string, vmid int, vmType string, command []string) (*ExecResult, error) {
	if vmType == "lxc" {
		return c.execLXC(ctx, node, vmid, command)
	}
	return c.execQEMU(ctx, node, vmid, command)
}

func (c *Client) execLXC(ctx context.Context, node string, vmid int, command []string) (*ExecResult, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/status/exec", node, vmid)
	params := url.Values{}
	params.Set("command", strings.Join(command, " "))
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		return nil, fmt.Errorf("exec lxc command: %w", err)
	}
	var result ExecResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse exec result: %w", err)
	}
	return &result, nil
}

func (c *Client) execQEMU(ctx context.Context, node string, vmid int, command []string) (*ExecResult, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec", node, vmid)
	params := url.Values{}
	params.Set("command", command[0])
	for i, arg := range command[1:] {
		params.Set(fmt.Sprintf("arg%d", i), arg)
	}
	data, err := c.doRequestWithBody(ctx, http.MethodPost, path, params)
	if err != nil {
		return nil, fmt.Errorf("exec qemu command: %w", err)
	}
	var pidResp struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal(data, &pidResp); err != nil {
		return nil, fmt.Errorf("parse exec pid: %w", err)
	}
	return c.waitExecResult(ctx, node, vmid, pidResp.PID)
}

func (c *Client) waitExecResult(ctx context.Context, node string, vmid int, pid int) (*ExecResult, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec-status?pid=%d", node, vmid, pid)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("exec timeout after 30s")
		case <-ticker.C:
			data, err := c.doRequest(ctx, http.MethodGet, path)
			if err != nil {
				return nil, fmt.Errorf("poll exec status: %w", err)
			}
			var status struct {
				Exited   int    `json:"exited"`
				ExitCode int    `json:"exitcode"`
				OutData  string `json:"out-data"`
				ErrData  string `json:"err-data"`
			}
			if err := json.Unmarshal(data, &status); err != nil {
				return nil, fmt.Errorf("parse exec status: %w", err)
			}
			if status.Exited == 1 {
				return &ExecResult{
					ExitCode: status.ExitCode,
					OutData:  status.OutData,
					ErrData:  status.ErrData,
				}, nil
			}
		}
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/proxmox/exec.go
git commit -m "feat(proxmox): add unified command execution for LXC and QEMU VMs"
```

---

### Task 6: VM Cockpit Handler (REST Endpoints)

**Files:**
- Create: `backend/internal/api/handler/vm_cockpit_handler.go`

- [ ] **Step 1: Write handler with system introspection endpoints**

The handler implements: ExecCommand, GetProcesses, GetServices, GetPorts, GetDiskUsage, ServiceAction, KillProcess. Each method checks VM permissions, executes the appropriate command via `nodeSvc.ExecVMCommand()`, parses the structured output, and returns typed JSON.

Key patterns:
- Constructor: `NewVMCockpitHandler(nodeSvc, permSvc, jwtSvc, allowedOrigins)`
- Each method extracts userID from context, nodeID and vmid from path params
- Permission check via `permSvc.CheckPermission()`
- Command execution via `nodeSvc.ExecVMCommand()`
- Output parsing: `parseProcesses()` (ps aux), `parseServices()` (systemctl), `parsePorts()` (ss -tlnp), `parseDisk()` (df -h)
- Response types: VMProcess, VMService, VMPort, VMDisk structs defined in handler

- [ ] **Step 2: Add ExecVMCommand to node service**

In `backend/internal/service/node/node_service.go`, add:
```go
func (s *Service) ExecVMCommand(ctx context.Context, nodeID uuid.UUID, vmid int, vmType string, command []string) (*proxmox.ExecResult, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	client := s.createClient(node)
	return client.ExecCommand(ctx, node.Name, vmid, vmType, command)
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/api/handler/vm_cockpit_handler.go backend/internal/service/node/node_service.go
git commit -m "feat(api): add VM cockpit handler with shell exec and system introspection"
```

---

### Task 7: VM Permission Handler + Route Registration

**Files:**
- Create: `backend/internal/api/handler/vm_permission_handler.go`
- Modify: `backend/internal/api/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Write permission handler**

Standard CRUD handler: List (by user_id query param), Create, Update, Delete.

- [ ] **Step 2: Add routes in router.go**

```go
// VM Cockpit
vmCockpit := v1.Group("/nodes/:nodeId/vms/:vmid/cockpit")
vmCockpit.POST("/exec", h.VMCockpit.ExecCommand, RoleOperator)
vmCockpit.GET("/processes", h.VMCockpit.GetProcesses, RoleOperator)
vmCockpit.GET("/services", h.VMCockpit.GetServices, RoleOperator)
vmCockpit.GET("/ports", h.VMCockpit.GetPorts, RoleOperator)
vmCockpit.GET("/disk", h.VMCockpit.GetDiskUsage, RoleOperator)
vmCockpit.POST("/services/action", h.VMCockpit.ServiceAction, RoleOperator)
vmCockpit.POST("/processes/kill", h.VMCockpit.KillProcess, RoleOperator)
e.GET("/api/v1/nodes/:nodeId/vms/:vmid/cockpit/shell", h.VMCockpit.HandleShell)

// VM Permissions (Admin only)
vmPerms := v1.Group("/vm-permissions", RoleAdmin)
vmPerms.GET("", h.VMPermission.List)
vmPerms.POST("", h.VMPermission.Create)
vmPerms.PUT("/:id", h.VMPermission.Update)
vmPerms.DELETE("/:id", h.VMPermission.Delete)
```

- [ ] **Step 3: Wire in main.go**

```go
vmPermRepo := repository.NewVMPermissionRepository(dbPool)
vmPermSvc := vm.NewPermissionService(vmPermRepo, userRepo)
vmCockpitHandler := handler.NewVMCockpitHandler(nodeSvc, vmPermSvc, jwtSvc, cfg.AllowedOrigins)
vmPermHandler := handler.NewVMPermissionHandler(vmPermSvc)
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/api/handler/vm_permission_handler.go backend/internal/api/router.go backend/cmd/server/main.go
git commit -m "feat(api): wire VM cockpit and permission routes"
```

---

## Chunk 3: Frontend — Dependencies + Types + API

### Task 8: Install xterm.js

- [ ] **Step 1: Install packages**

```bash
cd frontend && npm install xterm @xterm/addon-fit @xterm/addon-search
```

- [ ] **Step 2: Commit**

```bash
git add package.json package-lock.json
git commit -m "deps: add xterm.js with fit and search addons"
```

---

### Task 9: Types + API Client

**Files:**
- Modify: `frontend/src/types/api.ts`
- Create: `frontend/src/lib/vm-api.ts`

- [ ] **Step 1: Add types**

Append to `frontend/src/types/api.ts`:
```typescript
export interface VMPermission {
  id: string;
  user_id: string;
  target_type: "vm" | "group";
  target_id: string;
  node_id: string;
  permissions: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface VMProcess {
  user: string;
  pid: number;
  cpu: number;
  mem: number;
  vsz: string;
  rss: string;
  command: string;
}

export interface VMServiceInfo {
  unit: string;
  load_state: string;
  active_state: string;
  sub_state: string;
  description: string;
}

export interface VMPort {
  protocol: string;
  address: string;
  port: number;
  process: string;
}

export interface VMDisk {
  target: string;
  size: string;
  used: string;
  avail: string;
  percent: string;
}

export interface VMExecResult {
  exitcode: number;
  "out-data": string;
  "err-data": string;
}
```

- [ ] **Step 2: Create vm-api.ts**

```typescript
import api from "@/lib/api";
import type { VMProcess, VMServiceInfo, VMPort, VMDisk, VMExecResult, VMPermission } from "@/types/api";

export const vmCockpitApi = {
  exec: (nodeId: string, vmid: number, command: string, type = "lxc") =>
    api.post<{ data: VMExecResult }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/exec?type=${type}`, { command }),

  getProcesses: (nodeId: string, vmid: number, type = "lxc") =>
    api.get<{ data: VMProcess[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/processes?type=${type}`),

  getServices: (nodeId: string, vmid: number, type = "lxc") =>
    api.get<{ data: VMServiceInfo[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/services?type=${type}`),

  getPorts: (nodeId: string, vmid: number, type = "lxc") =>
    api.get<{ data: VMPort[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/ports?type=${type}`),

  getDisk: (nodeId: string, vmid: number, type = "lxc") =>
    api.get<{ data: VMDisk[] }>(`/nodes/${nodeId}/vms/${vmid}/cockpit/disk?type=${type}`),

  serviceAction: (nodeId: string, vmid: number, service: string, action: string, type = "lxc") =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/cockpit/services/action?type=${type}`, { service, action }),

  killProcess: (nodeId: string, vmid: number, pid: number, signal = "TERM", type = "lxc") =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/cockpit/processes/kill?type=${type}`, { pid, signal }),
};

export const vmPermissionApi = {
  list: (userId: string) => api.get<{ data: VMPermission[] }>(`/vm-permissions?user_id=${userId}`),
  create: (perm: Partial<VMPermission>) => api.post("/vm-permissions", perm),
  update: (id: string, perm: Partial<VMPermission>) => api.put(`/vm-permissions/${id}`, perm),
  delete: (id: string) => api.delete(`/vm-permissions/${id}`),
};
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/types/api.ts frontend/src/lib/vm-api.ts
git commit -m "feat(frontend): add VM cockpit types and API client"
```

---

## Chunk 4: Frontend — Store + Shell Tab

### Task 10: VM Cockpit Store

**Files:**
- Create: `frontend/src/stores/vm-cockpit-store.ts`

- [ ] **Step 1: Write store**

Zustand store with: setVM(), fetchProcesses(), fetchServices(), fetchPorts(), fetchDisk(), killProcess(), serviceAction(). Each fetch method calls the corresponding vmCockpitApi method and updates state.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/stores/vm-cockpit-store.ts
git commit -m "feat(store): add VM cockpit Zustand store"
```

---

### Task 11: WebSocket Shell Hook + Shell Tab

**Files:**
- Create: `frontend/src/hooks/use-vm-shell.ts`
- Create: `frontend/src/components/vm-cockpit/shell-tab.tsx`

- [ ] **Step 1: Write WebSocket hook**

Custom hook `useVMShell({ nodeId, vmid, vmType })` that manages a WebSocket connection to `/api/v1/nodes/{nodeId}/vms/{vmid}/cockpit/shell`. Returns: connect(), disconnect(), send(), setOnData(), isConnected.

- [ ] **Step 2: Write shell tab**

xterm.js-based component with:
- Multi-session support (up to 4 terminal tabs)
- Terminal theme matching dark UI (bg: #09090b)
- Session tab bar with add/close buttons
- Connection status badge
- FitAddon for auto-resize via ResizeObserver
- SearchAddon for text search
- Terminal container cleared via DOM API before re-attaching

- [ ] **Step 3: Commit**

```bash
git add frontend/src/hooks/use-vm-shell.ts frontend/src/components/vm-cockpit/shell-tab.tsx
git commit -m "feat(frontend): add Shell tab with xterm.js multi-session terminal"
```

---

## Chunk 5: Frontend — System Tab Components

### Task 12: System Sub-Components

**Files:**
- Create: `frontend/src/components/vm-cockpit/system-processes.tsx`
- Create: `frontend/src/components/vm-cockpit/system-services.tsx`
- Create: `frontend/src/components/vm-cockpit/system-ports.tsx`
- Create: `frontend/src/components/vm-cockpit/system-disk.tsx`
- Create: `frontend/src/components/vm-cockpit/system-tab.tsx`

- [ ] **Step 1: Write system-processes.tsx**

Table with columns: PID, User, CPU%, MEM%, Command. Filter input. Kill button with ConfirmDialog. Auto-fetch on mount. CPU > 50% highlighted red, MEM > 50% highlighted orange.

- [ ] **Step 2: Write system-services.tsx**

Table with columns: Service, Status (Badge), Description, Actions (Play/Stop/Restart buttons). Filter input. Status badge color: active=default, failed=destructive, other=secondary.

- [ ] **Step 3: Write system-ports.tsx**

Table with columns: Port, Protocol (Badge), Address, Process. Simple display, refresh button.

- [ ] **Step 4: Write system-disk.tsx**

Card per mountpoint with Progress bar. Color coding: >90% red, >70% orange. Shows target, used/size, percent.

- [ ] **Step 5: Write system-tab.tsx**

Container with Tabs: Prozesse, Services, Ports, Speicher. Each tab renders its sub-component.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/vm-cockpit/
git commit -m "feat(frontend): add System tab with processes, services, ports, and disk"
```

---

## Chunk 6: Frontend — VM Cockpit + Page Route

### Task 13: VM Cockpit Container + Detail Page

**Files:**
- Create: `frontend/src/components/vm-cockpit/vm-cockpit.tsx`
- Create: `frontend/src/app/(dashboard)/nodes/[id]/vms/[vmid]/page.tsx`
- Modify: `frontend/src/components/nodes/vm-list.tsx`

- [ ] **Step 1: Write vm-cockpit.tsx**

Main container: Header (VM name, status badge, type badge), Tabs (Shell, System, Monitoring). Sets VM context in store on mount.

- [ ] **Step 2: Write page route**

`/nodes/[id]/vms/[vmid]/page.tsx`: Uses useParams, fetches VM list from nodeStore, finds matching VM by vmid, renders VMCockpit. Loading skeleton fallback.

- [ ] **Step 3: Add click-through in vm-list.tsx**

Wrap VM name in Link to `/nodes/${nodeId}/vms/${vm.vmid}`.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/vm-cockpit/vm-cockpit.tsx
git add "frontend/src/app/(dashboard)/nodes/[id]/vms/[vmid]/page.tsx"
git add frontend/src/components/nodes/vm-list.tsx
git commit -m "feat(frontend): add VM detail page with cockpit and VM list click-through"
```

---

## Implementation Summary

**Phase 1 delivers:**
- VM detail page at `/nodes/{nodeId}/vms/{vmid}`
- Integrated browser terminal (xterm.js) with multi-session support
- System introspection: processes, services, ports, disk usage
- Service control (start/stop/restart) and process kill
- Fine-grained VM permission model (database + service + REST API)
- Click-through from VM list to VM cockpit

**Next plans:**
- Phase 2: Dateien-Tab + KI-Assistent
- Phase 3: Permission Admin UI + VM Groups
- Phase 4: Intelligenz, Automation, Dependency-Map
