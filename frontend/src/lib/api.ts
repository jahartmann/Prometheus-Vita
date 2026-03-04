import axios from "axios";
import { useAuthStore } from "@/stores/auth-store";

const api = axios.create({
  baseURL: "/api/v1",
  headers: {
    "Content-Type": "application/json",
  },
});

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Shared refresh promise to prevent concurrent refresh requests
let refreshPromise: Promise<string> | null = null;

api.interceptors.response.use(
  (response) => {
    // Unwrap { success, data } envelope from backend
    if (response.data && typeof response.data === "object" && "success" in response.data && "data" in response.data) {
      response.data = response.data.data;
    }
    return response;
  },
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        const refreshToken = useAuthStore.getState().refreshToken;
        if (!refreshToken) {
          useAuthStore.getState().logout();
          return Promise.reject(error);
        }

        // Use shared promise to deduplicate concurrent refresh calls
        if (!refreshPromise) {
          refreshPromise = axios
            .post("/api/v1/auth/refresh", { refresh_token: refreshToken })
            .then((response) => {
              const responseData = response.data?.data ?? response.data;
              const { access_token, refresh_token: new_refresh } = responseData;
              useAuthStore.getState().setAccessToken(access_token);
              if (new_refresh) {
                useAuthStore.getState().setRefreshToken(new_refresh);
              }
              return access_token;
            })
            .finally(() => {
              refreshPromise = null;
            });
        }

        const newToken = await refreshPromise;
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      } catch {
        useAuthStore.getState().logout();
        return Promise.reject(error);
      }
    }

    return Promise.reject(error);
  }
);

export default api;

/**
 * Safely extract an array from an API response, handling both
 * interceptor-unwrapped and raw envelope responses.
 */
export function toArray<T>(data: unknown): T[] {
  if (Array.isArray(data)) return data;
  if (data && typeof data === "object" && "data" in (data as Record<string, unknown>)) {
    const inner = (data as Record<string, unknown>).data;
    if (Array.isArray(inner)) return inner;
  }
  return [];
}

// Bulk VM API
export const bulkVmApi = {
  execute: (nodeId: string, data: { vmids: number[]; action: string }) =>
    api.post(`/nodes/${nodeId}/vms/bulk`, data),
};

// VM API
export const vmApi = {
  start: (nodeId: string, vmid: number, type: string) =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/start`, null, { params: { type } }),
  stop: (nodeId: string, vmid: number, type: string) =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/stop`, null, { params: { type } }),
  shutdown: (nodeId: string, vmid: number, type: string) =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/shutdown`, null, { params: { type } }),
  suspend: (nodeId: string, vmid: number, type: string) =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/suspend`, null, { params: { type } }),
  resume: (nodeId: string, vmid: number, type: string) =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/resume`, null, { params: { type } }),
  listSnapshots: (nodeId: string, vmid: number, type: string) =>
    api.get(`/nodes/${nodeId}/vms/${vmid}/snapshots`, { params: { type } }),
  createSnapshot: (nodeId: string, vmid: number, type: string, data: { name: string; description?: string; vmstate?: boolean }) =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/snapshots`, data, { params: { type } }),
  deleteSnapshot: (nodeId: string, vmid: number, type: string, snapname: string) =>
    api.delete(`/nodes/${nodeId}/vms/${vmid}/snapshots/${snapname}`, { params: { type } }),
  rollbackSnapshot: (nodeId: string, vmid: number, type: string, snapname: string) =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/snapshots/${snapname}/rollback`, null, { params: { type } }),
  getVNCProxy: (nodeId: string, vmid: number, type: string) =>
    api.post(`/nodes/${nodeId}/vms/${vmid}/vncproxy`, null, { params: { type } }),
};

// Backup API
export const backupApi = {
  listAll: () => api.get("/backups"),
  createBackup: (nodeId: string, data?: { backup_type?: string; notes?: string }) =>
    api.post(`/nodes/${nodeId}/backup`, data || {}),
  listBackups: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/backups`),
  getBackup: (backupId: string) =>
    api.get(`/backups/${backupId}`),
  getBackupFiles: (backupId: string) =>
    api.get(`/backups/${backupId}/files`),
  getBackupFile: (backupId: string, filePath: string) =>
    api.get(`/backups/${backupId}/files/${filePath}`),
  diffBackup: (backupId: string) =>
    api.get(`/backups/${backupId}/diff`),
  deleteBackup: (backupId: string) =>
    api.delete(`/backups/${backupId}`),
  restoreBackup: (backupId: string, data: { file_paths: string[]; dry_run: boolean }) =>
    api.post(`/backups/${backupId}/restore`, data),
  getRecoveryGuide: (backupId: string) =>
    api.get(`/backups/${backupId}/recovery-guide`),
  downloadBackup: (backupId: string) =>
    api.get(`/backups/${backupId}/download`, { responseType: "blob" }),
};

