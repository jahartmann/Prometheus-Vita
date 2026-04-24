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
	Permission     *handler.PermissionHandler
	VMCockpit      *handler.VMCockpitHandler
	VMPermission   *handler.VMPermissionHandler
	VMGroup        *handler.VMGroupHandler
	VMHealth       *handler.VMHealthHandler
	Operations     *handler.OperationsHandler

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

func SetupRouter(e *echo.Echo, cfg *config.Config, jwtSvc *auth.JWTService, h Handlers, gatewaySvc *gateway.Service, redisClient *redis.Client, auditRepo repository.AuditRepository, userRepo repository.UserRepository, rolePermissionRepo repository.RolePermissionRepository) {
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
	authGroup.POST("/invitations/accept", h.User.AcceptInvitation)

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(jwtSvc))
	if userRepo != nil {
		protected.Use(middleware.MustChangePassword(userRepo))
	}
	if rolePermissionRepo != nil {
		protected.Use(middleware.LoadRolePermissions(rolePermissionRepo))
	}

	// Auth - protected
	protected.GET("/auth/me", h.Auth.Me)

	// Cluster-level storage (aggregates all nodes)
	protected.GET("/storage", h.Node.GetClusterStorage, middleware.RequirePermission(model.PermissionNodesRead))

	// Operations aggregation (read-only Phase 6 APIs)
	if h.Operations != nil {
		protected.GET("/tasks", h.Operations.ListTasks, middleware.RequirePermission(model.PermissionNodesRead))
		protected.GET("/timeline", h.Operations.Timeline, middleware.RequirePermission(model.PermissionAuditRead))
		protected.POST("/rca/analyze", h.Operations.AnalyzeRCA, middleware.RequirePermission(model.PermissionSecurityRead))
		protected.GET("/knowledge-graph", h.Operations.KnowledgeGraph, middleware.RequirePermission(model.PermissionNodesRead))
		protected.POST("/reports/generate", h.Operations.GenerateReport, middleware.RequirePermission(model.PermissionSecurityRead))
	}

	// Cluster Dashboard
	if h.Cluster != nil {
		cluster := protected.Group("/cluster")
		cluster.Use(middleware.RequirePermission(model.PermissionNodesRead))
		cluster.GET("/summary", h.Cluster.GetSummary)
		cluster.GET("/history", h.Cluster.GetHistory)
	}

	// Nodes
	nodes := protected.Group("/nodes")
	nodes.GET("", h.Node.List, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id", h.Node.Get, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/status", h.Node.GetStatus, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/vms", h.Node.GetVMs, middleware.RequirePermission(model.PermissionVMsRead))
	nodes.GET("/:id/storage", h.Node.GetStorage, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/network", h.Node.GetNetworkInterfaces, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/ports", h.Node.GetPorts, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/disks", h.Node.GetDisks, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/tags", h.Tag.GetNodeTags, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/diagnose", h.Node.DiagnoseConnectivity, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/isos", h.Node.ListISOs, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/templates", h.Node.ListTemplates, middleware.RequirePermission(model.PermissionNodesRead))

	// VM tags (read)
	nodes.GET("/:id/vms/:vmid/tags", h.Tag.GetVMTags, middleware.RequirePermission(model.PermissionVMsRead))

	// Node backups (read)
	nodes.GET("/:id/backups", h.Backup.ListBackups, middleware.RequirePermission(model.PermissionBackupsRead))
	nodes.GET("/:id/backup-schedules", h.Schedule.ListSchedules, middleware.RequirePermission(model.PermissionBackupsRead))

	// Node metrics
	nodes.GET("/:id/metrics", h.Metrics.GetMetricsHistory, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/metrics/summary", h.Metrics.GetMetricsSummary, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/network-summary", h.Metrics.GetNodeNetworkSummary, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/vms/:vmid/metrics", h.Metrics.GetVMMetricsHistory, middleware.RequirePermission(model.PermissionVMsRead))
	nodes.GET("/:id/vms/:vmid/network-summary", h.Metrics.GetVMNetworkSummary, middleware.RequirePermission(model.PermissionVMsRead))
	nodes.GET("/:id/rrd", h.Metrics.GetNodeRRDData, middleware.RequirePermission(model.PermissionNodesRead))
	nodes.GET("/:id/vms/:vmid/rrd", h.Metrics.GetVMRRDData, middleware.RequirePermission(model.PermissionVMsRead))

	// Cluster-wide network summary
	protected.GET("/network-summary", h.Metrics.GetClusterNetworkSummary, middleware.RequirePermission(model.PermissionNodesRead))

	// Node logs
	if h.Log != nil {
		nodes.GET("/:id/logs", h.Log.GetLogs, middleware.RequirePermission(model.PermissionLogsRead))
	}

	// PBS (read)
	nodes.GET("/:id/pbs/datastores", h.PBS.GetDatastores, middleware.RequirePermission(model.PermissionBackupsRead))
	nodes.GET("/:id/pbs/backup-jobs", h.PBS.GetBackupJobs, middleware.RequirePermission(model.PermissionBackupsRead))

	// Permission-gated node operations
	adminOp := nodes.Group("")
	adminOp.POST("", h.Node.Create, middleware.RequirePermission(model.PermissionNodesWrite))
	adminOp.POST("/onboard", h.Node.Onboard, middleware.RequirePermission(model.PermissionNodesWrite))
	adminOp.PUT("/:id", h.Node.Update, middleware.RequirePermission(model.PermissionNodesWrite))
	adminOp.POST("/test", h.Node.TestConnection, middleware.RequirePermission(model.PermissionNodesWrite))
	// VM Actions
	adminOp.POST("/:id/vms/:vmid/start", h.Node.StartVM, middleware.RequirePermission(model.PermissionVMPower))
	adminOp.POST("/:id/vms/:vmid/stop", h.Node.StopVM, middleware.RequirePermission(model.PermissionVMPower))
	adminOp.POST("/:id/vms/:vmid/shutdown", h.Node.ShutdownVM, middleware.RequirePermission(model.PermissionVMPower))
	adminOp.POST("/:id/vms/:vmid/suspend", h.Node.SuspendVM, middleware.RequirePermission(model.PermissionVMPower))
	adminOp.POST("/:id/vms/:vmid/resume", h.Node.ResumeVM, middleware.RequirePermission(model.PermissionVMPower))

	// Snapshots
	nodes.GET("/:id/vms/:vmid/snapshots", h.Node.ListSnapshots, middleware.RequirePermission(model.PermissionVMsRead))
	adminOp.POST("/:id/vms/:vmid/snapshots", h.Node.CreateSnapshot, middleware.RequirePermission(model.PermissionVMsWrite))
	adminOp.DELETE("/:id/vms/:vmid/snapshots/:snapname", h.Node.DeleteSnapshot, middleware.RequirePermission(model.PermissionVMsWrite))
	adminOp.POST("/:id/vms/:vmid/snapshots/:snapname/rollback", h.Node.RollbackSnapshot, middleware.RequirePermission(model.PermissionVMsWrite))

	// Console
	adminOp.POST("/:id/vms/:vmid/vncproxy", h.Node.GetVNCProxy, middleware.RequirePermission(model.PermissionVMsWrite))

	adminOp.POST("/:id/backup", h.Backup.CreateBackup, middleware.RequirePermission(model.PermissionBackupsCreate))
	adminOp.POST("/:id/vzdump", h.Backup.CreateVzdumpBackup, middleware.RequirePermission(model.PermissionBackupsCreate))
	adminOp.POST("/:id/backup-schedules", h.Schedule.CreateSchedule, middleware.RequirePermission(model.PermissionBackupsCreate))
	adminOp.POST("/:id/sync-content", h.Node.SyncContent, middleware.RequirePermission(model.PermissionNodesWrite))
	adminOp.PUT("/:id/network/:iface/alias", h.Node.SetNetworkAlias, middleware.RequirePermission(model.PermissionNodesWrite))
	adminOp.POST("/:id/tags", h.Tag.AddTagToNode, middleware.RequirePermission(model.PermissionNodesWrite))
	adminOp.DELETE("/:id/tags/:tagId", h.Tag.RemoveTagFromNode, middleware.RequirePermission(model.PermissionNodesWrite))

	// VM tag assignment (admin/operator)
	adminOp.POST("/:id/vms/:vmid/tags", h.Tag.AddTagToVM, middleware.RequirePermission(model.PermissionVMsWrite))
	adminOp.DELETE("/:id/vms/:vmid/tags/:tagId", h.Tag.RemoveTagFromVM, middleware.RequirePermission(model.PermissionVMsWrite))

	// Permission-gated node administration
	admin := nodes.Group("")
	admin.DELETE("/:id", h.Node.Delete, middleware.RequirePermission(model.PermissionNodesDelete))
	admin.GET("/:id/storage/debug", h.Node.DebugStorage, middleware.RequirePermission(model.PermissionNodesRead))

	// Backups (non-node-scoped)
	backups := protected.Group("/backups")
	backups.GET("", h.Backup.ListAll, middleware.RequirePermission(model.PermissionBackupsRead))
	backups.GET("/:id", h.Backup.GetBackup, middleware.RequirePermission(model.PermissionBackupsRead))
	backups.GET("/:id/files", h.Backup.GetBackupFiles, middleware.RequirePermission(model.PermissionBackupsRead))
	backups.GET("/:id/files/*", h.Backup.GetBackupFile, middleware.RequirePermission(model.PermissionBackupsRead))
	backups.GET("/:id/diff", h.Backup.DiffBackup, middleware.RequirePermission(model.PermissionBackupsRead))
	backups.GET("/:id/download", h.Backup.DownloadBackup, middleware.RequirePermission(model.PermissionBackupsRead))

	backupsAdmin := backups.Group("")
	backupsAdmin.POST("/:id/restore", h.Backup.RestoreBackup, middleware.RequirePermission(model.PermissionBackupsRestore))
	backupsAdmin.DELETE("/:id", h.Backup.DeleteBackup, middleware.RequirePermission(model.PermissionBackupsDelete))

	// Backup Schedules (non-node-scoped)
	schedules := protected.Group("/backup-schedules")
	schedulesAdmin := schedules.Group("")
	schedulesAdmin.PUT("/:id", h.Schedule.UpdateSchedule, middleware.RequirePermission(model.PermissionBackupsCreate))
	schedulesAdmin.DELETE("/:id", h.Schedule.DeleteSchedule, middleware.RequirePermission(model.PermissionBackupsDelete))

	// Tags
	tags := protected.Group("/tags")
	tags.GET("", h.Tag.ListTags, middleware.RequirePermission(model.PermissionNodesRead))
	tags.GET("/:id/vms", h.Tag.GetVMsByTag, middleware.RequirePermission(model.PermissionVMsRead))

	tagsAdmin := tags.Group("")
	tagsAdmin.Use(middleware.RequirePermission(model.PermissionNodesWrite))
	tagsAdmin.POST("", h.Tag.CreateTag)
	tagsAdmin.POST("/:id/bulk-assign", h.Tag.BulkAssignTag)
	tagsAdmin.POST("/:id/bulk-remove", h.Tag.BulkRemoveTag)

	tagsAdminOnly := tags.Group("")
	tagsAdminOnly.Use(middleware.RequirePermission(model.PermissionNodesWrite))
	tagsAdminOnly.DELETE("/:id", h.Tag.DeleteTag)

	// Sync Center (cluster-wide ISO + tag sync)
	if h.SyncCenter != nil {
		protected.GET("/isos", h.SyncCenter.ListClusterISOs, middleware.RequirePermission(model.PermissionNodesRead))

		syncCenterAdmin := protected.Group("")
		syncCenterAdmin.Use(middleware.RequirePermission(model.PermissionNodesWrite))
		syncCenterAdmin.POST("/tags/sync-all", h.SyncCenter.SyncAllTags)
	}

	// Users
	users := protected.Group("/users")
	usersAdmin := users.Group("")
	usersAdmin.Use(middleware.RequirePermission(model.PermissionUsersManage))
	usersAdmin.GET("", h.User.List)
	usersAdmin.GET("/invitations", h.User.ListInvitations)
	usersAdmin.POST("/invitations", h.User.CreateInvitation)
	usersAdmin.DELETE("/invitations/:id", h.User.DeleteInvitation)
	usersAdmin.GET("/:id", h.User.GetByID)
	usersAdmin.POST("", h.User.Create)
	usersAdmin.PUT("/:id", h.User.Update)
	usersAdmin.DELETE("/:id", h.User.Delete)
	usersAdmin.GET("/:id/sessions", h.User.ListSessions)
	usersAdmin.POST("/:id/sessions/:sessionId/revoke", h.User.RevokeSession)
	usersAdmin.POST("/:id/revoke-access", h.User.RevokeAllAccess)
	usersAdmin.GET("/:id/api-tokens", h.User.ListAPITokens)

	// Password change (admin + self) - protected but not admin-only
	users.POST("/:id/password", h.User.ChangePassword)

	// Password Policy
	if h.PasswordPolicy != nil {
		policyRoutes := protected.Group("/password-policy")
		policyRoutes.GET("", h.PasswordPolicy.Get, middleware.RequirePermission(model.PermissionSettingsManage))
		policyAdmin := policyRoutes.Group("")
		policyAdmin.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		policyAdmin.PUT("", h.PasswordPolicy.Update)
	}

	// Permission catalog
	if h.Permission != nil {
		permissionRoutes := protected.Group("/permissions")
		permissionRoutes.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		permissionRoutes.GET("/catalog", h.Permission.GetCatalog)
		permissionRoutes.PUT("/roles/:role", h.Permission.UpdateRole)
	}

	// DR - node-scoped (read)
	nodes.GET("/:id/dr/profile", h.DR.GetLatestProfile, middleware.RequirePermission(model.PermissionBackupsRead))
	nodes.GET("/:id/dr/readiness", h.DR.GetReadiness, middleware.RequirePermission(model.PermissionBackupsRead))
	nodes.GET("/:id/dr/runbooks", h.DR.ListRunbooks, middleware.RequirePermission(model.PermissionBackupsRead))

	// DR - node-scoped (admin/operator)
	adminOp.POST("/:id/dr/profile", h.DR.CollectProfile, middleware.RequirePermission(model.PermissionBackupsRestore))
	adminOp.POST("/:id/dr/readiness", h.DR.CalculateReadiness, middleware.RequirePermission(model.PermissionBackupsRestore))
	adminOp.POST("/:id/dr/runbooks", h.DR.GenerateRunbook, middleware.RequirePermission(model.PermissionBackupsRestore))

	// DR (non-node-scoped)
	dr := protected.Group("/dr")
	dr.GET("/runbooks/:id", h.DR.GetRunbook, middleware.RequirePermission(model.PermissionBackupsRead))
	dr.GET("/scores", h.DR.ListAllScores, middleware.RequirePermission(model.PermissionBackupsRead))

	drAdmin := dr.Group("")
	drAdmin.Use(middleware.RequirePermission(model.PermissionBackupsRestore))
	drAdmin.PUT("/runbooks/:id", h.DR.UpdateRunbook)
	drAdmin.POST("/simulate", h.DR.SimulateDR)

	drAdminOnly := dr.Group("")
	drAdminOnly.Use(middleware.RequirePermission(model.PermissionBackupsDelete))
	drAdminOnly.DELETE("/runbooks/:id", h.DR.DeleteRunbook)

	// Notifications
	notifications := protected.Group("/notifications")
	notifAdmin := notifications.Group("")
	notifAdmin.Use(middleware.RequirePermission(model.PermissionSettingsManage))
	notifAdmin.GET("/channels", h.Notification.ListChannels)
	notifAdmin.POST("/channels", h.Notification.CreateChannel)
	notifAdmin.GET("/channels/:id", h.Notification.GetChannel)
	notifAdmin.PUT("/channels/:id", h.Notification.UpdateChannel)
	notifAdmin.DELETE("/channels/:id", h.Notification.DeleteChannel)
	notifAdmin.POST("/channels/:id/test", h.Notification.TestChannel)

	notifReadable := notifications.Group("")
	notifReadable.Use(middleware.RequirePermission(model.PermissionSettingsManage))
	notifReadable.GET("/history", h.Notification.ListHistory)

	// Alert rules
	alerts := protected.Group("/alerts")
	alertsReadable := alerts.Group("")
	alertsReadable.Use(middleware.RequirePermission(model.PermissionSecurityRead))
	alertsReadable.GET("/rules", h.Notification.ListAlertRules)
	alertsReadable.GET("/rules/:id", h.Notification.GetAlertRule)

	alertsAdmin := alerts.Group("")
	alertsAdmin.Use(middleware.RequirePermission(model.PermissionSecurityManage))
	alertsAdmin.POST("/rules", h.Notification.CreateAlertRule)
	alertsAdmin.PUT("/rules/:id", h.Notification.UpdateAlertRule)
	alertsAdmin.DELETE("/rules/:id", h.Notification.DeleteAlertRule)

	// Chat (all authenticated roles)
	if h.Chat != nil {
		chat := protected.Group("/chat")
		chat.Use(middleware.RequirePermission(model.PermissionAgentUse))
		chat.POST("", h.Chat.Chat)
		chat.GET("/conversations", h.Chat.ListConversations)
		chat.GET("/conversations/:id", h.Chat.GetConversation)
		chat.GET("/conversations/:id/messages", h.Chat.GetMessages)
		chat.DELETE("/conversations/:id", h.Chat.DeleteConversation)
	}

	// Migrations
	if h.Migration != nil {
		migrations := protected.Group("/migrations")
		migrations.GET("", h.Migration.List, middleware.RequirePermission(model.PermissionVMsRead))
		migrations.GET("/:id", h.Migration.Get, middleware.RequirePermission(model.PermissionVMsRead))
		migrations.GET("/:id/logs", h.Migration.GetLogs, middleware.RequirePermission(model.PermissionVMsRead))

		migrationsAdmin := migrations.Group("")
		migrationsAdmin.Use(middleware.RequirePermission(model.PermissionVMsWrite))
		migrationsAdmin.POST("", h.Migration.Start)
		migrationsAdmin.POST("/:id/cancel", h.Migration.Cancel)
		migrationsAdmin.DELETE("/:id", h.Migration.Delete)

		// Node-scoped migrations
		nodes.GET("/:id/migrations", h.Migration.ListByNode, middleware.RequirePermission(model.PermissionVMsRead))
	}

	// Escalation policies + incidents
	if h.Escalation != nil {
		escalation := protected.Group("/escalation")

		escalationReadable := escalation.Group("")
		escalationReadable.Use(middleware.RequirePermission(model.PermissionSecurityRead))
		escalationReadable.GET("/policies", h.Escalation.ListPolicies)
		escalationReadable.GET("/policies/:id", h.Escalation.GetPolicy)
		escalationReadable.GET("/incidents", h.Escalation.ListIncidents)
		escalationReadable.GET("/incidents/:id", h.Escalation.GetIncident)

		escalationAdmin := escalation.Group("")
		escalationAdmin.Use(middleware.RequirePermission(model.PermissionSecurityManage))
		escalationAdmin.POST("/incidents/:id/acknowledge", h.Escalation.AcknowledgeIncident)
		escalationAdmin.POST("/incidents/:id/resolve", h.Escalation.ResolveIncident)
		escalationAdmin.POST("/policies", h.Escalation.CreatePolicy)
		escalationAdmin.PUT("/policies/:id", h.Escalation.UpdatePolicy)
		escalationAdmin.DELETE("/policies/:id", h.Escalation.DeletePolicy)
	}

	// Anomalies
	if h.Anomaly != nil {
		anomalies := protected.Group("/anomalies")
		anomalies.GET("", h.Anomaly.ListUnresolved, middleware.RequirePermission(model.PermissionSecurityRead))
		anomaliesAdmin := anomalies.Group("")
		anomaliesAdmin.Use(middleware.RequirePermission(model.PermissionSecurityManage))
		anomaliesAdmin.POST("/:id/resolve", h.Anomaly.Resolve)

		// Node-scoped anomalies
		nodes.GET("/:id/anomalies", h.Anomaly.ListByNode, middleware.RequirePermission(model.PermissionSecurityRead))
	}

	// Predictions
	if h.Prediction != nil {
		predictions := protected.Group("/predictions")
		predictions.Use(middleware.RequirePermission(model.PermissionNodesRead))
		predictions.GET("", h.Prediction.ListCritical)

		// Node-scoped predictions
		nodes.GET("/:id/predictions", h.Prediction.ListByNode, middleware.RequirePermission(model.PermissionNodesRead))
	}

	// Morning Briefings
	if h.Briefing != nil {
		briefings := protected.Group("/briefings")
		briefings.Use(middleware.RequirePermission(model.PermissionNodesRead))
		briefings.GET("", h.Briefing.List)
		briefings.GET("/latest", h.Briefing.GetLatest)
		briefings.GET("/live", h.Briefing.GetLiveSummary)
	}

	// Agent Approvals
	if h.Approval != nil {
		approvals := protected.Group("/approvals")
		approvals.Use(middleware.RequirePermission(model.PermissionAgentExecute))
		approvals.GET("", h.Approval.ListPending)
		approvals.POST("/:id/approve", h.Approval.Approve)
		approvals.POST("/:id/reject", h.Approval.Reject)
	}

	// Telegram link
	if h.Telegram != nil {
		tg := protected.Group("/telegram")
		tg.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		tg.POST("/link", h.Telegram.LinkTelegram)
		tg.GET("/status", h.Telegram.GetTelegramStatus)
		tg.DELETE("/unlink", h.Telegram.UnlinkTelegram)
	}

	// Drift Detection
	if h.Drift != nil {
		driftRoutes := protected.Group("/drift")
		driftRoutes.GET("", h.Drift.ListAll, middleware.RequirePermission(model.PermissionNodesRead))
		driftRoutes.POST("/compare-nodes", h.Drift.CompareNodes, middleware.RequirePermission(model.PermissionNodesWrite))
		driftRoutes.POST("/:id/accept", h.Drift.AcceptBaseline, middleware.RequirePermission(model.PermissionNodesWrite))
		driftRoutes.POST("/:id/ignore", h.Drift.IgnoreDrift, middleware.RequirePermission(model.PermissionNodesWrite))

		// Node-scoped drift
		nodes.GET("/:id/drift", h.Drift.ListByNode, middleware.RequirePermission(model.PermissionNodesRead))
		adminOp.POST("/:id/drift/check", h.Drift.TriggerCheck, middleware.RequirePermission(model.PermissionNodesWrite))
	}

	// Environments
	if h.Environment != nil {
		envRoutes := protected.Group("/environments")
		envRoutes.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		envRoutes.GET("", h.Environment.List)
		envRoutes.GET("/:id", h.Environment.Get)

		envAdmin := envRoutes.Group("")
		envAdmin.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		envAdmin.POST("", h.Environment.Create)
		envAdmin.PUT("/:id", h.Environment.Update)

		envAdminOnly := envRoutes.Group("")
		envAdminOnly.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		envAdminOnly.DELETE("/:id", h.Environment.Delete)

		// Node environment assignment
		adminOp.PUT("/:id/environment", h.Environment.AssignNode, middleware.RequirePermission(model.PermissionSettingsManage))
	}

	// Update Intelligence
	if h.Update != nil {
		updateRoutes := protected.Group("/updates")
		updateRoutes.Use(middleware.RequirePermission(model.PermissionNodesRead))
		updateRoutes.GET("", h.Update.ListAll)

		// Node-scoped updates
		nodes.GET("/:id/updates", h.Update.ListByNode, middleware.RequirePermission(model.PermissionNodesRead))
		adminOp.POST("/:id/updates/check", h.Update.TriggerCheck, middleware.RequirePermission(model.PermissionNodesWrite))
	}

	// Resource Right-Sizing
	if h.Rightsizing != nil {
		rsRoutes := protected.Group("/rightsizing")
		rsRoutes.Use(middleware.RequirePermission(model.PermissionVMsRead))
		rsRoutes.GET("", h.Rightsizing.ListAll)

		// Node-scoped rightsizing
		nodes.GET("/:id/rightsizing", h.Rightsizing.ListByNode, middleware.RequirePermission(model.PermissionVMsRead))
		adminOp.POST("/:id/rightsizing/analyze", h.Rightsizing.TriggerAnalysis, middleware.RequirePermission(model.PermissionVMsWrite))
	}

	// SSH Key Management
	if h.SSHKey != nil {
		// Node-scoped SSH keys
		nodes.GET("/:id/ssh-keys", h.SSHKey.ListByNode, middleware.RequirePermission(model.PermissionSettingsManage))
		nodes.GET("/:id/ssh-keys/rotation", h.SSHKey.GetRotationSchedule, middleware.RequirePermission(model.PermissionSettingsManage))

		sshAdmin := nodes.Group("")
		sshAdmin.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		sshAdmin.POST("/trust", h.SSHKey.TrustNodes)
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
		gwRoutes.GET("/tokens", h.Gateway.ListTokens, middleware.RequirePermission(model.PermissionAPITokensManage))

		gwOperator := gwRoutes.Group("")
		gwOperator.Use(middleware.RequirePermission(model.PermissionAPITokensManage))
		gwOperator.POST("/tokens", h.Gateway.CreateToken)
		gwOperator.POST("/tokens/:id/revoke", h.Gateway.RevokeToken)
		gwOperator.DELETE("/tokens/:id", h.Gateway.DeleteToken)

		gwAdmin := gwRoutes.Group("")
		gwAdmin.Use(middleware.RequirePermission(model.PermissionAuditRead))
		gwAdmin.GET("/audit", h.Gateway.ListAuditLog)
	}

	// Topology
	if h.Topology != nil {
		protected.GET("/topology", h.Topology.GetTopology, middleware.RequirePermission(model.PermissionNodesRead))
	}

	// Brain (Wissensbasis)
	if h.Brain != nil {
		brainRoutes := protected.Group("/brain")
		brainRoutes.Use(middleware.RequirePermission(model.PermissionAgentUse))
		brainRoutes.GET("", h.Brain.List)
		brainRoutes.GET("/search", h.Brain.Search)

		brainAdmin := brainRoutes.Group("")
		brainAdmin.Use(middleware.RequirePermission(model.PermissionAgentManage))
		brainAdmin.POST("", h.Brain.Create)
		brainAdmin.DELETE("/:id", h.Brain.Delete)
	}

	// Reflexes
	if h.Reflex != nil {
		reflexRoutes := protected.Group("/reflexes")
		reflexRoutes.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		reflexRoutes.GET("", h.Reflex.List)
		reflexRoutes.GET("/:id", h.Reflex.Get)

		reflexAdmin := reflexRoutes.Group("")
		reflexAdmin.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		reflexAdmin.POST("", h.Reflex.Create)
		reflexAdmin.PUT("/:id", h.Reflex.Update)
		reflexAdmin.DELETE("/:id", h.Reflex.Delete)
	}

	// Agent Config
	if h.AgentConfig != nil {
		agentRoutes := protected.Group("/agent")
		agentRoutes.Use(middleware.RequirePermission(model.PermissionAgentManage))
		agentRoutes.GET("/config", h.AgentConfig.GetConfig)
		agentRoutes.PUT("/config", h.AgentConfig.UpdateConfig)
		agentRoutes.POST("/secrets/:provider/rotate", h.AgentConfig.RotateSecret)
		agentRoutes.DELETE("/secrets/:provider", h.AgentConfig.DeleteSecret)
		agentRoutes.GET("/models", h.AgentConfig.GetModels)
		if h.Chat != nil {
			agentRoutes.GET("/tools", h.Chat.ToolCatalog)
		}
	}

	// Security Events
	if h.Security != nil {
		security := protected.Group("/security")
		security.GET("/events", h.Security.ListUnacknowledged, middleware.RequirePermission(model.PermissionSecurityRead))
		security.GET("/events/recent", h.Security.ListRecent, middleware.RequirePermission(model.PermissionSecurityRead))
		security.GET("/events/stats", h.Security.GetStats, middleware.RequirePermission(model.PermissionSecurityRead))

		security.GET("/mode", h.Security.GetMode, middleware.RequirePermission(model.PermissionSecurityRead))

		securityAdmin := security.Group("")
		securityAdmin.Use(middleware.RequirePermission(model.PermissionSecurityManage))
		securityAdmin.POST("/events/:id/acknowledge", h.Security.Acknowledge)
		securityAdmin.PUT("/mode", h.Security.SetMode)

		// Node-scoped security events
		nodes.GET("/:id/security/events", h.Security.ListByNode, middleware.RequirePermission(model.PermissionSecurityRead))
	}

	// Backup Recovery Guide
	backups.GET("/:id/recovery-guide", h.Backup.GetRecoveryGuide, middleware.RequirePermission(model.PermissionBackupsRead))

	// Bulk VM Actions
	adminOp.POST("/:id/vms/bulk", h.Node.BulkVMAction, middleware.RequirePermission(model.PermissionVMPower))

	// Tag Sync from Proxmox
	adminOp.POST("/:id/tags/sync", h.Node.SyncTags, middleware.RequirePermission(model.PermissionNodesWrite))

	// VM Cockpit
	if h.VMCockpit != nil {
		vmCockpit := nodes.Group("/:id/vms/:vmid/cockpit")
		vmCockpit.Use(middleware.RequirePermission(model.PermissionVMsWrite))
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

	// VM Permissions
	if h.VMPermission != nil {
		vmPerms := protected.Group("/vm-permissions")
		vmPerms.Use(middleware.RequirePermission(model.PermissionSettingsManage))
		vmPerms.GET("", h.VMPermission.List)
		vmPerms.POST("", h.VMPermission.Create)
		vmPerms.PUT("/upsert", h.VMPermission.Upsert)
		vmPerms.PUT("/:id", h.VMPermission.Update)
		vmPerms.DELETE("/:id", h.VMPermission.Delete)
		vmPerms.GET("/effective", h.VMPermission.GetEffective)
		vmPerms.GET("/all", h.VMPermission.ListAllPermissions)
	}

	// VM Groups
	if h.VMGroup != nil {
		vmGroups := protected.Group("/vm-groups")
		vmGroups.Use(middleware.RequirePermission(model.PermissionSettingsManage))
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
		nodes.GET("/:id/vms/:vmid/health", h.VMHealth.GetVMHealth, middleware.RequirePermission(model.PermissionVMsRead))
		nodes.GET("/:id/vm-health", h.VMHealth.GetAllVMHealth, middleware.RequirePermission(model.PermissionVMsRead))

		// VM-level rightsizing
		nodes.GET("/:id/vms/:vmid/rightsizing", h.VMHealth.GetVMRightsizing, middleware.RequirePermission(model.PermissionVMsRead))

		// VM-level anomalies
		nodes.GET("/:id/vms/:vmid/anomalies", h.VMHealth.GetVMAnomalies, middleware.RequirePermission(model.PermissionSecurityRead))

		// Snapshot policies
		nodes.GET("/:id/vms/:vmid/snapshot-policies", h.VMHealth.ListSnapshotPolicies, middleware.RequirePermission(model.PermissionVMsRead))
		adminOp.POST("/:id/vms/:vmid/snapshot-policies", h.VMHealth.CreateSnapshotPolicy, middleware.RequirePermission(model.PermissionVMsWrite))
		adminOp.PUT("/:id/vms/:vmid/snapshot-policies/:policyId", h.VMHealth.UpdateSnapshotPolicy, middleware.RequirePermission(model.PermissionVMsWrite))
		adminOp.DELETE("/:id/vms/:vmid/snapshot-policies/:policyId", h.VMHealth.DeleteSnapshotPolicy, middleware.RequirePermission(model.PermissionVMsWrite))

		// Scheduled actions
		nodes.GET("/:id/vms/:vmid/scheduled-actions", h.VMHealth.ListScheduledActions, middleware.RequirePermission(model.PermissionVMsRead))
		adminOp.POST("/:id/vms/:vmid/scheduled-actions", h.VMHealth.CreateScheduledAction, middleware.RequirePermission(model.PermissionVMsWrite))
		adminOp.DELETE("/:id/vms/:vmid/scheduled-actions/:actionId", h.VMHealth.DeleteScheduledAction, middleware.RequirePermission(model.PermissionVMsWrite))

		// VM Dependencies (non-node-scoped)
		vmDeps := protected.Group("/vm-dependencies")
		vmDeps.GET("", h.VMHealth.ListAllDependencies, middleware.RequirePermission(model.PermissionVMsRead))
		vmDepsAdmin := vmDeps.Group("")
		vmDepsAdmin.Use(middleware.RequirePermission(model.PermissionVMsWrite))
		vmDepsAdmin.POST("", h.VMHealth.CreateDependency)
		vmDepsAdmin.DELETE("/:depId", h.VMHealth.DeleteDependency)

		// VM Dependencies (node-scoped)
		nodes.GET("/:id/vms/:vmid/dependencies", h.VMHealth.ListVMDependencies, middleware.RequirePermission(model.PermissionVMsRead))
	}

	// Log WebSocket (top-level, auth handled internally)
	if h.LogStream != nil {
		e.GET("/api/v1/ws/logs", h.LogStream.HandleWS)
	}

	// Log routes
	if h.LogAnalysis != nil {
		logs := protected.Group("/logs")
		logs.POST("/analyze", h.LogAnalysis.Analyze, middleware.RequirePermission(model.PermissionLogsManage))
		logs.GET("/analyses", h.LogAnalysis.ListAnalyses, middleware.RequirePermission(model.PermissionLogsRead))

		if h.LogExport != nil {
			logs.GET("/export", h.LogExport.Export, middleware.RequirePermission(model.PermissionLogsRead))
		}

		// Report schedules (operator+)
		if h.LogReportSchedule != nil {
			schedOp := logs.Group("/report-schedules")
			schedOp.Use(middleware.RequirePermission(model.PermissionLogsManage))
			schedOp.POST("", h.LogReportSchedule.Create)
			schedOp.GET("", h.LogReportSchedule.List)
			schedOp.PUT("/:id", h.LogReportSchedule.Update)
			schedOp.DELETE("/:id", h.LogReportSchedule.Delete)
		}

		// Node-scoped log routes
		nodes.GET("/:id/log-anomalies", h.LogAnalysis.ListAnomalies, middleware.RequirePermission(model.PermissionLogsRead))
		nodes.GET("/:id/log-bookmarks", h.LogBookmark.ListByNode, middleware.RequirePermission(model.PermissionLogsRead))
		nodes.GET("/:id/log-sources", h.LogSource.ListByNode, middleware.RequirePermission(model.PermissionLogsRead))

		// Anomaly actions (operator+)
		logAnomalies := protected.Group("/log-anomalies")
		logAnomalies.GET("/:id", h.LogAnalysis.GetAnomaly, middleware.RequirePermission(model.PermissionLogsRead))
		logAnomaliesOp := logAnomalies.Group("")
		logAnomaliesOp.Use(middleware.RequirePermission(model.PermissionLogsManage))
		logAnomaliesOp.POST("/:id/acknowledge", h.LogAnalysis.Acknowledge)

		// Bookmarks
		bookmarks := protected.Group("/log-bookmarks")
		bookmarks.POST("", h.LogBookmark.Create, middleware.RequirePermission(model.PermissionLogsManage))
		bookmarks.DELETE("/:id", h.LogBookmark.Delete, middleware.RequirePermission(model.PermissionLogsManage))

		// Source management (operator+)
		logSourcesOp := nodes.Group("/:id/log-sources")
		logSourcesOp.Use(middleware.RequirePermission(model.PermissionLogsManage))
		logSourcesOp.PUT("", h.LogSource.Update)
	}

	// Network routes
	if h.NetworkScan != nil {
		nodes.GET("/:id/network-scans", h.NetworkScan.ListByNode, middleware.RequirePermission(model.PermissionSecurityRead))
		nodes.GET("/:id/network-devices", h.NetworkDevice.ListByNode, middleware.RequirePermission(model.PermissionSecurityRead))
		nodes.GET("/:id/network-anomalies", h.NetworkAnomaly.ListByNode, middleware.RequirePermission(model.PermissionSecurityRead))
		nodes.GET("/:id/scan-baselines", h.ScanBaseline.ListByNode, middleware.RequirePermission(model.PermissionSecurityRead))

		netScans := protected.Group("/network-scans")
		netScans.GET("/:id", h.NetworkScan.Get, middleware.RequirePermission(model.PermissionSecurityRead))
		netScans.GET("/:id1/diff/:id2", h.NetworkScan.Diff, middleware.RequirePermission(model.PermissionSecurityRead))

		// Trigger scan (operator+)
		netScansOp := nodes.Group("/:id/network-scans")
		netScansOp.Use(middleware.RequirePermission(model.PermissionSecurityManage))
		netScansOp.POST("", h.NetworkScan.Trigger)

		netDevices := protected.Group("/network-devices")
		netDevices.PUT("/:id", h.NetworkDevice.Update, middleware.RequirePermission(model.PermissionSecurityManage))

		netAnomaliesOp := protected.Group("/network-anomalies")
		netAnomaliesOp.Use(middleware.RequirePermission(model.PermissionSecurityManage))
		netAnomaliesOp.POST("/:id/acknowledge", h.NetworkAnomaly.Acknowledge)

		baselinesOp := nodes.Group("/:id/scan-baselines")
		baselinesOp.Use(middleware.RequirePermission(model.PermissionSecurityManage))
		baselinesOp.POST("", h.ScanBaseline.Create)

		baselinesMgmtOp := protected.Group("/scan-baselines")
		baselinesMgmtOp.Use(middleware.RequirePermission(model.PermissionSecurityManage))
		baselinesMgmtOp.PUT("/:id", h.ScanBaseline.Update)
		baselinesMgmtOp.DELETE("/:id", h.ScanBaseline.Delete)
		baselinesMgmtOp.POST("/:id/activate", h.ScanBaseline.Activate)
	}

	// WebSocket
	e.GET("/api/v1/ws", h.WS.HandleWS)
}
