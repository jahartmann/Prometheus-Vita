package api

import (
	"github.com/antigravity/prometheus/internal/api/handler"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/config"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/auth"
	"github.com/antigravity/prometheus/internal/service/gateway"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type Handlers struct {
	Health       *handler.HealthHandler
	Auth         *handler.AuthHandler
	Node         *handler.NodeHandler
	WS           *handler.WSHandler
	Backup       *handler.BackupHandler
	Schedule     *handler.ScheduleHandler
	Metrics      *handler.MetricsHandler
	Tag          *handler.TagHandler
	PBS          *handler.PBSHandler
	User         *handler.UserHandler
	DR           *handler.DRHandler
	Notification *handler.NotificationHandler
	Chat         *handler.ChatHandler
	Migration    *handler.MigrationHandler
	Escalation   *handler.EscalationHandler
	Telegram     *handler.TelegramHandler
	Cluster      *handler.ClusterHandler
	Anomaly      *handler.AnomalyHandler
	Prediction   *handler.PredictionHandler
	Briefing     *handler.BriefingHandler
	Approval     *handler.ApprovalHandler
	Drift        *handler.DriftHandler
	Environment  *handler.EnvironmentHandler
	Update       *handler.UpdateHandler
	Rightsizing  *handler.RightsizingHandler
	SSHKey       *handler.SSHKeyHandler
	Gateway        *handler.GatewayHandler
	Log            *handler.LogHandler
	Topology       *handler.TopologyHandler
	Brain          *handler.BrainHandler
	Reflex         *handler.ReflexHandler
	AgentConfig    *handler.AgentConfigHandler
	SyncCenter     *handler.SyncCenterHandler
	Security       *handler.SecurityHandler
	PasswordPolicy *handler.PasswordPolicyHandler
	VMCockpit      *handler.VMCockpitHandler
	VMPermission   *handler.VMPermissionHandler
	VMGroup        *handler.VMGroupHandler
	VMHealth       *handler.VMHealthHandler

	// Log & Network Analysis
	LogAnalysis       *handler.LogAnalysisHandler
	LogBookmark       *handler.LogBookmarkHandler
	LogSource         *handler.LogSourceHandler
	LogExport         *handler.LogExportHandler
	LogReportSchedule *handler.LogReportScheduleHandler
	LogStream         *handler.LogStreamHandler
	NetworkScan       *handler.NetworkScanHandler
	NetworkDevice     *handler.NetworkDeviceHandler
	NetworkAnomaly    *handler.NetworkAnomalyHandler
	ScanBaseline      *handler.ScanBaselineHandler
}

func SetupRouter(e *echo.Echo, cfg *config.Config, jwtSvc *auth.JWTService, h Handlers, gatewaySvc *gateway.Service, redisClient *redis.Client, auditRepo repository.AuditRepository, userRepo repository.UserRepository) {
	// Global middleware
	e.Use(middleware.Recovery())
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLogger())
	e.Use(middleware.SecurityHeaders())
	e.Use(middleware.CORS(cfg.CORS))

	// Rate limiting (applied globally)
	if redisClient != nil {
		e.Use(middleware.RateLimit(redisClient, middleware.RateLimitConfig{
			RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
			Enabled:           cfg.RateLimit.Enabled,
		}))
	}

	// Audit logging
	if auditRepo != nil {
		e.Use(middleware.AuditLog(auditRepo))
	}

	// Health check (no auth)
	e.GET("/health", h.Health.Check)

	// API v1
	v1 := e.Group("/api/v1")

	// API key auth (before JWT, so X-API-Key works as alternative)
	if gatewaySvc != nil {
		v1.Use(middleware.APIKeyAuth(gatewaySvc))
	}

	// Auth routes (no JWT)
	authGroup := v1.Group("/auth")
	authGroup.POST("/login", h.Auth.Login)
	authGroup.POST("/logout", h.Auth.Logout)
	authGroup.POST("/refresh", h.Auth.Refresh)

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(jwtSvc))
	if userRepo != nil {
		protected.Use(middleware.MustChangePassword(userRepo))
	}

	// Auth - protected
	protected.GET("/auth/me", h.Auth.Me)

	// Cluster-level storage (aggregates all nodes)
	protected.GET("/storage", h.Node.GetClusterStorage)

	// Cluster Dashboard
	if h.Cluster != nil {
		cluster := protected.Group("/cluster")
		cluster.GET("/summary", h.Cluster.GetSummary)
		cluster.GET("/history", h.Cluster.GetHistory)
	}

	// Nodes
	nodes := protected.Group("/nodes")
	nodes.GET("", h.Node.List)
	nodes.GET("/:id", h.Node.Get)
	nodes.GET("/:id/status", h.Node.GetStatus)
	nodes.GET("/:id/vms", h.Node.GetVMs)
	nodes.GET("/:id/storage", h.Node.GetStorage)
	nodes.GET("/:id/network", h.Node.GetNetworkInterfaces)
	nodes.GET("/:id/ports", h.Node.GetPorts)
	nodes.GET("/:id/disks", h.Node.GetDisks)
	nodes.GET("/:id/tags", h.Tag.GetNodeTags)
	nodes.GET("/:id/diagnose", h.Node.DiagnoseConnectivity)
	nodes.GET("/:id/isos", h.Node.ListISOs)
	nodes.GET("/:id/templates", h.Node.ListTemplates)

	// VM tags (read)
	nodes.GET("/:id/vms/:vmid/tags", h.Tag.GetVMTags)

	// Node backups (read)
	nodes.GET("/:id/backups", h.Backup.ListBackups)
	nodes.GET("/:id/backup-schedules", h.Schedule.ListSchedules)

	// Node metrics
	nodes.GET("/:id/metrics", h.Metrics.GetMetricsHistory)
	nodes.GET("/:id/metrics/summary", h.Metrics.GetMetricsSummary)
	nodes.GET("/:id/network-summary", h.Metrics.GetNodeNetworkSummary)
	nodes.GET("/:id/vms/:vmid/metrics", h.Metrics.GetVMMetricsHistory)
	nodes.GET("/:id/vms/:vmid/network-summary", h.Metrics.GetVMNetworkSummary)
	nodes.GET("/:id/rrd", h.Metrics.GetNodeRRDData)
	nodes.GET("/:id/vms/:vmid/rrd", h.Metrics.GetVMRRDData)

	// Cluster-wide network summary
	protected.GET("/network-summary", h.Metrics.GetClusterNetworkSummary)

	// Node logs
	if h.Log != nil {
		nodes.GET("/:id/logs", h.Log.GetLogs)
	}

	// PBS (read)
	nodes.GET("/:id/pbs/datastores", h.PBS.GetDatastores)
	nodes.GET("/:id/pbs/backup-jobs", h.PBS.GetBackupJobs)

	// Admin/Operator only
	adminOp := nodes.Group("")
	adminOp.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	adminOp.POST("", h.Node.Create)
	adminOp.POST("/onboard", h.Node.Onboard)
	adminOp.PUT("/:id", h.Node.Update)
	adminOp.POST("/test", h.Node.TestConnection)
	// VM Actions
	adminOp.POST("/:id/vms/:vmid/start", h.Node.StartVM)
	adminOp.POST("/:id/vms/:vmid/stop", h.Node.StopVM)
	adminOp.POST("/:id/vms/:vmid/shutdown", h.Node.ShutdownVM)
	adminOp.POST("/:id/vms/:vmid/suspend", h.Node.SuspendVM)
	adminOp.POST("/:id/vms/:vmid/resume", h.Node.ResumeVM)

	// Snapshots
	nodes.GET("/:id/vms/:vmid/snapshots", h.Node.ListSnapshots)
	adminOp.POST("/:id/vms/:vmid/snapshots", h.Node.CreateSnapshot)
	adminOp.DELETE("/:id/vms/:vmid/snapshots/:snapname", h.Node.DeleteSnapshot)
	adminOp.POST("/:id/vms/:vmid/snapshots/:snapname/rollback", h.Node.RollbackSnapshot)

	// Console
	adminOp.POST("/:id/vms/:vmid/vncproxy", h.Node.GetVNCProxy)

	adminOp.POST("/:id/backup", h.Backup.CreateBackup)
	adminOp.POST("/:id/vzdump", h.Backup.CreateVzdumpBackup)
	adminOp.POST("/:id/backup-schedules", h.Schedule.CreateSchedule)
	adminOp.POST("/:id/sync-content", h.Node.SyncContent)
	adminOp.PUT("/:id/network/:iface/alias", h.Node.SetNetworkAlias)
	adminOp.POST("/:id/tags", h.Tag.AddTagToNode)
	adminOp.DELETE("/:id/tags/:tagId", h.Tag.RemoveTagFromNode)

	// VM tag assignment (admin/operator)
	adminOp.POST("/:id/vms/:vmid/tags", h.Tag.AddTagToVM)
	adminOp.DELETE("/:id/vms/:vmid/tags/:tagId", h.Tag.RemoveTagFromVM)

	// Admin only
	admin := nodes.Group("")
	admin.Use(middleware.RequireRole(model.RoleAdmin))
	admin.DELETE("/:id", h.Node.Delete)
	admin.GET("/:id/storage/debug", h.Node.DebugStorage)

	// Backups (non-node-scoped)
	backups := protected.Group("/backups")
	backups.GET("", h.Backup.ListAll)
	backups.GET("/:id", h.Backup.GetBackup)
	backups.GET("/:id/files", h.Backup.GetBackupFiles)
	backups.GET("/:id/files/*", h.Backup.GetBackupFile)
	backups.GET("/:id/diff", h.Backup.DiffBackup)
	backups.GET("/:id/download", h.Backup.DownloadBackup)

	backupsAdmin := backups.Group("")
	backupsAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	backupsAdmin.POST("/:id/restore", h.Backup.RestoreBackup)
	backupsAdmin.DELETE("/:id", h.Backup.DeleteBackup)

	// Backup Schedules (non-node-scoped)
	schedules := protected.Group("/backup-schedules")
	schedulesAdmin := schedules.Group("")
	schedulesAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	schedulesAdmin.PUT("/:id", h.Schedule.UpdateSchedule)
	schedulesAdmin.DELETE("/:id", h.Schedule.DeleteSchedule)

	// Tags
	tags := protected.Group("/tags")
	tags.GET("", h.Tag.ListTags)
	tags.GET("/:id/vms", h.Tag.GetVMsByTag)

	tagsAdmin := tags.Group("")
	tagsAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	tagsAdmin.POST("", h.Tag.CreateTag)
	tagsAdmin.POST("/:id/bulk-assign", h.Tag.BulkAssignTag)
	tagsAdmin.POST("/:id/bulk-remove", h.Tag.BulkRemoveTag)

	tagsAdminOnly := tags.Group("")
	tagsAdminOnly.Use(middleware.RequireRole(model.RoleAdmin))
	tagsAdminOnly.DELETE("/:id", h.Tag.DeleteTag)

	// Sync Center (cluster-wide ISO + tag sync)
	if h.SyncCenter != nil {
		protected.GET("/isos", h.SyncCenter.ListClusterISOs)

		syncCenterAdmin := protected.Group("")
		syncCenterAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		syncCenterAdmin.POST("/tags/sync-all", h.SyncCenter.SyncAllTags)
	}

	// Users (admin only)
	users := protected.Group("/users")
	usersAdmin := users.Group("")
	usersAdmin.Use(middleware.RequireRole(model.RoleAdmin))
	usersAdmin.GET("", h.User.List)
	usersAdmin.GET("/:id", h.User.GetByID)
	usersAdmin.POST("", h.User.Create)
	usersAdmin.PUT("/:id", h.User.Update)
	usersAdmin.DELETE("/:id", h.User.Delete)

	// Password change (admin + self) - protected but not admin-only
	users.POST("/:id/password", h.User.ChangePassword)

	// Password Policy (admin only for updates, all authenticated for read)
	if h.PasswordPolicy != nil {
		policyRoutes := protected.Group("/password-policy")
		policyRoutes.GET("", h.PasswordPolicy.Get)
		policyAdmin := policyRoutes.Group("")
		policyAdmin.Use(middleware.RequireRole(model.RoleAdmin))
		policyAdmin.PUT("", h.PasswordPolicy.Update)
	}

	// DR - node-scoped (read)
	nodes.GET("/:id/dr/profile", h.DR.GetLatestProfile)
	nodes.GET("/:id/dr/readiness", h.DR.GetReadiness)
	nodes.GET("/:id/dr/runbooks", h.DR.ListRunbooks)

	// DR - node-scoped (admin/operator)
	adminOp.POST("/:id/dr/profile", h.DR.CollectProfile)
	adminOp.POST("/:id/dr/readiness", h.DR.CalculateReadiness)
	adminOp.POST("/:id/dr/runbooks", h.DR.GenerateRunbook)

	// DR (non-node-scoped)
	dr := protected.Group("/dr")
	dr.GET("/runbooks/:id", h.DR.GetRunbook)
	dr.GET("/scores", h.DR.ListAllScores)

	drAdmin := dr.Group("")
	drAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	drAdmin.PUT("/runbooks/:id", h.DR.UpdateRunbook)
	drAdmin.POST("/simulate", h.DR.SimulateDR)

	drAdminOnly := dr.Group("")
	drAdminOnly.Use(middleware.RequireRole(model.RoleAdmin))
	drAdminOnly.DELETE("/runbooks/:id", h.DR.DeleteRunbook)

	// Notifications (admin only)
	notifications := protected.Group("/notifications")
	notifAdmin := notifications.Group("")
	notifAdmin.Use(middleware.RequireRole(model.RoleAdmin))
	notifAdmin.GET("/channels", h.Notification.ListChannels)
	notifAdmin.POST("/channels", h.Notification.CreateChannel)
	notifAdmin.GET("/channels/:id", h.Notification.GetChannel)
	notifAdmin.PUT("/channels/:id", h.Notification.UpdateChannel)
	notifAdmin.DELETE("/channels/:id", h.Notification.DeleteChannel)
	notifAdmin.POST("/channels/:id/test", h.Notification.TestChannel)

	notifReadable := notifications.Group("")
	notifReadable.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	notifReadable.GET("/history", h.Notification.ListHistory)

	// Alert rules
	alerts := protected.Group("/alerts")
	alertsReadable := alerts.Group("")
	alertsReadable.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	alertsReadable.GET("/rules", h.Notification.ListAlertRules)
	alertsReadable.GET("/rules/:id", h.Notification.GetAlertRule)

	alertsAdmin := alerts.Group("")
	alertsAdmin.Use(middleware.RequireRole(model.RoleAdmin))
	alertsAdmin.POST("/rules", h.Notification.CreateAlertRule)
	alertsAdmin.PUT("/rules/:id", h.Notification.UpdateAlertRule)
	alertsAdmin.DELETE("/rules/:id", h.Notification.DeleteAlertRule)

	// Chat (all authenticated roles)
	if h.Chat != nil {
		chat := protected.Group("/chat")
		chat.POST("", h.Chat.Chat)
		chat.GET("/conversations", h.Chat.ListConversations)
		chat.GET("/conversations/:id", h.Chat.GetConversation)
		chat.GET("/conversations/:id/messages", h.Chat.GetMessages)
		chat.DELETE("/conversations/:id", h.Chat.DeleteConversation)
	}

	// Migrations
	if h.Migration != nil {
		migrations := protected.Group("/migrations")
		migrations.GET("", h.Migration.List)
		migrations.GET("/:id", h.Migration.Get)
		migrations.GET("/:id/logs", h.Migration.GetLogs)

		migrationsAdmin := migrations.Group("")
		migrationsAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		migrationsAdmin.POST("", h.Migration.Start)
		migrationsAdmin.POST("/:id/cancel", h.Migration.Cancel)
		migrationsAdmin.DELETE("/:id", h.Migration.Delete)

		// Node-scoped migrations
		nodes.GET("/:id/migrations", h.Migration.ListByNode)
	}

	// Escalation policies + incidents
	if h.Escalation != nil {
		escalation := protected.Group("/escalation")

		escalationReadable := escalation.Group("")
		escalationReadable.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		escalationReadable.GET("/policies", h.Escalation.ListPolicies)
		escalationReadable.GET("/policies/:id", h.Escalation.GetPolicy)
		escalationReadable.GET("/incidents", h.Escalation.ListIncidents)
		escalationReadable.GET("/incidents/:id", h.Escalation.GetIncident)
		escalationReadable.POST("/incidents/:id/acknowledge", h.Escalation.AcknowledgeIncident)
		escalationReadable.POST("/incidents/:id/resolve", h.Escalation.ResolveIncident)

		escalationAdmin := escalation.Group("")
		escalationAdmin.Use(middleware.RequireRole(model.RoleAdmin))
		escalationAdmin.POST("/policies", h.Escalation.CreatePolicy)
		escalationAdmin.PUT("/policies/:id", h.Escalation.UpdatePolicy)
		escalationAdmin.DELETE("/policies/:id", h.Escalation.DeletePolicy)
	}

	// Anomalies
	if h.Anomaly != nil {
		anomalies := protected.Group("/anomalies")
		anomalies.GET("", h.Anomaly.ListUnresolved)
		anomaliesAdmin := anomalies.Group("")
		anomaliesAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		anomaliesAdmin.POST("/:id/resolve", h.Anomaly.Resolve)

		// Node-scoped anomalies
		nodes.GET("/:id/anomalies", h.Anomaly.ListByNode)
	}

	// Predictions
	if h.Prediction != nil {
		predictions := protected.Group("/predictions")
		predictions.GET("", h.Prediction.ListCritical)

		// Node-scoped predictions
		nodes.GET("/:id/predictions", h.Prediction.ListByNode)
	}

	// Morning Briefings
	if h.Briefing != nil {
		briefings := protected.Group("/briefings")
		briefings.GET("", h.Briefing.List)
		briefings.GET("/latest", h.Briefing.GetLatest)
		briefings.GET("/live", h.Briefing.GetLiveSummary)
	}

	// Agent Approvals
	if h.Approval != nil {
		approvals := protected.Group("/approvals")
		approvals.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		approvals.GET("", h.Approval.ListPending)
		approvals.POST("/:id/approve", h.Approval.Approve)
		approvals.POST("/:id/reject", h.Approval.Reject)
	}

	// Telegram link
	if h.Telegram != nil {
		tg := protected.Group("/telegram")
		tg.POST("/link", h.Telegram.LinkTelegram)
		tg.GET("/status", h.Telegram.GetTelegramStatus)
		tg.DELETE("/unlink", h.Telegram.UnlinkTelegram)
	}

	// Drift Detection
	if h.Drift != nil {
		driftRoutes := protected.Group("/drift")
		driftRoutes.GET("", h.Drift.ListAll)
		driftRoutes.POST("/compare-nodes", h.Drift.CompareNodes)
		driftRoutes.POST("/:id/accept", h.Drift.AcceptBaseline)
		driftRoutes.POST("/:id/ignore", h.Drift.IgnoreDrift)

		// Node-scoped drift
		nodes.GET("/:id/drift", h.Drift.ListByNode)
		adminOp.POST("/:id/drift/check", h.Drift.TriggerCheck)
	}

	// Environments
	if h.Environment != nil {
		envRoutes := protected.Group("/environments")
		envRoutes.GET("", h.Environment.List)
		envRoutes.GET("/:id", h.Environment.Get)

		envAdmin := envRoutes.Group("")
		envAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		envAdmin.POST("", h.Environment.Create)
		envAdmin.PUT("/:id", h.Environment.Update)

		envAdminOnly := envRoutes.Group("")
		envAdminOnly.Use(middleware.RequireRole(model.RoleAdmin))
		envAdminOnly.DELETE("/:id", h.Environment.Delete)

		// Node environment assignment
		adminOp.PUT("/:id/environment", h.Environment.AssignNode)
	}

	// Update Intelligence
	if h.Update != nil {
		updateRoutes := protected.Group("/updates")
		updateRoutes.GET("", h.Update.ListAll)

		// Node-scoped updates
		nodes.GET("/:id/updates", h.Update.ListByNode)
		adminOp.POST("/:id/updates/check", h.Update.TriggerCheck)
	}

	// Resource Right-Sizing
	if h.Rightsizing != nil {
		rsRoutes := protected.Group("/rightsizing")
		rsRoutes.GET("", h.Rightsizing.ListAll)

		// Node-scoped rightsizing
		nodes.GET("/:id/rightsizing", h.Rightsizing.ListByNode)
		adminOp.POST("/:id/rightsizing/analyze", h.Rightsizing.TriggerAnalysis)
	}

	// SSH Key Management
	if h.SSHKey != nil {
		// Node-scoped SSH keys
		nodes.GET("/:id/ssh-keys", h.SSHKey.ListByNode)
		nodes.GET("/:id/ssh-keys/rotation", h.SSHKey.GetRotationSchedule)

		sshAdmin := nodes.Group("")
		sshAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		sshAdmin.POST("/:id/ssh-keys", h.SSHKey.Generate)
		sshAdmin.POST("/:id/ssh-keys/:keyId/deploy", h.SSHKey.Deploy)
		sshAdmin.POST("/:id/ssh-keys/:keyId/trust", h.SSHKey.TrustAll)
		sshAdmin.POST("/:id/ssh-keys/rotate", h.SSHKey.Rotate)
		sshAdmin.DELETE("/:id/ssh-keys/:keyId", h.SSHKey.Delete)
		sshAdmin.POST("/:id/ssh-keys/rotation", h.SSHKey.CreateRotationSchedule)
	}

	// API Gateway Management
	if h.Gateway != nil {
		gwRoutes := protected.Group("/gateway")
		gwRoutes.GET("/tokens", h.Gateway.ListTokens)

		gwOperator := gwRoutes.Group("")
		gwOperator.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		gwOperator.POST("/tokens", h.Gateway.CreateToken)
		gwOperator.POST("/tokens/:id/revoke", h.Gateway.RevokeToken)
		gwOperator.DELETE("/tokens/:id", h.Gateway.DeleteToken)

		gwAdmin := gwRoutes.Group("")
		gwAdmin.Use(middleware.RequireRole(model.RoleAdmin))
		gwAdmin.GET("/audit", h.Gateway.ListAuditLog)
	}

	// Topology
	if h.Topology != nil {
		protected.GET("/topology", h.Topology.GetTopology)
	}

	// Brain (Wissensbasis)
	if h.Brain != nil {
		brainRoutes := protected.Group("/brain")
		brainRoutes.GET("", h.Brain.List)
		brainRoutes.GET("/search", h.Brain.Search)

		brainAdmin := brainRoutes.Group("")
		brainAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		brainAdmin.POST("", h.Brain.Create)
		brainAdmin.DELETE("/:id", h.Brain.Delete)
	}

	// Reflexes
	if h.Reflex != nil {
		reflexRoutes := protected.Group("/reflexes")
		reflexRoutes.GET("", h.Reflex.List)
		reflexRoutes.GET("/:id", h.Reflex.Get)

		reflexAdmin := reflexRoutes.Group("")
		reflexAdmin.Use(middleware.RequireRole(model.RoleAdmin))
		reflexAdmin.POST("", h.Reflex.Create)
		reflexAdmin.PUT("/:id", h.Reflex.Update)
		reflexAdmin.DELETE("/:id", h.Reflex.Delete)
	}

	// Agent Config
	if h.AgentConfig != nil {
		agentRoutes := protected.Group("/agent")
		agentRoutes.Use(middleware.RequireRole(model.RoleAdmin))
		agentRoutes.GET("/config", h.AgentConfig.GetConfig)
		agentRoutes.PUT("/config", h.AgentConfig.UpdateConfig)
		agentRoutes.GET("/models", h.AgentConfig.GetModels)
	}

	// Security Events
	if h.Security != nil {
		security := protected.Group("/security")
		security.GET("/events", h.Security.ListUnacknowledged)
		security.GET("/events/recent", h.Security.ListRecent)
		security.GET("/events/stats", h.Security.GetStats)

		security.GET("/mode", h.Security.GetMode)

		securityAdmin := security.Group("")
		securityAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		securityAdmin.POST("/events/:id/acknowledge", h.Security.Acknowledge)
		securityAdmin.PUT("/mode", h.Security.SetMode)

		// Node-scoped security events
		nodes.GET("/:id/security/events", h.Security.ListByNode)
	}

	// Backup Recovery Guide
	backups.GET("/:id/recovery-guide", h.Backup.GetRecoveryGuide)

	// Bulk VM Actions
	adminOp.POST("/:id/vms/bulk", h.Node.BulkVMAction)

	// Tag Sync from Proxmox
	adminOp.POST("/:id/tags/sync", h.Node.SyncTags)

	// VM Cockpit
	if h.VMCockpit != nil {
		vmCockpit := adminOp.Group("/:id/vms/:vmid/cockpit")
		vmCockpit.GET("/osinfo", h.VMCockpit.GetOSInfo)
		vmCockpit.POST("/exec", h.VMCockpit.ExecCommand)
		vmCockpit.GET("/processes", h.VMCockpit.GetProcesses)
		vmCockpit.GET("/services", h.VMCockpit.GetServices)
		vmCockpit.GET("/ports", h.VMCockpit.GetPorts)
		vmCockpit.GET("/disk", h.VMCockpit.GetDiskUsage)
		vmCockpit.POST("/services/action", h.VMCockpit.ServiceAction)
		vmCockpit.POST("/processes/kill", h.VMCockpit.KillProcess)
		vmCockpit.GET("/shell", h.VMCockpit.HandleShell)

		// File operations (Phase 2)
		vmCockpit.GET("/files", h.VMCockpit.ListFiles)
		vmCockpit.GET("/files/read", h.VMCockpit.ReadFile)
		vmCockpit.POST("/files/write", h.VMCockpit.WriteFile)
		vmCockpit.POST("/files/upload", h.VMCockpit.UploadFile)
		vmCockpit.DELETE("/files", h.VMCockpit.DeleteFile)
		vmCockpit.POST("/files/mkdir", h.VMCockpit.MakeDir)
	}

	// VM Permissions (Admin only)
	if h.VMPermission != nil {
		vmPerms := protected.Group("/vm-permissions")
		vmPerms.Use(middleware.RequireRole(model.RoleAdmin))
		vmPerms.GET("", h.VMPermission.List)
		vmPerms.POST("", h.VMPermission.Create)
		vmPerms.PUT("/upsert", h.VMPermission.Upsert)
		vmPerms.PUT("/:id", h.VMPermission.Update)
		vmPerms.DELETE("/:id", h.VMPermission.Delete)
		vmPerms.GET("/effective", h.VMPermission.GetEffective)
		vmPerms.GET("/all", h.VMPermission.ListAllPermissions)
	}

	// VM Groups (Admin only)
	if h.VMGroup != nil {
		vmGroups := protected.Group("/vm-groups")
		vmGroups.Use(middleware.RequireRole(model.RoleAdmin))
		vmGroups.GET("", h.VMGroup.List)
		vmGroups.GET("/:id", h.VMGroup.Get)
		vmGroups.POST("", h.VMGroup.Create)
		vmGroups.PUT("/:id", h.VMGroup.Update)
		vmGroups.DELETE("/:id", h.VMGroup.Delete)
		vmGroups.GET("/:id/members", h.VMGroup.ListMembers)
		vmGroups.POST("/:id/members", h.VMGroup.AddMember)
		vmGroups.DELETE("/:id/members", h.VMGroup.RemoveMember)
	}

	// VM Health, Rightsizing, Anomalies, Snapshot Policies, Dependencies (Phase 4)
	if h.VMHealth != nil {
		// Health scores
		nodes.GET("/:id/vms/:vmid/health", h.VMHealth.GetVMHealth)
		nodes.GET("/:id/vm-health", h.VMHealth.GetAllVMHealth)

		// VM-level rightsizing
		nodes.GET("/:id/vms/:vmid/rightsizing", h.VMHealth.GetVMRightsizing)

		// VM-level anomalies
		nodes.GET("/:id/vms/:vmid/anomalies", h.VMHealth.GetVMAnomalies)

		// Snapshot policies
		nodes.GET("/:id/vms/:vmid/snapshot-policies", h.VMHealth.ListSnapshotPolicies)
		adminOp.POST("/:id/vms/:vmid/snapshot-policies", h.VMHealth.CreateSnapshotPolicy)
		adminOp.PUT("/:id/vms/:vmid/snapshot-policies/:policyId", h.VMHealth.UpdateSnapshotPolicy)
		adminOp.DELETE("/:id/vms/:vmid/snapshot-policies/:policyId", h.VMHealth.DeleteSnapshotPolicy)

		// Scheduled actions
		nodes.GET("/:id/vms/:vmid/scheduled-actions", h.VMHealth.ListScheduledActions)
		adminOp.POST("/:id/vms/:vmid/scheduled-actions", h.VMHealth.CreateScheduledAction)
		adminOp.DELETE("/:id/vms/:vmid/scheduled-actions/:actionId", h.VMHealth.DeleteScheduledAction)

		// VM Dependencies (non-node-scoped)
		vmDeps := protected.Group("/vm-dependencies")
		vmDeps.GET("", h.VMHealth.ListAllDependencies)
		vmDepsAdmin := vmDeps.Group("")
		vmDepsAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		vmDepsAdmin.POST("", h.VMHealth.CreateDependency)
		vmDepsAdmin.DELETE("/:depId", h.VMHealth.DeleteDependency)

		// VM Dependencies (node-scoped)
		nodes.GET("/:id/vms/:vmid/dependencies", h.VMHealth.ListVMDependencies)
	}

	// Log WebSocket (top-level, auth handled internally)
	if h.LogStream != nil {
		e.GET("/api/v1/ws/logs", h.LogStream.HandleWS)
	}

	// Log routes
	if h.LogAnalysis != nil {
		logs := protected.Group("/logs")
		logs.POST("/analyze", h.LogAnalysis.Analyze)
		logs.GET("/analyses", h.LogAnalysis.ListAnalyses)

		if h.LogExport != nil {
			logs.GET("/export", h.LogExport.Export)
		}

		// Report schedules (operator+)
		if h.LogReportSchedule != nil {
			schedOp := logs.Group("/report-schedules")
			schedOp.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
			schedOp.POST("", h.LogReportSchedule.Create)
			schedOp.GET("", h.LogReportSchedule.List)
			schedOp.PUT("/:id", h.LogReportSchedule.Update)
			schedOp.DELETE("/:id", h.LogReportSchedule.Delete)
		}

		// Node-scoped log routes
		nodes.GET("/:id/log-anomalies", h.LogAnalysis.ListAnomalies)
		nodes.GET("/:id/log-bookmarks", h.LogBookmark.ListByNode)
		nodes.GET("/:id/log-sources", h.LogSource.ListByNode)

		// Anomaly actions (operator+)
		logAnomalies := protected.Group("/log-anomalies")
		logAnomalies.GET("/:id", h.LogAnalysis.GetAnomaly)
		logAnomaliesOp := logAnomalies.Group("")
		logAnomaliesOp.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		logAnomaliesOp.POST("/:id/acknowledge", h.LogAnalysis.Acknowledge)

		// Bookmarks
		bookmarks := protected.Group("/log-bookmarks")
		bookmarks.POST("", h.LogBookmark.Create)
		bookmarks.DELETE("/:id", h.LogBookmark.Delete)

		// Source management (operator+)
		logSourcesOp := nodes.Group("/:id/log-sources")
		logSourcesOp.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		logSourcesOp.PUT("", h.LogSource.Update)
	}

	// Network routes
	if h.NetworkScan != nil {
		nodes.GET("/:id/network-scans", h.NetworkScan.ListByNode)
		nodes.GET("/:id/network-devices", h.NetworkDevice.ListByNode)
		nodes.GET("/:id/network-anomalies", h.NetworkAnomaly.ListByNode)
		nodes.GET("/:id/scan-baselines", h.ScanBaseline.ListByNode)

		netScans := protected.Group("/network-scans")
		netScans.GET("/:id", h.NetworkScan.Get)
		netScans.GET("/:id1/diff/:id2", h.NetworkScan.Diff)

		// Trigger scan (operator+)
		netScansOp := nodes.Group("/:id/network-scans")
		netScansOp.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		netScansOp.POST("", h.NetworkScan.Trigger)

		netDevices := protected.Group("/network-devices")
		netDevices.PUT("/:id", h.NetworkDevice.Update)

		netAnomaliesOp := protected.Group("/network-anomalies")
		netAnomaliesOp.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		netAnomaliesOp.POST("/:id/acknowledge", h.NetworkAnomaly.Acknowledge)

		baselinesOp := nodes.Group("/:id/scan-baselines")
		baselinesOp.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		baselinesOp.POST("", h.ScanBaseline.Create)

		baselinesMgmtOp := protected.Group("/scan-baselines")
		baselinesMgmtOp.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
		baselinesMgmtOp.PUT("/:id", h.ScanBaseline.Update)
		baselinesMgmtOp.DELETE("/:id", h.ScanBaseline.Delete)
		baselinesMgmtOp.POST("/:id/activate", h.ScanBaseline.Activate)
	}

	// WebSocket
	e.GET("/api/v1/ws", h.WS.HandleWS)
}
