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
	Anomaly      *handler.AnomalyHandler
	Prediction   *handler.PredictionHandler
	Briefing     *handler.BriefingHandler
	Approval     *handler.ApprovalHandler
	Drift        *handler.DriftHandler
	Environment  *handler.EnvironmentHandler
	Update       *handler.UpdateHandler
	Rightsizing  *handler.RightsizingHandler
	SSHKey       *handler.SSHKeyHandler
	Gateway      *handler.GatewayHandler
	Topology     *handler.TopologyHandler
	Brain        *handler.BrainHandler
	Reflex       *handler.ReflexHandler
	AgentConfig  *handler.AgentConfigHandler
}

func SetupRouter(e *echo.Echo, cfg *config.Config, jwtSvc *auth.JWTService, h Handlers, gatewaySvc *gateway.Service, redisClient *redis.Client, auditRepo repository.AuditRepository) {
	// Global middleware
	e.Use(middleware.Recovery())
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLogger())
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

	// Auth - protected
	protected.GET("/auth/me", h.Auth.Me)

	// Nodes
	nodes := protected.Group("/nodes")
	nodes.GET("", h.Node.List)
	nodes.GET("/:id", h.Node.Get)
	nodes.GET("/:id/status", h.Node.GetStatus)
	nodes.GET("/:id/vms", h.Node.GetVMs)
	nodes.GET("/:id/storage", h.Node.GetStorage)
	nodes.GET("/:id/network", h.Node.GetNetworkInterfaces)
	nodes.GET("/:id/disks", h.Node.GetDisks)
	nodes.GET("/:id/tags", h.Tag.GetNodeTags)
	nodes.GET("/:id/isos", h.Node.ListISOs)
	nodes.GET("/:id/templates", h.Node.ListTemplates)

	// Node backups (read)
	nodes.GET("/:id/backups", h.Backup.ListBackups)
	nodes.GET("/:id/backup-schedules", h.Schedule.ListSchedules)

	// Node metrics
	nodes.GET("/:id/metrics", h.Metrics.GetMetricsHistory)
	nodes.GET("/:id/metrics/summary", h.Metrics.GetMetricsSummary)

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

	// Admin only
	admin := nodes.Group("")
	admin.Use(middleware.RequireRole(model.RoleAdmin))
	admin.DELETE("/:id", h.Node.Delete)

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

	tagsAdmin := tags.Group("")
	tagsAdmin.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	tagsAdmin.POST("", h.Tag.CreateTag)

	tagsAdminOnly := tags.Group("")
	tagsAdminOnly.Use(middleware.RequireRole(model.RoleAdmin))
	tagsAdminOnly.DELETE("/:id", h.Tag.DeleteTag)

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
	}

	// Agent Approvals
	if h.Approval != nil {
		approvals := protected.Group("/approvals")
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
		sshAdmin.POST("/:id/ssh-keys/rotate", h.SSHKey.Rotate)
		sshAdmin.DELETE("/:id/ssh-keys/:keyId", h.SSHKey.Delete)
		sshAdmin.POST("/:id/ssh-keys/rotation", h.SSHKey.CreateRotationSchedule)
	}

	// API Gateway Management
	if h.Gateway != nil {
		gwRoutes := protected.Group("/gateway")
		gwRoutes.GET("/tokens", h.Gateway.ListTokens)
		gwRoutes.POST("/tokens", h.Gateway.CreateToken)
		gwRoutes.POST("/tokens/:id/revoke", h.Gateway.RevokeToken)
		gwRoutes.DELETE("/tokens/:id", h.Gateway.DeleteToken)

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

	// Backup Recovery Guide
	backups.GET("/:id/recovery-guide", h.Backup.GetRecoveryGuide)

	// Bulk VM Actions
	adminOp.POST("/:id/vms/bulk", h.Node.BulkVMAction)

	// Tag Sync from Proxmox
	adminOp.POST("/:id/tags/sync", h.Node.SyncTags)

	// WebSocket
	e.GET("/api/v1/ws", h.WS.HandleWS)
}
