package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/antigravity/prometheus/internal/api"
	"github.com/antigravity/prometheus/internal/api/handler"
	"github.com/antigravity/prometheus/internal/api/middleware"
	"github.com/antigravity/prometheus/internal/config"
	"github.com/antigravity/prometheus/internal/llm"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/scheduler"
	"github.com/antigravity/prometheus/internal/service/agent"
	"github.com/antigravity/prometheus/internal/service/anomaly"
	"github.com/antigravity/prometheus/internal/service/auth"
	"github.com/antigravity/prometheus/internal/service/backup"
	"github.com/antigravity/prometheus/internal/service/brain"
	"github.com/antigravity/prometheus/internal/service/briefing"
	"github.com/antigravity/prometheus/internal/service/crypto"
	"github.com/antigravity/prometheus/internal/service/drift"
	"github.com/antigravity/prometheus/internal/service/environment"
	"github.com/antigravity/prometheus/internal/service/gateway"
	"github.com/antigravity/prometheus/internal/service/intelligence"
	"github.com/antigravity/prometheus/internal/service/loganalyzer"
	"github.com/antigravity/prometheus/internal/service/logscan"
	"github.com/antigravity/prometheus/internal/service/logstream"
	migrationService "github.com/antigravity/prometheus/internal/service/migration"
	"github.com/antigravity/prometheus/internal/service/monitor"
	"github.com/antigravity/prometheus/internal/service/netscan"
	nodeService "github.com/antigravity/prometheus/internal/service/node"
	"github.com/antigravity/prometheus/internal/service/notification"
	operationsService "github.com/antigravity/prometheus/internal/service/operations"
	"github.com/antigravity/prometheus/internal/service/prediction"
	"github.com/antigravity/prometheus/internal/service/recovery"
	"github.com/antigravity/prometheus/internal/service/reflex"
	"github.com/antigravity/prometheus/internal/service/rightsizing"
	"github.com/antigravity/prometheus/internal/service/sshkeys"
	telegramService "github.com/antigravity/prometheus/internal/service/telegram"
	topologyService "github.com/antigravity/prometheus/internal/service/topology"
	"github.com/antigravity/prometheus/internal/service/updates"
	userService "github.com/antigravity/prometheus/internal/service/user"
	vmService "github.com/antigravity/prometheus/internal/service/vm"
	"github.com/antigravity/prometheus/internal/ssh"
	"github.com/labstack/echo/v4"
)

