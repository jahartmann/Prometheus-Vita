export interface User {
  id: string;
  username: string;
  email: string;
  role: string;
  is_active: boolean;
  autonomy_level: number;
  must_change_password: boolean;
  last_login?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface PasswordPolicy {
  id: string;
  min_length: number;
  require_uppercase: boolean;
  require_lowercase: boolean;
  require_digit: boolean;
  require_special: boolean;
  max_length: number;
  disallow_username: boolean;
  updated_at: string;
  updated_by?: string;
}

export interface UpdatePasswordPolicyRequest {
  min_length?: number;
  require_uppercase?: boolean;
  require_lowercase?: boolean;
  require_digit?: boolean;
  require_special?: boolean;
  max_length?: number;
  disallow_username?: boolean;
}

export interface UserResponse {
  id: string;
  username: string;
  email: string;
  role: string;
  is_active: boolean;
  autonomy_level: number;
  created_at: string;
  updated_at: string;
  last_login?: string | null;
}

export interface CreateUserRequest {
  username: string;
  email?: string;
  password: string;
  role: string;
}

export interface UpdateUserRequest {
  username?: string;
  email?: string;
  role?: string;
  is_active?: boolean;
  autonomy_level?: number;
}

export interface ChangePasswordRequest {
  current_password?: string;
  new_password: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
}

export type NodeType = "pve" | "pbs";
export interface Node {
  id: string;
  name: string;
  type: NodeType;
  hostname: string;
  port: number;
  is_online: boolean;
  last_seen?: string | null;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface NodeStatus {
  node: string;
  node_id: string;
  cpu_usage: number;
  cpu_cores: number;
  cpu_model: string;
  memory_total: number;
  memory_used: number;
  memory_free: number;
  swap_total: number;
  swap_used: number;
  disk_total: number;
  disk_used: number;
  net_in: number;
  net_out: number;
  uptime: number;
  load_average: number[];
  kernel_version: string;
  pve_version: string;
  vm_count: number;
  vm_running: number;
  ct_count: number;
  ct_running: number;
}

export interface NodeMetrics {
  timestamp: string;
  cpu_usage: number;
  memory_usage: number;
  disk_io_read: number;
  disk_io_write: number;
  network_in: number;
  network_out: number;
}

export type VMType = "qemu" | "lxc";
export type VMStatus = "running" | "stopped" | "paused" | "suspended";

export interface VM {
  vmid: number;
  name: string;
  type: VMType;
  status: VMStatus;
  cpu_usage: number;
  cpu_cores: number;
  memory_total: number;
  memory_used: number;
  disk_total: number;
  disk_used: number;
  uptime: number;
  net_in: number;
  net_out: number;
  disk_read: number;
  disk_write: number;
  node_id: string;
  tags: string[];
}

export interface BulkVMResult {
  vmid: number;
  success: boolean;
  error?: string;
  upid?: string;
}

export interface CreateNodeRequest {
  name: string;
  type: NodeType;
  hostname: string;
  port: number;
  api_token_id: string;
  api_token_secret: string;
  metadata?: Record<string, unknown>;
}

export interface OnboardNodeRequest {
  name: string;
  type: 'pve' | 'pbs';
  hostname: string;
  password: string;
  port?: number;
  ssh_port?: number;
  username?: string;
}

export interface TestConnectionRequest {
  hostname: string;
  port: number;
  type: NodeType;
  api_token_id: string;
  api_token_secret: string;
}

export interface TestConnectionResponse {
  success: boolean;
  version?: string;
  node?: string;
  error?: string;
}

export interface Alert {
  id: string;
  node_id: string;
  severity: "info" | "warning" | "critical";
  category: string;
  message: string;
  acknowledged: boolean;
  created_at: string;
  resolved_at: string | null;
}

export interface ApiError {
  error: string;
  message: string;
  status_code: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
}

// Backup types
export type BackupType = "manual" | "scheduled" | "pre_update";
export type BackupStatus = "pending" | "running" | "completed" | "failed";

export interface ConfigBackup {
  id: string;
  node_id: string;
  version: number;
  backup_type: BackupType;
  file_count: number;
  total_size: number;
  status: BackupStatus;
  error_message?: string;
  notes?: string;
  recovery_guide?: string;
  created_at: string;
  completed_at?: string;
}

export interface BackupFile {
  id: string;
  backup_id: string;
  file_path: string;
  file_hash: string;
  file_size: number;
  file_permissions?: string;
  file_owner?: string;
  diff_from_previous?: string;
  created_at: string;
}

export interface BackupSchedule {
  id: string;
  node_id: string;
  cron_expression: string;
  is_active: boolean;
  retention_count: number;
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

// Metrics types
export interface MetricsRecord {
  id: number;
  node_id: string;
  recorded_at: string;
  cpu_usage: number;
  memory_used: number;
  memory_total: number;
  disk_used: number;
  disk_total: number;
  net_in: number;
  net_out: number;
  load_avg: number[];
}

export interface MetricsSummary {
  node_id: string;
  period: string;
  cpu_avg: number;
  cpu_max: number;
  cpu_min: number;
  cpu_current: number;
  memory_avg_percent: number;
  memory_max_percent: number;
  memory_min_percent?: number;
  memory_current_percent?: number;
  disk_avg_percent: number;
  disk_max_percent: number;
  disk_min_percent?: number;
  disk_current_percent?: number;
}

// Network types
export interface NetworkInterface {
  iface: string;
  type: string;
  cidr?: string;
  address?: string;
  gateway?: string;
  active: number;
  method?: string;
  comments?: string;
  autostart: number;
  display_name?: string;
  description?: string;
  color?: string;
}

// Node port types
export interface NodePort {
  protocol: string;
  state: string;
  local_address: string;
  local_port: number;
  peer_address?: string;
  peer_port?: number;
  process?: string;
}

export interface NodePortsData {
  listening: NodePort[];
  established: NodePort[];
  other: NodePort[];
}

// Disk types
export interface DiskInfo {
  devpath: string;
  size: number;
  model?: string;
  serial?: string;
  type: string;
  health?: string;
  wearout?: string;
  gpt: number;
  vendor?: string;
}

// Tag types
export interface Tag {
  id: string;
  name: string;
  color: string;
  category?: string;
  created_at: string;
}

export interface VMTag {
  node_id: string;
  vmid: number;
  vm_type: string;
  tag_id: string;
  created_at: string;
}

// PBS types
export interface PBSDatastore {
  name: string;
  path?: string;
  comment?: string;
  total?: number;
  used?: number;
  available?: number;
  usage_percent?: number;
  gc_status?: string;
}

export interface PBSBackupJob {
  id: string;
  store: string;
  schedule?: string;
  comment?: string;
  remote?: string;
  remote_store?: string;
}

// Diff types
export interface FileDiff {
  file_path: string;
  status: "added" | "removed" | "modified" | "unchanged";
  diff?: string;
}

// Restore types
export interface RestorePreview {
  files: RestoreFilePreview[];
}

export interface RestoreFilePreview {
  file_path: string;
  action: string;
  diff?: string;
  current_hash?: string;
  backup_hash: string;
}

// Disaster Recovery types
export interface NodeProfile {
  id: string;
  node_id: string;
  collected_at: string;
  cpu_model?: string;
  cpu_cores?: number;
  cpu_threads?: number;
  memory_total_bytes?: number;
  memory_modules?: unknown;
  disks?: unknown;
  network_interfaces?: unknown;
  pve_version?: string;
  kernel_version?: string;
  installed_packages?: unknown;
  storage_layout?: unknown;
  custom_data?: unknown;
}

export interface DRReadinessScore {
  id: string;
  node_id: string;
  overall_score: number;
  backup_score: number;
  profile_score: number;
  config_score: number;
  details?: Record<string, unknown>;
  calculated_at: string;
}

export interface RunbookStep {
  title: string;
  description: string;
  command?: string;
  expected_output?: string;
  is_manual: boolean;
}

export interface RecoveryRunbook {
  id: string;
  node_id?: string;
  title: string;
  scenario: string;
  steps: RunbookStep[];
  is_template: boolean;
  generated_at: string;
  updated_at: string;
}

export interface DRSimulationCheck {
  name: string;
  passed: boolean;
  message: string;
}

export interface DRSimulationResult {
  node_id: string;
  scenario: string;
  ready: boolean;
  checks: DRSimulationCheck[];
  summary: string;
}

// Notification types
export type NotificationChannelType = 'email' | 'telegram' | 'webhook';
export type NotificationStatus = 'pending' | 'sent' | 'failed';
export type AlertSeverity = 'info' | 'warning' | 'critical';

export interface NotificationChannel {
  id: string;
  name: string;
  type: NotificationChannelType;
  config: Record<string, unknown>;
  is_active: boolean;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface NotificationHistoryEntry {
  id: string;
  channel_id?: string;
  event_type: string;
  subject: string;
  body: string;
  status: NotificationStatus;
  error_message?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  sent_at?: string;
}

export interface AlertRule {
  id: string;
  name: string;
  node_id: string;
  metric: string;
  operator: string;
  threshold: number;
  duration_seconds: number;
  severity: AlertSeverity;
  channel_ids: string[];
  escalation_policy_id?: string;
  is_active: boolean;
  last_triggered_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateChannelRequest {
  name: string;
  type: NotificationChannelType;
  config: Record<string, unknown>;
}

export interface UpdateChannelRequest {
  name?: string;
  config?: Record<string, unknown>;
  is_active?: boolean;
}

export interface CreateAlertRuleRequest {
  name: string;
  node_id: string;
  metric: string;
  operator: string;
  threshold: number;
  duration_seconds?: number;
  severity: AlertSeverity;
  channel_ids?: string[];
  escalation_policy_id?: string;
  is_active?: boolean;
}

export interface UpdateAlertRuleRequest {
  name?: string;
  metric?: string;
  operator?: string;
  threshold?: number;
  duration_seconds?: number;
  severity?: AlertSeverity;
  channel_ids?: string[];
  escalation_policy_id?: string;
  is_active?: boolean;
}

// Escalation types
export interface EscalationStep {
  id: string;
  policy_id: string;
  step_order: number;
  delay_seconds: number;
  channel_ids: string[];
  created_at: string;
}

export interface EscalationPolicy {
  id: string;
  name: string;
  description?: string;
  is_active: boolean;
  steps?: EscalationStep[];
  created_at: string;
  updated_at: string;
}

export type IncidentStatus = 'triggered' | 'acknowledged' | 'resolved';

export interface AlertIncident {
  id: string;
  alert_rule_id: string;
  status: IncidentStatus;
  current_step: number;
  triggered_at: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  resolved_at?: string;
  resolved_by?: string;
  last_escalated_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateEscalationStepInput {
  step_order: number;
  delay_seconds: number;
  channel_ids: string[];
}

export interface CreateEscalationPolicyRequest {
  name: string;
  description?: string;
  steps?: CreateEscalationStepInput[];
}

export interface UpdateEscalationPolicyRequest {
  name?: string;
  description?: string;
  is_active?: boolean;
  steps?: CreateEscalationStepInput[];
}

// Telegram types
export interface TelegramLinkResponse {
  verification_code: string;
  bot_username: string;
  is_verified: boolean;
}

export interface TelegramStatus {
  linked: boolean;
  is_verified: boolean;
  telegram_username?: string;
  verification_code?: string;
  bot_enabled: boolean;
  bot_username?: string;
}

// Chat / AI Agent types
export type ChatMessageRole = 'user' | 'assistant' | 'system' | 'tool';

export interface ChatConversation {
  id: string;
  user_id: string;
  title: string;
  model: string;
  created_at: string;
  updated_at: string;
}

export interface ChatMessage {
  id: string;
  conversation_id: string;
  role: ChatMessageRole;
  content: string;
  tool_calls?: unknown;
  tool_call_id?: string;
  created_at: string;
}

export interface AgentToolCall {
  id: string;
  message_id: string;
  tool_name: string;
  arguments: unknown;
  result: unknown;
  status: string;
  duration_ms: number;
  created_at: string;
}

export interface ChatRequest {
  conversation_id?: string;
  message: string;
  model?: string;
}

export interface ChatResponse {
  conversation_id: string;
  message: ChatMessage;
  tool_calls?: AgentToolCall[];
}

// VM Migration types
export type MigrationStatus =
  | "pending"
  | "preparing"
  | "backing_up"
  | "transferring"
  | "restoring"
  | "cleaning_up"
  | "completed"
  | "failed"
  | "cancelled";

export type MigrationMode = "stop" | "snapshot" | "suspend";

export interface VMMigration {
  id: string;
  source_node_id: string;
  target_node_id: string;
  vmid: number;
  vm_name: string;
  vm_type: string;
  status: MigrationStatus;
  mode: MigrationMode;
  target_storage: string;
  progress: number;
  current_step: string;
  transfer_bytes_sent: number;
  transfer_speed_bps: number;
  error_message?: string;
  log_entries?: string[];
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface StartMigrationRequest {
  source_node_id: string;
  target_node_id: string;
  vmid: number;
  target_storage: string;
  mode?: MigrationMode;
  new_vmid?: number;
  cleanup_source?: boolean;
  cleanup_target?: boolean;
}

// Phase 4: Autonomy, Anomaly, Prediction, Briefing types

export type ApprovalStatus = 'pending' | 'approved' | 'rejected';

export interface AgentPendingApproval {
  id: string;
  user_id: string;
  conversation_id: string;
  message_id: string;
  tool_name: string;
  arguments: unknown;
  status: ApprovalStatus;
  resolved_by?: string;
  resolved_at?: string;
  created_at: string;
}

export interface AnomalyRecord {
  id: string;
  node_id: string;
  metric: string;
  value: number;
  z_score: number;
  mean: number;
  stddev: number;
  severity: string;
  is_resolved: boolean;
  detected_at: string;
  resolved_at?: string;
  // Enriched context fields
  node_name?: string;
  description?: string;
  impact?: string;
  recommendation?: string;
  affected_vms?: string[];
}

export interface MaintenancePrediction {
  id: string;
  node_id: string;
  metric: string;
  current_value: number;
  predicted_value: number;
  threshold: number;
  days_until_threshold?: number;
  slope: number;
  intercept: number;
  r_squared: number;
  severity: string;
  predicted_at: string;
  // Enriched context fields
  node_name?: string;
  description?: string;
  recommendation?: string;
  trend_direction?: string;
  affected_vms?: string[];
  vm_count?: number;
}

export interface MorningBriefing {
  id: string;
  summary: string;
  data: BriefingData;
  generated_at: string;
}

export interface BriefingData {
  total_nodes: number;
  online_nodes: number;
  offline_nodes: number;
  active_alerts: number;
  unresolved_anomalies: number;
  critical_predictions: number;
  node_summaries: BriefingNodeSummary[];
}

export interface BriefingNodeSummary {
  node_id: string;
  node_name: string;
  is_online: boolean;
  cpu_avg: number;
  mem_pct: number;
  disk_pct: number;
}

// Live Briefing types
export interface LiveBriefingSummary {
  nodes_online: number;
  nodes_offline: number;
  nodes_total: number;
  vms_running: number;
  vms_stopped: number;
  vms_total: number;
  avg_cpu: number;
  avg_ram: number;
  avg_disk: number;
  top_nodes_by_cpu: NodeCPURank[];
  top_vms_by_ram: VMRAMRank[];
  unresolved_anomalies: number;
  critical_predictions: number;
  node_details: LiveNodeDetail[];
}

export interface NodeCPURank {
  node_id: string;
  node_name: string;
  cpu_usage: number;
}

export interface VMRAMRank {
  node_id: string;
  node_name: string;
  vmid: number;
  vm_name: string;
  mem_used_pct: number;
  mem_used: number;
  mem_total: number;
}

export interface LiveNodeDetail {
  node_id: string;
  node_name: string;
  is_online: boolean;
  cpu_usage: number;
  mem_pct: number;
  disk_pct: number;
  vm_count: number;
  vm_running: number;
  uptime: number;
}

// Reflex types
export type ReflexActionType = 'restart_service' | 'clear_cache' | 'notify' | 'run_command' | 'start_vm' | 'stop_vm' | 'scale_up' | 'scale_down' | 'snapshot' | 'ai_analyze';

export interface ReflexRule {
  id: string;
  name: string;
  description?: string;
  trigger_metric: string;
  operator: string;
  threshold: number;
  action_type: ReflexActionType;
  action_config: Record<string, unknown>;
  cooldown_seconds: number;
  is_active: boolean;
  node_id?: string;
  last_triggered_at?: string;
  trigger_count: number;
  // Time-based scheduling
  schedule_type?: string;
  schedule_cron?: string;
  time_window_start?: string;
  time_window_end?: string;
  time_window_days?: number[];
  // AI integration
  ai_enabled?: boolean;
  ai_severity?: string;
  ai_recommendation?: string;
  // Organization
  priority?: number;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

// Phase 6: Drift Detection types
export type DriftStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface DriftCheck {
  id: string;
  node_id: string;
  status: DriftStatus;
  total_files: number;
  changed_files: number;
  added_files: number;
  removed_files: number;
  details?: DriftFileDetail[];
  ai_analysis?: AIAnalysisResult;
  error_message?: string;
  baseline_updated_at?: string;
  checked_at: string;
  created_at: string;
}

export interface AIFileAnalysis {
  file_path: string;
  severity: number;
  severity_reason: string;
  category: 'Security' | 'Performance' | 'Network' | 'Configuration' | 'Cosmetic';
  risk_assessment: string;
  recommendation: 'fix' | 'accept' | 'monitor';
  summary: string;
}

export interface AIAnalysisResult {
  analyzed_at: string;
  model: string;
  file_analyses: AIFileAnalysis[];
  overall_severity: number;
  overall_summary: string;
}

export interface DriftFileDetail {
  file_path: string;
  status: 'added' | 'removed' | 'modified' | 'unchanged';
  diff?: string;
  acknowledged: boolean;
  ai_file_analysis?: AIFileAnalysis;
}

export interface CompareNodesRequest {
  file_paths: string[];
  node_ids: string[];
}

export interface NodeFileContent {
  node_id: string;
  node_name: string;
  content: string;
  error?: string;
}

export interface NodeDifference {
  node_a: string;
  node_a_name: string;
  node_b: string;
  node_b_name: string;
  diff: string;
  identical: boolean;
}

export interface NodeComparisonEntry {
  file_path: string;
  node_files: NodeFileContent[];
  differences: NodeDifference[];
}

export interface CompareNodesResponse {
  comparisons: NodeComparisonEntry[];
}

// Phase 6: Environment types
export interface Environment {
  id: string;
  name: string;
  description?: string;
  color: string;
  created_at: string;
  updated_at: string;
}

export interface CreateEnvironmentRequest {
  name: string;
  description?: string;
  color?: string;
}

export interface UpdateEnvironmentRequest {
  name?: string;
  description?: string;
  color?: string;
}

// Phase 6: Resource Right-Sizing types
export type RecommendationType = 'downsize' | 'upsize' | 'optimal';

export interface ResourceRecommendation {
  id: string;
  node_id: string;
  vmid: number;
  vm_name: string;
  vm_type: string;
  resource_type: string;
  current_value: number;
  recommended_value: number;
  avg_usage: number;
  max_usage: number;
  recommendation_type: RecommendationType;
  reason?: string;
  vm_context?: string;
  context_reason?: string;
  created_at: string;
}

// Phase 6: SSH Key types
export interface SSHKey {
  id: string;
  node_id: string;
  name: string;
  key_type: string;
  public_key: string;
  fingerprint: string;
  is_deployed: boolean;
  deployed_at?: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export interface SSHKeyRotationSchedule {
  id: string;
  node_id: string;
  interval_days: number;
  is_active: boolean;
  last_rotated_at?: string;
  next_rotation_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSSHKeyRequest {
  name: string;
  key_type?: string;
  expires_at?: string;
  deploy?: boolean;
}

// Phase 6: API Gateway types
export interface APIToken {
  id: string;
  user_id: string;
  name: string;
  token_prefix: string;
  permissions: string[];
  is_active: boolean;
  last_used_at?: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAPITokenRequest {
  name: string;
  permissions?: string[];
  expires_at?: string;
}

export interface CreateAPITokenResponse {
  token: string;
  token_id: string;
  name: string;
  prefix: string;
}

export interface AuditLogEntry {
  id: string;
  user_id?: string;
  username?: string;
  api_token_id?: string;
  method: string;
  path: string;
  status_code: number;
  ip_address?: string;
  user_agent?: string;
  duration_ms: number;
  created_at: string;
}

// VM Snapshot types
export interface VMSnapshot {
  name: string;
  description: string;
  parent: string;
  snaptime: number;
  vmstate: number;
}

// VNC Proxy types
export interface VNCProxyTicket {
  ticket: string;
  port: string;
  cert: string;
  user: string;
  upid: string;
}


// RRD data types
export interface RRDDataPoint {
  time: number;
  cpu: number;
  net_in: number;
  net_out: number;
  mem_used: number;
  mem_total: number;
  root_used: number;
  root_total: number;
  load_avg: number;
  io_wait: number;
}

// VM Metrics types
export interface VMMetricsRecord {
  id: string;
  node_id: string;
  vmid: number;
  vm_type: string;
  cpu_usage: number;
  memory_used: number;
  memory_total: number;
  net_in: number;
  net_out: number;
  disk_read: number;
  disk_write: number;
  recorded_at: string;
}

export interface NetworkSummary {
  total_in: number;
  total_out: number;
  avg_in_rate: number;
  avg_out_rate: number;
  peak_in_rate: number;
  peak_out_rate: number;
}

// Cluster Storage types
export interface ClusterStorageItem {
  node_id: string;
  node_name: string;
  storage: string;
  type: string;
  content: string;
  total: number;
  used: number;
  available: number;
  usage_percent: number;
  active: boolean;
  shared: boolean;
}

// ISO/Template types
export interface StorageContent {
  volid: string;
  format: string;
  size: number;
  ctime: number;
}

// Cluster-wide ISO type
export interface ClusterISO {
  name: string;
  volid: string;
  format: string;
  size: number;
  ctime: number;
  nodes: string[];
}

// Security Event types
export type SecurityCategory = "performance" | "security" | "capacity" | "availability" | "config";
export type SecuritySeverity = "info" | "warning" | "critical" | "emergency";

export interface SecurityEvent {
  id: string;
  node_id: string;
  category: SecurityCategory;
  severity: SecuritySeverity;
  title: string;
  description: string;
  impact: string;
  recommendation: string;
  metrics?: Record<string, unknown>;
  affected_vms?: string[];
  node_name?: string;
  is_acknowledged: boolean;
  detected_at: string;
  acknowledged_at?: string;
  analysis_model?: string;
}

export interface SecurityStats {
  total: number;
  unacknowledged: number;
  by_severity: Record<string, number>;
  by_category: Record<string, number>;
}

// VM Cockpit types
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

export interface VMFileEntry {
  name: string;
  type: "file" | "directory" | "symlink";
  permissions: string;
  owner: string;
  group: string;
  size: number;
  modified: string;
  link_target?: string;
}

// Tag sync-all response
export interface TagSyncAllResult {
  total_imported: number;
  results: TagSyncNodeResult[];
}

export interface TagSyncNodeResult {
  node_id: string;
  node_name: string;
  imported: number;
  error?: string;
}

// VM Group types
export interface VMGroup {
  id: string;
  name: string;
  description: string;
  tag_filter: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  member_count?: number;
}

export interface VMGroupMember {
  group_id: string;
  node_id: string;
  vmid: number;
}

// VM Health Score types (Phase 4)
export interface VMHealthScore {
  node_id: string;
  vmid: number;
  vm_name: string;
  vm_type: string;
  score: number;
  status: "healthy" | "warning" | "critical" | "stopped";
  breakdown: VMHealthBreakdown;
  updated_at: string;
}

export interface VMHealthBreakdown {
  cpu_score: number;
  cpu_avg: number;
  ram_score: number;
  ram_avg: number;
  disk_score: number;
  disk_usage: number;
  stability_score: number;
  uptime_days: number;
  crash_count: number;
}

export interface VMRightsizingResult {
  node_id: string;
  vmid: number;
  vm_name: string;
  vm_type: string;
  resources: VMResourceRec[];
  analyzed_at: string;
}

export interface VMResourceRec {
  resource: string;
  current_value: string;
  recommended_value: string;
  avg_usage: number;
  max_usage: number;
  status: "optimal" | "reduce" | "increase";
  reason: string;
}

export interface VMCockpitAnomaly {
  node_id: string;
  vmid: number;
  vm_name: string;
  metric: string;
  value: number;
  mean: number;
  stddev: number;
  z_score: number;
  severity: "warning" | "critical";
  message: string;
  detected_at: string;
}

export interface SnapshotPolicy {
  id: string;
  node_id: string;
  vmid: number;
  vm_type: string;
  name: string;
  keep_daily: number;
  keep_weekly: number;
  keep_monthly: number;
  schedule_cron: string;
  is_active: boolean;
  last_run?: string;
  created_at: string;
}

export interface ScheduledAction {
  id: string;
  node_id: string;
  vmid?: number;
  vm_type?: string;
  action: string;
  schedule_cron: string;
  is_active: boolean;
  description?: string;
  created_at: string;
}

export interface VMDependency {
  id: string;
  source_node_id: string;
  source_vmid: number;
  target_node_id: string;
  target_vmid: number;
  dependency_type: string;
  description?: string;
  created_at: string;
  source_vm_name?: string;
  target_vm_name?: string;
  source_vm_type?: string;
  target_vm_type?: string;
  source_status?: string;
  target_status?: string;
}