// Vzdump API
export const vzdumpApi = {
  create: (nodeId: string, data: { vmid: number; storage?: string; mode?: string; compress?: string }) =>
    api.post(`/nodes/${nodeId}/vzdump`, data),
};

// Node API (for vzdump dialog)
export const nodeApi = {
  list: () => api.get("/nodes"),
  getVMs: (nodeId: string) => api.get(`/nodes/${nodeId}/vms`),
  getStorage: (nodeId: string) => api.get(`/nodes/${nodeId}/storage`),
};

// Schedule API
export const scheduleApi = {
  listSchedules: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/backup-schedules`),
  createSchedule: (nodeId: string, data: { cron_expression: string; is_active?: boolean; retention_count?: number }) =>
    api.post(`/nodes/${nodeId}/backup-schedules`, data),
  updateSchedule: (scheduleId: string, data: { cron_expression?: string; is_active?: boolean; retention_count?: number }) =>
    api.put(`/backup-schedules/${scheduleId}`, data),
  deleteSchedule: (scheduleId: string) =>
    api.delete(`/backup-schedules/${scheduleId}`),
};

// Metrics API
export const metricsApi = {
  getHistory: (nodeId: string, since?: string, until?: string) =>
    api.get(`/nodes/${nodeId}/metrics`, { params: { since, until } }),
  getSummary: (nodeId: string, period?: string) =>
    api.get(`/nodes/${nodeId}/metrics/summary`, { params: { period } }),
};

// Network API
export const networkApi = {
  getInterfaces: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/network`),
  setAlias: (nodeId: string, iface: string, data: { display_name: string; description?: string; color?: string }) =>
    api.put(`/nodes/${nodeId}/network/${iface}/alias`, data),
};

// Disk API
export const diskApi = {
  getDisks: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/disks`),
};

// Tag API
export const tagApi = {
  list: () => api.get("/tags"),
  create: (data: { name: string; color?: string; category?: string }) =>
    api.post("/tags", data),
  delete: (tagId: string) => api.delete(`/tags/${tagId}`),
  getNodeTags: (nodeId: string) => api.get(`/nodes/${nodeId}/tags`),
  addToNode: (nodeId: string, tagId: string) =>
    api.post(`/nodes/${nodeId}/tags`, { tag_id: tagId }),
  removeFromNode: (nodeId: string, tagId: string) =>
    api.delete(`/nodes/${nodeId}/tags/${tagId}`),
  syncFromProxmox: (nodeId: string) =>
    api.post(`/nodes/${nodeId}/tags/sync`),
};

// PBS API
export const pbsApi = {
  getDatastores: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/pbs/datastores`),
  getBackupJobs: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/pbs/backup-jobs`),
};

// DR API
export const drApi = {
  getProfile: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/dr/profile`),
  collectProfile: (nodeId: string) =>
    api.post(`/nodes/${nodeId}/dr/profile`),
  getReadiness: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/dr/readiness`),
  calculateReadiness: (nodeId: string) =>
    api.post(`/nodes/${nodeId}/dr/readiness`),
  listRunbooks: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/dr/runbooks`),
  generateRunbook: (nodeId: string, scenario: string) =>
    api.post(`/nodes/${nodeId}/dr/runbooks`, { scenario }),
  getRunbook: (runbookId: string) =>
    api.get(`/dr/runbooks/${runbookId}`),
  updateRunbook: (runbookId: string, data: { title?: string; steps?: unknown }) =>
    api.put(`/dr/runbooks/${runbookId}`, data),
  deleteRunbook: (runbookId: string) =>
    api.delete(`/dr/runbooks/${runbookId}`),
  simulate: (nodeId: string, scenario: string) =>
    api.post("/dr/simulate", { node_id: nodeId, scenario }),
  listAllScores: () =>
    api.get("/dr/scores"),
};

// Notification API
export const notificationApi = {
  listChannels: () => api.get("/notifications/channels"),
  createChannel: (data: { name: string; type: string; config: Record<string, unknown> }) =>
    api.post("/notifications/channels", data),
  getChannel: (id: string) => api.get(`/notifications/channels/${id}`),
  updateChannel: (id: string, data: { name?: string; config?: Record<string, unknown>; is_active?: boolean }) =>
    api.put(`/notifications/channels/${id}`, data),
  deleteChannel: (id: string) => api.delete(`/notifications/channels/${id}`),
  testChannel: (id: string) => api.post(`/notifications/channels/${id}/test`),
  listHistory: (limit?: number, offset?: number) => {
    const params = new URLSearchParams();
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    return api.get("/notifications/history", { params });
  },
};