func main() {
	// Bootstrap logger with a sensible default so that any error during
	// config loading is still captured before we switch to the configured
	// level/format.
	bootstrapLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(bootstrapLogger)

	slog.Info("starting Prometheus server")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	// Re-init the logger with values from config (LOG_LEVEL / LOG_FORMAT).
	logHandlerOpts := &slog.HandlerOptions{Level: cfg.Log.SlogLevel()}
	var logHandler slog.Handler
	if cfg.Log.Format == "text" {
		logHandler = slog.NewTextHandler(os.Stdout, logHandlerOpts)
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, logHandlerOpts)
	}
	slog.SetDefault(slog.New(logHandler))
	slog.Info("logger configured",
		slog.String("level", cfg.Log.Level),
		slog.String("format", cfg.Log.Format),
	)

	// Context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Database
	dbPool, err := repository.NewPostgresPool(ctx, cfg.Database.DSN(), cfg.Database.MaxConns)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", slog.Any("error", err))
		os.Exit(1)
	}
	defer dbPool.Close()

	// Redis
	redisClient, err := repository.NewRedisClient(ctx, cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		slog.Error("failed to connect to Redis", slog.Any("error", err))
		os.Exit(1)
	}
	defer redisClient.Close()

	// Run migrations
	migrator := repository.NewMigrator(dbPool, "migrations")
	if err := migrator.Run(ctx); err != nil {
		slog.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	// Encryption
	encryptor, err := crypto.NewEncryptor(cfg.Encryption.Key)
	if err != nil {
		slog.Error("failed to create encryptor", slog.Any("error", err))
		os.Exit(1)
	}

	// Repositories
	userRepo := repository.NewUserRepository(dbPool)
	nodeRepo := repository.NewNodeRepository(dbPool)
	tokenRepo := repository.NewTokenRepository(dbPool)
	backupRepo := repository.NewBackupRepository(dbPool)
	backupFileRepo := repository.NewBackupFileRepository(dbPool)
	scheduleRepo := repository.NewScheduleRepository(dbPool)

	// Sweep backups left in PENDING/RUNNING after a crash. Anything older
	// than 1h with no completion is by definition dead — the process that
	// owned it is gone. Marking them FAILED stops them showing as
	// "in progress" in the UI forever and frees the version slot.
	if cleaned, err := backupRepo.MarkStuckRunningAsFailed(ctx, 3600,
		"Server wurde während des Backups beendet — Eintrag automatisch als fehlgeschlagen markiert"); err != nil {
		slog.Warn("startup backup-sweep failed", slog.Any("error", err))
	} else if cleaned > 0 {
		slog.Info("startup backup-sweep marked stuck backups as failed", slog.Int("count", cleaned))
	}
	metricsRepo := repository.NewMetricsRepository(dbPool)
	aliasRepo := repository.NewNetworkAliasRepository(dbPool)
	tagRepo := repository.NewTagRepository(dbPool)

	// Notification Repositories
	channelRepo := repository.NewNotificationChannelRepository(dbPool)
	historyRepo := repository.NewNotificationHistoryRepository(dbPool)
	ruleRepo := repository.NewAlertRuleRepository(dbPool)

	// Escalation & Incident Repositories
	escalationPolicyRepo := repository.NewEscalationPolicyRepository(dbPool)
	alertIncidentRepo := repository.NewAlertIncidentRepository(dbPool)

	// Telegram Repositories
	telegramLinkRepo := repository.NewTelegramLinkRepository(dbPool)
	telegramConvRepo := repository.NewTelegramConversationRepository(dbPool)

	// Migration Repositories
	migrationRepo := repository.NewMigrationRepository(dbPool)
	migrationLogRepo := repository.NewMigrationLogRepository(dbPool)

	// DR Repositories
	profileRepo := repository.NewNodeProfileRepository(dbPool)
	readinessRepo := repository.NewDRReadinessRepository(dbPool)
	runbookRepo := repository.NewRunbookRepository(dbPool)

	// Phase 4 Repositories
	approvalRepo := repository.NewApprovalRepository(dbPool)
	anomalyRepo := repository.NewAnomalyRepository(dbPool)
	predictionRepo := repository.NewPredictionRepository(dbPool)
	briefingRepo := repository.NewBriefingRepository(dbPool)

	// Phase 6 Repositories
	driftRepo := repository.NewDriftRepository(dbPool)
	envRepo := repository.NewEnvironmentRepository(dbPool)
	updateRepo := repository.NewUpdateRepository(dbPool)
	recRepo := repository.NewRecommendationRepository(dbPool)
	sshKeyRepo := repository.NewSSHKeyRepository(dbPool)
	apiTokenRepo := repository.NewAPITokenRepository(dbPool)
	auditRepo := repository.NewAuditRepository(dbPool)
	policyRepo := repository.NewPasswordPolicyRepository(dbPool)
	rolePermissionRepo := repository.NewRolePermissionRepository(dbPool)
	userInvitationRepo := repository.NewUserInvitationRepository(dbPool)

	// Phase 8 Repositories
	securityEventRepo := repository.NewSecurityEventRepository(dbPool)

	// VM Permission & Group Repositories
	vmPermRepo := repository.NewVMPermissionRepository(dbPool)
	vmGroupRepo := repository.NewVMGroupRepository(dbPool)

	// Log & Network Repositories
	logAnomalyRepo := repository.NewLogAnomalyRepository(dbPool)
	logAnalysisRepo := repository.NewLogAnalysisRepository(dbPool)
	logBookmarkRepo := repository.NewLogBookmarkRepository(dbPool)
	logSourceRepo := repository.NewLogSourceRepository(dbPool)
	logReportScheduleRepo := repository.NewLogReportScheduleRepository(dbPool)
	networkScanRepo := repository.NewNetworkScanRepository(dbPool)
	networkDeviceRepo := repository.NewNetworkDeviceRepository(dbPool)
	networkPortRepo := repository.NewNetworkPortRepository(dbPool)
	networkAnomalyRepo := repository.NewNetworkAnomalyRepository(dbPool)
	scanBaselineRepo := repository.NewScanBaselineRepository(dbPool)

	// Phase 7 Repositories
	brainRepo := repository.NewBrainRepository(dbPool)
	reflexRepo := repository.NewReflexRuleRepository(dbPool)
	agentConfigRepo := repository.NewAgentConfigRepository(dbPool)

	// Services
	jwtSvc := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	authService := auth.NewService(userRepo, tokenRepo, jwtSvc, redisClient)

	// Proxmox TLS configuration
	proxmoxTLS := buildProxmoxTLSConfig(cfg.Proxmox)
	clientFactory := proxmox.NewClientFactory(encryptor, proxmoxTLS)

	// SSH Pool
	sshPool := ssh.NewPool(ssh.PoolConfig{})

	nodeSvc := nodeService.NewService(nodeRepo, encryptor, clientFactory, aliasRepo, tagRepo, sshPool)
	monitorSvc := monitor.NewService(nodeRepo, redisClient, metricsRepo)

	// Seed admin user
	seedAdmin(ctx, authService)

	// WebSocket hub
	wsHub := monitor.NewWSHub()
	go wsHub.Run()

	// Backup Services
	backupSvc := backup.NewService(backupRepo, backupFileRepo, nodeRepo, encryptor, sshPool, wsHub)
	restoreSvc := backup.NewRestoreService(backupRepo, backupFileRepo, nodeRepo, encryptor, sshPool)

	// User Service
	pwValidator := userService.NewPasswordValidator(policyRepo)
	userSvc := userService.NewService(userRepo, pwValidator).WithAccessRepositories(tokenRepo, apiTokenRepo, userInvitationRepo)
	// Wire user-deactivation events through to the JWT-middleware cache so
	// admin actions take effect for in-flight tokens immediately.
	userService.SetActiveUserCacheInvalidator(middleware.InvalidateActiveUserCache)

	// Notification Service
	notifSvc := notification.NewService(channelRepo, historyRepo, encryptor)
	alertSvc := notification.NewAlertService(ruleRepo, metricsRepo, nodeRepo, notifSvc, wsHub)

	// Escalation Service
	escalationSvc := notification.NewEscalationService(escalationPolicyRepo, alertIncidentRepo, ruleRepo, notifSvc)
	alertSvc.SetEscalationService(escalationSvc)

	// DR Services
	profileSvc := recovery.NewProfileService(profileRepo, nodeRepo, encryptor, sshPool)
	readinessSvc := recovery.NewReadinessService(readinessRepo, profileRepo, backupRepo, nodeRepo)
	runbookSvc := recovery.NewRunbookService(runbookRepo, profileRepo, nodeRepo)

	// Migration Service
	migrationSvc := migrationService.NewService(migrationRepo, nodeRepo, encryptor, sshPool, clientFactory, wsHub, migrationLogRepo)

	// Recover migrations that were interrupted by a server restart
	if err := migrationSvc.RecoverOrphanedMigrations(ctx); err != nil {
		slog.Error("failed to recover orphaned migrations", slog.Any("error", err))
	}

	// LLM Registry
	llmRegistry := llm.NewRegistry()
	// Always create ollamaProvider — use configured URL or default to localhost
	ollamaURL := cfg.LLM.OllamaURL
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollamaProvider := llm.NewOllamaProvider(ollamaURL)
	llmRegistry.Register("ollama", ollamaProvider)
	if cfg.LLM.OpenAIKey != "" {
		llmRegistry.Register("openai", llm.NewOpenAIProvider(cfg.LLM.OpenAIKey, cfg.LLM.OpenAIURL))
	}
	if cfg.LLM.AnthropicKey != "" {
		llmRegistry.Register("anthropic", llm.NewAnthropicProvider(cfg.LLM.AnthropicKey))
	}
	if cfg.LLM.DefaultModel != "" {
		llmRegistry.SetDefault(cfg.LLM.DefaultModel)
	}

	// Phase 4 Services
	anomalySvc := anomaly.NewService(anomalyRepo, metricsRepo, nodeRepo)
	predictionSvc := prediction.NewService(predictionRepo, metricsRepo, nodeRepo)
	briefingSvc := briefing.NewService(briefingRepo, nodeRepo, metricsRepo, anomalyRepo, predictionRepo, llmRegistry)
	briefingSvc.SetNodeService(nodeSvc)
	anomalySvc.SetNodeService(nodeSvc)
	predictionSvc.SetNodeService(nodeSvc)

	// Phase 6 Services
	driftSvc := drift.NewService(driftRepo, backupRepo, backupFileRepo, nodeRepo, encryptor, sshPool, llmRegistry, backupSvc)
	envSvc := environment.NewService(envRepo, nodeRepo)
	updateSvc := updates.NewService(updateRepo, nodeRepo, encryptor, sshPool)
	rightsizingSvc := rightsizing.NewService(recRepo, nodeRepo, clientFactory)
	sshkeySvc := sshkeys.NewService(sshKeyRepo, nodeRepo, encryptor, sshPool)
	gatewaySvc := gateway.NewService(apiTokenRepo, userRepo)
	topologySvc := topologyService.NewService(nodeRepo, clientFactory)

	// Phase 8 Services
	analysisSvc := intelligence.NewAnalysisService(securityEventRepo, metricsRepo, nodeRepo, anomalyRepo, predictionRepo, llmRegistry, wsHub)
	analysisSvc.SetNodeService(nodeSvc)

	// Phase 7 Services
	brainSvc := brain.NewService(brainRepo)
	reflexSvc := reflex.NewService(reflexRepo, metricsRepo, nodeRepo, nodeSvc, notifSvc)

	// Log Streaming
	logStreamMgr := logstream.NewStreamManager(sshPool, redisClient, nodeRepo, logSourceRepo, logstream.StreamConfig{
		WorkerPoolSize:   20,
		RotationInterval: 30 * time.Second,
		RedisMaxLen:      10000,
		RedisMaxAge:      30 * time.Minute,
	})

	// Log Discovery
	logDiscoverySvc := logscan.NewDiscoveryService(logSourceRepo, sshPool, nodeRepo)

	// Log Analyzer
	logClassifier := loganalyzer.NewClassifier(llmRegistry, "", 3)
	logConsumer := loganalyzer.NewConsumer(redisClient, logClassifier, logAnomalyRepo, wsHub, loganalyzer.ConsumerConfig{
		BatchSize:        10,
		BatchTimeout:     2 * time.Second,
		AnomalyThreshold: 0.5,
		AlertThreshold:   0.8,
		DedupWindow:      5 * time.Minute,
	})
	logReporter := loganalyzer.NewReporter(redisClient, llmRegistry, logAnalysisRepo)

	// Network Scanning
	netScanner := netscan.NewScanScheduler(sshPool, nodeRepo, networkScanRepo, networkDeviceRepo, networkPortRepo, networkAnomalyRepo, scanBaselineRepo, wsHub, netscan.ScanConfig{
		QuickInterval: 5 * time.Minute,
		FullInterval:  60 * time.Minute,
		MaxParallel:   5,
		TopPorts:      1000,
	})

	// Start log streaming and consumer
	go logStreamMgr.Start(ctx)
	go logConsumer.Start(ctx)

	// VM Permission & Group Services
	vmPermSvc := vmService.NewPermissionService(vmPermRepo, userRepo)
	vmGroupSvc := vmService.NewGroupService(vmGroupRepo)

	// Phase 4 VM Cockpit Services
	snapshotPolicyRepo := repository.NewSnapshotPolicyRepository(dbPool)
	scheduledActionRepo := repository.NewScheduledActionRepository(dbPool)
	vmDependencyRepo := repository.NewVMDependencyRepository(dbPool)

	vmHealthSvc := vmService.NewHealthService(nodeRepo, metricsRepo, clientFactory)
	vmRightsizingSvc := vmService.NewRightsizingService(nodeRepo, clientFactory)
	vmAnomalySvc := vmService.NewAnomalyService(nodeRepo, metricsRepo, clientFactory)
	snapshotPolicySvc := vmService.NewSnapshotPolicyService(snapshotPolicyRepo, nodeRepo, clientFactory)
	scheduledActionSvc := vmService.NewScheduledActionService(scheduledActionRepo, nodeSvc)
	vmDependencySvc := vmService.NewDependencyService(vmDependencyRepo, nodeRepo, clientFactory)
	operationsSvc := operationsService.NewService(
		nodeRepo,
		backupRepo,
		migrationRepo,
		auditRepo,
		securityEventRepo,
		anomalyRepo,
		predictionRepo,
		alertIncidentRepo,
		approvalRepo,
		historyRepo,
		scheduleRepo,
		logReportScheduleRepo,
		snapshotPolicyRepo,
		vmDependencyRepo,
		scheduledActionRepo,
		networkDeviceRepo,
		networkPortRepo,
		nodeSvc,
		llmRegistry,
	)

	// Chat Repositories
	chatConvRepo := repository.NewChatConversationRepository(dbPool)
	chatMsgRepo := repository.NewChatMessageRepository(dbPool)
	toolCallRepo := repository.NewToolCallRepository(dbPool)

	// Agent Tool Registry
	agentToolRegistry := agent.NewToolRegistry()
	agentToolRegistry.Register(agent.NewListNodesTool(nodeSvc))
	agentToolRegistry.Register(agent.NewNodeStatusTool(nodeSvc))
	agentToolRegistry.Register(agent.NewGetVMsTool(nodeSvc))
	agentToolRegistry.Register(agent.NewCreateBackupTool(backupSvc))
	agentToolRegistry.Register(agent.NewGetMetricsTool(metricsRepo))
	agentToolRegistry.Register(agent.NewGetStorageTool(nodeSvc))
	agentToolRegistry.Register(agent.NewMigrateVMTool(migrationSvc))
	agentToolRegistry.Register(agent.NewStartVMTool(nodeSvc))
	agentToolRegistry.Register(agent.NewStopVMTool(nodeSvc))
	agentToolRegistry.Register(agent.NewRestoreConfigTool(restoreSvc))
	agentToolRegistry.Register(agent.NewRunSSHCommandTool(nodeSvc))
	agentToolRegistry.Register(agent.NewGetNetworkTool(nodeSvc))
	agentToolRegistry.Register(agent.NewGetAnomaliesTool(anomalySvc))
	agentToolRegistry.Register(agent.NewGetPredictionsTool(predictionSvc))
	agentToolRegistry.Register(agent.NewGetBriefingTool(briefingSvc))
	agentToolRegistry.Register(agent.NewCheckDriftTool(driftSvc))
	agentToolRegistry.Register(agent.NewCheckUpdatesTool(updateSvc))
	agentToolRegistry.Register(agent.NewRightsizingTool(rightsizingSvc))
	agentToolRegistry.Register(agent.NewSaveKnowledgeTool(brainSvc))
	agentToolRegistry.Register(agent.NewRecallKnowledgeTool(brainSvc))

	// VM Cockpit AI Tools (Phase 2)
	agentToolRegistry.Register(agent.NewVMExecTool(nodeSvc))
	agentToolRegistry.Register(agent.NewVMFileReadTool(nodeSvc))
	agentToolRegistry.Register(agent.NewVMFileWriteTool(nodeSvc))
	agentToolRegistry.Register(agent.NewVMProcessesTool(nodeSvc))
	agentToolRegistry.Register(agent.NewVMServicesTool(nodeSvc))
	agentToolRegistry.Register(agent.NewVMServiceActionTool(nodeSvc))
	agentToolRegistry.Register(agent.NewVMDiskUsageTool(nodeSvc))
	agentToolRegistry.Register(agent.NewVMNetworkInfoTool(nodeSvc))

	// Agent Service
	agentSvc := agent.NewService(llmRegistry, agentToolRegistry, chatConvRepo, chatMsgRepo, toolCallRepo, approvalRepo, userRepo, rolePermissionRepo, agentConfigRepo)

	// Telegram Bot Service (conditional)
	var telegramBotSvc *telegramService.BotService
	telegramBotEnabled := cfg.Telegram.Enabled && cfg.Telegram.BotToken != ""
	if telegramBotEnabled {
		telegramBotSvc = telegramService.NewBotService(
			cfg.Telegram.BotToken,
			agentSvc,
			telegramLinkRepo,
			telegramConvRepo,
			agentConfigRepo,
			approvalRepo,
		)
		slog.Info("telegram bot enabled")
	}

	// Proactive push pipeline — wires the Telegram bot (and any future
	// transports) into the security/briefing services so the agent can
	// notify linked admins on its own initiative. nil-safe: if no notifier
	// is wired up, the push calls become no-ops.
	var pushSvc *agent.PushService
	if telegramBotSvc != nil {
		pushSvc = agent.NewPushService(telegramBotSvc)
		analysisSvc.SetPushNotifier(intelligencePushAdapter{pushSvc})
		briefingSvc.SetPushNotifier(briefingPushAdapter{pushSvc})

		// Interactive approval flow: when the agent creates a pending
		// approval, the bot sends an inline-keyboard prompt to the
		// requesting user. Tap → resolves the approval directly.
		agentSvc.SetApprovalNotifier(telegramBotSvc)
		slog.Info("agent push pipeline enabled")
	}

	// Scheduler
	sched := scheduler.New()
	nodeStatusJob := scheduler.NewNodeStatusJob(nodeRepo, clientFactory, redisClient, wsHub, 30*time.Second)
	sched.AddJob(nodeStatusJob)
	metricsJob := scheduler.NewMetricsCollectionJob(nodeRepo, metricsRepo, clientFactory, wsHub, 60*time.Second)
	sched.AddJob(metricsJob)
	backupScheduleJob := scheduler.NewBackupScheduleJob(scheduleRepo, backupRepo, backupSvc, 60*time.Second)
	sched.AddJob(backupScheduleJob)
	alertEvalJob := scheduler.NewAlertEvaluationJob(alertSvc, 30*time.Second)
	sched.AddJob(alertEvalJob)
	drProfileJob := scheduler.NewDRProfileJob(nodeRepo, profileSvc, readinessSvc, 24*time.Hour)
	sched.AddJob(drProfileJob)
	escalationJob := scheduler.NewEscalationJob(escalationSvc, 30*time.Second)
	sched.AddJob(escalationJob)
	anomalyDetectionJob := scheduler.NewAnomalyDetectionJob(anomalySvc, 5*time.Minute)
	sched.AddJob(anomalyDetectionJob)
	predictionJob := scheduler.NewPredictionJob(predictionSvc, 6*time.Hour)
	sched.AddJob(predictionJob)
	if cfg.Briefing.Enabled {
		briefingJob := scheduler.NewBriefingJob(briefingSvc, cfg.Briefing.Hour)
		sched.AddJob(briefingJob)
	}
	// Phase 6 Scheduler Jobs
	driftJob := scheduler.NewDriftCheckJob(driftSvc, nodeRepo, 6*time.Hour)
	sched.AddJob(driftJob)
	updateCheckJob := scheduler.NewUpdateCheckJob(updateSvc, nodeRepo, 24*time.Hour)
	sched.AddJob(updateCheckJob)
	rightsizingJob := scheduler.NewRightsizingJob(rightsizingSvc, nodeRepo, 24*time.Hour)
	sched.AddJob(rightsizingJob)
	// VM snapshot policies and scheduled VM actions are cron-based; poll every
	// 60s (cron granularity is minutes) and run the ones that are due.
	snapshotPolicyJob := scheduler.NewSnapshotPolicyJob(snapshotPolicyRepo, snapshotPolicySvc, 60*time.Second)
	sched.AddJob(snapshotPolicyJob)
	scheduledActionJob := scheduler.NewScheduledActionJob(scheduledActionRepo, scheduledActionSvc, 60*time.Second)
	sched.AddJob(scheduledActionJob)
	keyRotationJob := scheduler.NewKeyRotationJob(sshkeySvc, 1*time.Hour)
	sched.AddJob(keyRotationJob)
	reflexEvalJob := scheduler.NewReflexEvaluationJob(reflexSvc, 30*time.Second)
	sched.AddJob(reflexEvalJob)
	intelligenceJob := scheduler.NewIntelligenceJob(analysisSvc, 10*time.Minute)
	sched.AddJob(intelligenceJob)
	logDiscoveryJob := scheduler.NewLogDiscoveryJob(logDiscoverySvc, nodeRepo, 5*time.Minute)
	sched.AddJob(logDiscoveryJob)
	netQuickScanJob := scheduler.NewNetQuickScanJob(netScanner, 5*time.Minute)
	sched.AddJob(netQuickScanJob)
	netFullScanJob := scheduler.NewNetFullScanJob(netScanner, 60*time.Minute)
	sched.AddJob(netFullScanJob)
	logRetentionJob := scheduler.NewLogRetentionJob(logAnomalyRepo, logAnalysisRepo, networkScanRepo, networkAnomalyRepo, 24*time.Hour)
	sched.AddJob(logRetentionJob)
	logReportJob := scheduler.NewLogReportScheduleJob(logReportScheduleRepo, logReporter, notifSvc, 60*time.Second)
	sched.AddJob(logReportJob)

	if telegramBotEnabled && telegramBotSvc != nil {
		pollInterval := time.Duration(cfg.Telegram.PollInterval) * time.Second
		if pollInterval < time.Second {
			pollInterval = 3 * time.Second
		}
		telegramPollJob := scheduler.NewTelegramPollJob(telegramBotSvc, pollInterval)
		sched.AddJob(telegramPollJob)
	}
	sched.Start(ctx)
	defer sched.Stop()

	// Integrate notifications into existing services
	backupSvc.SetNotificationService(notifSvc)
	nodeStatusJob.SetNotificationService(notifSvc)

	// Echo setup
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Handlers
	handlers := api.Handlers{
		Health:         handler.NewHealthHandler(dbPool, redisClient),
		Auth:           handler.NewAuthHandler(authService, userRepo, redisClient),
		Node:           handler.NewNodeHandler(nodeSvc),
		WS:             handler.NewWSHandler(wsHub, jwtSvc, cfg.CORS.AllowOrigins),
		Backup:         handler.NewBackupHandler(backupSvc, restoreSvc, nodeSvc),
		Schedule:       handler.NewScheduleHandler(scheduleRepo),
		Metrics:        handler.NewMetricsHandler(monitorSvc, nodeSvc),
		Tag:            handler.NewTagHandler(tagRepo),
		PBS:            handler.NewPBSHandler(nodeRepo, encryptor),
		User:           handler.NewUserHandler(userSvc),
		Notification:   handler.NewNotificationHandler(notifSvc, alertSvc),
		DR:             handler.NewDRHandler(profileSvc, readinessSvc, runbookSvc),
		Chat:           handler.NewChatHandler(agentSvc),
		Migration:      handler.NewMigrationHandler(migrationSvc),
		Escalation:     handler.NewEscalationHandler(escalationSvc),
		Telegram:       handler.NewTelegramHandler(telegramLinkRepo, telegramBotSvc, telegramBotEnabled),
		Cluster:        handler.NewClusterHandler(monitorSvc),
		Anomaly:        handler.NewAnomalyHandler(anomalySvc),
		Prediction:     handler.NewPredictionHandler(predictionSvc),
		Briefing:       handler.NewBriefingHandler(briefingSvc),
		Approval:       handler.NewApprovalHandler(approvalRepo, agentSvc),
		Drift:          handler.NewDriftHandler(driftSvc),
		Environment:    handler.NewEnvironmentHandler(envSvc),
		Update:         handler.NewUpdateHandler(updateSvc),
		Rightsizing:    handler.NewRightsizingHandler(rightsizingSvc),
		SSHKey:         handler.NewSSHKeyHandler(sshkeySvc),
		Gateway:        handler.NewGatewayHandler(gatewaySvc, auditRepo),
		Log:            handler.NewLogHandler(nodeSvc),
		Topology:       handler.NewTopologyHandler(topologySvc),
		Brain:          handler.NewBrainHandler(brainSvc),
		Reflex:         handler.NewReflexHandler(reflexSvc),
		AgentConfig:    handler.NewAgentConfigHandler(agentConfigRepo, llmRegistry, ollamaProvider, encryptor),
		SyncCenter:     handler.NewSyncCenterHandler(nodeSvc, nodeRepo, tagRepo),
		Security:       handler.NewSecurityHandler(securityEventRepo, analysisSvc),
		PasswordPolicy: handler.NewPasswordPolicyHandler(policyRepo),
		Permission:     handler.NewPermissionHandler(rolePermissionRepo),
		VMCockpit:      handler.NewVMCockpitHandler(nodeSvc, vmPermSvc, jwtSvc, cfg.CORS.AllowOrigins, proxmoxTLS),
		VMPermission:   handler.NewVMPermissionHandler(vmPermSvc),
		VMGroup:        handler.NewVMGroupHandler(vmGroupSvc),
		VMHealth:       handler.NewVMHealthHandler(vmHealthSvc, vmRightsizingSvc, vmAnomalySvc, snapshotPolicySvc, scheduledActionSvc, vmDependencySvc),
		Operations:     handler.NewOperationsHandler(operationsSvc),

		// Log & Network Analysis
		LogAnalysis:       handler.NewLogAnalysisHandler(logReporter, logAnomalyRepo, logAnalysisRepo),
		LogBookmark:       handler.NewLogBookmarkHandler(logBookmarkRepo),
		LogSource:         handler.NewLogSourceHandler(logSourceRepo, logDiscoverySvc),
		LogExport:         handler.NewLogExportHandler(redisClient, logAnomalyRepo, logBookmarkRepo),
		LogReportSchedule: handler.NewLogReportScheduleHandler(logReportScheduleRepo),
		LogStream:         handler.NewLogStreamHandler(logStreamMgr, jwtSvc, cfg.CORS.AllowOrigins),
		NetworkScan:       handler.NewNetworkScanHandler(netScanner, networkScanRepo),
		NetworkDevice:     handler.NewNetworkDeviceHandler(networkDeviceRepo),
		NetworkAnomaly:    handler.NewNetworkAnomalyHandler(networkAnomalyRepo),
		ScanBaseline:      handler.NewScanBaselineHandler(scanBaselineRepo),
		Bandwidth:         handler.NewBandwidthHandler(nodeSvc),
	}

	// Setup routes
	api.SetupRouter(e, cfg, jwtSvc, handlers, gatewaySvc, redisClient, auditRepo, userRepo, rolePermissionRepo)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("server listening", slog.String("addr", addr))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("server goroutine panicked",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
				os.Exit(1)
			}
		}()
		if err := e.Start(addr); err != nil {
			slog.Info("server stopped", slog.Any("reason", err))
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := logStreamMgr.Shutdown(shutdownCtx); err != nil {
		slog.Error("log stream shutdown error", slog.Any("error", err))
	}
	if err := logConsumer.Shutdown(shutdownCtx); err != nil {
		slog.Error("log consumer shutdown error", slog.Any("error", err))
	}
	if err := netScanner.Shutdown(shutdownCtx); err != nil {
		slog.Error("network scanner shutdown error", slog.Any("error", err))
	}
	if err := wsHub.Shutdown(shutdownCtx); err != nil {
		slog.Error("ws hub shutdown error", slog.Any("error", err))
	}

	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", slog.Any("error", err))
	}

	slog.Info("server shutdown complete")
}

