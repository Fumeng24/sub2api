package main

import (
	"log"
	"sync"

	"github.com/Wei-Shaw/sub2api/ent"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// provideCustomCleanup keeps the upstream cleanup lifecycle intact and only
// layers shutdown hooks for services implemented by the local overlay.
func provideCustomCleanup(
	entClient *ent.Client,
	rdb *redis.Client,
	opsMetricsCollector *service.OpsMetricsCollector,
	opsAggregation *service.OpsAggregationService,
	opsAlertEvaluator *service.OpsAlertEvaluatorService,
	opsCleanup *service.OpsCleanupService,
	opsScheduledReport *service.OpsScheduledReportService,
	opsSystemLogSink *service.OpsSystemLogSink,
	opsService *service.OpsService,
	opsIngressReject *service.OpsIngressRejectAggregator,
	apiKeyService *service.APIKeyService,
	authCacheInvalidationWorker *service.AuthCacheInvalidationWorker,
	schedulerSnapshot *service.SchedulerSnapshotService,
	tokenRefresh *service.TokenRefreshService,
	accountExpiry *service.AccountExpiryService,
	codexVersionSync *service.OpenAICodexVersionSyncService,
	proxyExpiry *service.ProxyExpiryService,
	subscriptionExpiry *service.SubscriptionExpiryService,
	usageCleanup *service.UsageCleanupService,
	idempotencyCleanup *service.IdempotencyCleanupService,
	batchImageCleanup *service.BatchImageCleanupService,
	batchImageWorker *service.BatchImageWorkerRuntime,
	pricing *service.PricingService,
	emailQueue *service.EmailQueueService,
	billingCache *service.BillingCacheService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	subscriptionService *service.SubscriptionService,
	oauth *service.OAuthService,
	openaiOAuth *service.OpenAIOAuthService,
	geminiOAuth *service.GeminiOAuthService,
	antigravityOAuth *service.AntigravityOAuthService,
	grokOAuth *service.GrokOAuthService,
	openAIGateway *service.OpenAIGatewayService,
	scheduledTestRunner *service.ScheduledTestRunnerService,
	backupSvc *service.BackupService,
	paymentOrderExpiry *service.PaymentOrderExpiryService,
	channelMonitorRunner *service.ChannelMonitorRunner,
	quotaFlusher *service.UserPlatformQuotaUsageFlusher,
	upstreamBillingProbe *service.UpstreamBillingProbeService,
	upstreamHandler *adminhandler.UpstreamHandler,
	ollamaCloudUsage *service.OllamaCloudUsageService,
	auditLog *service.AuditLogService,
	promptAudit *securityaudit.PromptService,
	ticketService *service.TicketService,
	accountMonitorRunner *service.AccountMonitorRunner,
	transientRecoveryProbe *service.TransientRecoveryProbeService,
	groupAutoSortService *service.GroupAutoSortService,
) func() {
	upstreamCleanup := provideCleanup(
		entClient,
		rdb,
		opsMetricsCollector,
		opsAggregation,
		opsAlertEvaluator,
		opsCleanup,
		opsScheduledReport,
		opsSystemLogSink,
		opsService,
		opsIngressReject,
		apiKeyService,
		authCacheInvalidationWorker,
		schedulerSnapshot,
		tokenRefresh,
		accountExpiry,
		codexVersionSync,
		proxyExpiry,
		subscriptionExpiry,
		usageCleanup,
		idempotencyCleanup,
		batchImageCleanup,
		batchImageWorker,
		pricing,
		emailQueue,
		billingCache,
		usageRecordWorkerPool,
		subscriptionService,
		oauth,
		openaiOAuth,
		geminiOAuth,
		antigravityOAuth,
		grokOAuth,
		openAIGateway,
		scheduledTestRunner,
		backupSvc,
		paymentOrderExpiry,
		channelMonitorRunner,
		quotaFlusher,
		upstreamBillingProbe,
		upstreamHandler,
		ollamaCloudUsage,
		auditLog,
		promptAudit,
	)

	return func() {
		steps := []struct {
			name string
			stop func()
		}{
			{"TicketService", func() {
				if ticketService != nil {
					ticketService.Stop()
				}
			}},
			{"AccountMonitorRunner", func() {
				if accountMonitorRunner != nil {
					accountMonitorRunner.Stop()
				}
			}},
			{"TransientRecoveryProbeService", func() {
				if transientRecoveryProbe != nil {
					transientRecoveryProbe.Stop()
				}
			}},
			{"GroupAutoSortService", func() {
				if groupAutoSortService != nil {
					groupAutoSortService.Stop()
				}
			}},
		}

		var wg sync.WaitGroup
		for i := range steps {
			step := steps[i]
			wg.Add(1)
			go func() {
				defer wg.Done()
				step.stop()
				log.Printf("[Cleanup] %s succeeded", step.name)
			}()
		}
		wg.Wait()

		upstreamCleanup()
	}
}