// Alert API
export const alertApi = {
  listRules: () => api.get("/alerts/rules"),
  createRule: (data: {
    name: string;
    node_id: string;
    metric: string;
    operator: string;
    threshold: number;
    duration_seconds?: number;
    severity: string;
    channel_ids?: string[];
    escalation_policy_id?: string;
    is_active?: boolean;
  }) => api.post("/alerts/rules", data),
  getRule: (id: string) => api.get(`/alerts/rules/${id}`),
  updateRule: (id: string, data: {
    name?: string;
    metric?: string;
    operator?: string;
    threshold?: number;
    duration_seconds?: number;
    severity?: string;
    channel_ids?: string[];
    escalation_policy_id?: string;
    is_active?: boolean;
  }) => api.put(`/alerts/rules/${id}`, data),
  deleteRule: (id: string) => api.delete(`/alerts/rules/${id}`),
};

// Escalation API
export const escalationApi = {
  listPolicies: () => api.get("/escalation/policies"),
  createPolicy: (data: {
    name: string;
    description?: string;
    steps?: { step_order: number; delay_seconds: number; channel_ids: string[] }[];
  }) => api.post("/escalation/policies", data),
  getPolicy: (id: string) => api.get(`/escalation/policies/${id}`),
  updatePolicy: (id: string, data: {
    name?: string;
    description?: string;
    is_active?: boolean;
    steps?: { step_order: number; delay_seconds: number; channel_ids: string[] }[];
  }) => api.put(`/escalation/policies/${id}`, data),
  deletePolicy: (id: string) => api.delete(`/escalation/policies/${id}`),
  listIncidents: (limit?: number, offset?: number) => {
    const params = new URLSearchParams();
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    return api.get("/escalation/incidents", { params });
  },
  getIncident: (id: string) => api.get(`/escalation/incidents/${id}`),
  acknowledgeIncident: (id: string) => api.post(`/escalation/incidents/${id}/acknowledge`),
  resolveIncident: (id: string) => api.post(`/escalation/incidents/${id}/resolve`),
};

// Telegram API
export const telegramApi = {
  link: () => api.post("/telegram/link"),
  status: () => api.get("/telegram/status"),
  unlink: () => api.delete("/telegram/unlink"),
};

// Chat API
export const chatApi = {
  chat: (data: { conversation_id?: string; message: string; model?: string }) =>
    api.post("/chat", data).then((r) => r.data?.data ?? r.data),
  listConversations: () =>
    api.get("/chat/conversations").then((r) => toArray(r.data)),
  getConversation: (id: string) =>
    api.get(`/chat/conversations/${id}`).then((r) => r.data?.data ?? r.data),
  getMessages: (id: string) =>
    api.get(`/chat/conversations/${id}/messages`).then((r) => toArray(r.data)),
  deleteConversation: (id: string) =>
    api.delete(`/chat/conversations/${id}`),
};

// Migration API
export const migrationApi = {
  start: (data: {
    source_node_id: string;
    target_node_id: string;
    vmid: number;
    target_storage: string;
    mode?: string;
    new_vmid?: number;
    cleanup_source?: boolean;
    cleanup_target?: boolean;
  }) => api.post("/migrations", data),
  list: () => api.get("/migrations"),
  get: (id: string) => api.get(`/migrations/${id}`),
  cancel: (id: string) => api.post(`/migrations/${id}/cancel`),
  delete: (id: string) => api.delete(`/migrations/${id}`),
  listByNode: (nodeId: string) => api.get(`/nodes/${nodeId}/migrations`),
};

// User API
export const userApi = {
  list: () => api.get("/users"),
  getById: (id: string) => api.get(`/users/${id}`),
  create: (data: { username: string; email?: string; password: string; role: string }) =>
    api.post("/users", data),
  update: (id: string, data: { username?: string; email?: string; role?: string; is_active?: boolean; autonomy_level?: number }) =>
    api.put(`/users/${id}`, data),
  delete: (id: string) => api.delete(`/users/${id}`),
  changePassword: (id: string, data: { current_password?: string; new_password: string }) =>
    api.post(`/users/${id}/password`, data),
};

// Anomaly API
export const anomalyApi = {
  listUnresolved: () => api.get("/anomalies").then((r) => toArray(r.data)),
  listByNode: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/anomalies`).then((r) => toArray(r.data)),
  resolve: (id: string) => api.post(`/anomalies/${id}/resolve`),
};

// Prediction API
export const predictionApi = {
  listCritical: () => api.get("/predictions").then((r) => toArray(r.data)),
  listByNode: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/predictions`).then((r) => toArray(r.data)),
};

// Briefing API
export const briefingApi = {
  getLatest: () => api.get("/briefings/latest").then((r) => r.data?.data ?? r.data),
  list: (limit?: number) => {
    return api.get("/briefings", { params: limit !== undefined ? { limit } : undefined }).then((r) => toArray(r.data));
  },
};