// buildProxmoxTLSConfig creates a *tls.Config based on the Proxmox configuration.
// Returns nil when TLS verification is disabled (InsecureSkipVerify mode).
func buildProxmoxTLSConfig(cfg config.ProxmoxConfig) *tls.Config {
	if cfg.TLSInsecure && cfg.TLSCACert == "" {
		slog.Warn("Proxmox TLS-Verifikation ist deaktiviert (PROXMOX_TLS_INSECURE=true). " +
			"Für Produktionsumgebungen wird empfohlen, ein CA-Zertifikat zu hinterlegen (PROXMOX_TLS_CA_CERT)")
		// Return nil so NewClient applies its insecure default
		return nil
	}

	tlsCfg := &tls.Config{}

	if cfg.TLSCACert != "" {
		caCert, err := os.ReadFile(cfg.TLSCACert)
		if err != nil {
			slog.Error("Proxmox CA-Zertifikat konnte nicht gelesen werden",
				slog.String("path", cfg.TLSCACert),
				slog.Any("error", err))
			os.Exit(1)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			slog.Error("Proxmox CA-Zertifikat konnte nicht geparst werden – ungültiges PEM-Format",
				slog.String("path", cfg.TLSCACert))
			os.Exit(1)
		}
		tlsCfg.RootCAs = caCertPool
		slog.Info("Proxmox TLS mit benutzerdefiniertem CA-Zertifikat konfiguriert",
			slog.String("ca_cert", cfg.TLSCACert))
	}

	if cfg.TLSInsecure {
		slog.Warn("Proxmox TLS-Verifikation ist deaktiviert (PROXMOX_TLS_INSECURE=true), " +
			"obwohl ein CA-Zertifikat konfiguriert ist – das CA-Zertifikat wird ignoriert")
		tlsCfg.InsecureSkipVerify = true
	}

	return tlsCfg
}