// Approval API
export const approvalApi = {
  listPending: () => api.get("/approvals").then((r) => toArray(r.data)),
  approve: (id: string) => api.post(`/approvals/${id}/approve`),
  reject: (id: string) => api.post(`/approvals/${id}/reject`),
};

// Drift Detection API
export const driftApi = {
  listAll: () => api.get("/drift"),
  listByNode: (nodeId: string) => api.get(`/nodes/${nodeId}/drift`),
  triggerCheck: (nodeId: string) => api.post(`/nodes/${nodeId}/drift/check`),
};

// Environment API
export const environmentApi = {
  list: () => api.get("/environments"),
  get: (id: string) => api.get(`/environments/${id}`),
  create: (data: { name: string; description?: string; color?: string }) =>
    api.post("/environments", data),
  update: (id: string, data: { name?: string; description?: string; color?: string }) =>
    api.put(`/environments/${id}`, data),
  delete: (id: string) => api.delete(`/environments/${id}`),
  assignNode: (nodeId: string, environmentId: string) =>
    api.put(`/nodes/${nodeId}/environment`, { environment_id: environmentId }),
};


// Right-Sizing API
export const rightsizingApi = {
  listAll: () => api.get("/rightsizing"),
  listByNode: (nodeId: string) => api.get(`/nodes/${nodeId}/rightsizing`),
  triggerAnalysis: (nodeId: string) => api.post(`/nodes/${nodeId}/rightsizing/analyze`),
};

// SSH Key API
export const sshKeyApi = {
  listByNode: (nodeId: string) => api.get(`/nodes/${nodeId}/ssh-keys`),
  generate: (nodeId: string, data: { name: string; key_type?: string; expires_at?: string; deploy?: boolean }) =>
    api.post(`/nodes/${nodeId}/ssh-keys`, data),
  deploy: (nodeId: string, keyId: string) =>
    api.post(`/nodes/${nodeId}/ssh-keys/${keyId}/deploy`),
  rotate: (nodeId: string) =>
    api.post(`/nodes/${nodeId}/ssh-keys/rotate`),
  delete: (nodeId: string, keyId: string) =>
    api.delete(`/nodes/${nodeId}/ssh-keys/${keyId}`),
  getRotationSchedule: (nodeId: string) =>
    api.get(`/nodes/${nodeId}/ssh-keys/rotation`),
  createRotationSchedule: (nodeId: string, data: { interval_days: number; is_active: boolean }) =>
    api.post(`/nodes/${nodeId}/ssh-keys/rotation`, data),
};

// API Gateway API
export const gatewayApi = {
  listTokens: () => api.get("/gateway/tokens"),
  createToken: (data: { name: string; permissions?: string[]; expires_at?: string }) =>
    api.post("/gateway/tokens", data),
  revokeToken: (id: string) => api.post(`/gateway/tokens/${id}/revoke`),
  deleteToken: (id: string) => api.delete(`/gateway/tokens/${id}`),
  listAuditLog: (limit?: number, offset?: number) => {
    const params = new URLSearchParams();
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    return api.get("/gateway/audit", { params });
  },
};

// Agent Config API
export const agentConfigApi = {
  get: () => api.get("/agent/config"),
  update: (data: Record<string, string>) => api.put("/agent/config", data),
  getModels: () => api.get("/agent/models"),
};

// ISO/Template API
export const isoApi = {
  listISOs: (nodeId: string) => api.get(`/nodes/${nodeId}/isos`),
  listTemplates: (nodeId: string) => api.get(`/nodes/${nodeId}/templates`),
  syncContent: (nodeId: string, data: { source_node_id: string; volid: string; target_storage: string }) =>
    api.post(`/nodes/${nodeId}/sync-content`, data),
};

// Reflex API
export const reflexApi = {
  list: () => api.get("/reflexes"),
  create: (data: {
    name: string;
    description?: string;
    trigger_metric: string;
    operator: string;
    threshold: number;
    action_type: string;
    action_config?: Record<string, unknown>;
    cooldown_seconds?: number;
    is_active?: boolean;
    node_id?: string;
  }) => api.post("/reflexes", data),
  update: (id: string, data: {
    name?: string;
    description?: string;
    trigger_metric?: string;
    operator?: string;
    threshold?: number;
    action_type?: string;
    action_config?: Record<string, unknown>;
    cooldown_seconds?: number;
    is_active?: boolean;
    node_id?: string;
  }) => api.put(`/reflexes/${id}`, data),
  delete: (id: string) => api.delete(`/reflexes/${id}`),
};


// Brain API
export const brainApi = {
  list: () => api.get("/brain"),
  create: (data: { category: string; subject: string; content: string }) => api.post("/brain", data),
  delete: (id: string) => api.delete(`/brain/${id}`),
  search: (query: string) => api.get(`/brain/search?q=${encodeURIComponent(query)}`),
};
