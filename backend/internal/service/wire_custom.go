package service

import (
	"database/sql"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/wire"
)

var CustomProviderSet = wire.NewSet(
	ProvideAdminServiceCustomDependencies,
	ApplyAdminServiceCustomization,
	ApplyConcurrencyCacheCustomization,
	ProvideTicketService,
	NewInvoiceService,
	NewWhamVerifyService,
	ProvideSlotPoolService,
	ProvideAccountMonitorService,
	ProvideTransientRecoveryProbeService,
	ProvideAccountMonitorRunner,
	ProvideGroupAutoSortService,
	ApplyRateLimitCustomization,
	ApplyOpsRuntimeCustomization,
	ApplyAPIKeyCustomization,
	ApplyGatewayCustomization,
	ApplyChannelMonitorCustomization,
)

type RateLimitCustomization struct{}

func ApplyRateLimitCustomization(svc *RateLimitService, cache TransientErrorCounterCache) RateLimitCustomization {
	svc.SetTransientErrorCounterCache(cache)
	return RateLimitCustomization{}
}

type OpsRuntimeCustomization struct{}

func ApplyOpsRuntimeCustomization(svc *OpsService, repo OpsRepository, email *EmailService, gateway *OpenAIGatewayService) OpsRuntimeCustomization {
	runtimeAlerts := NewOpsRuntimeAlertService(repo, svc, email)
	if svc != nil {
		svc.SetRuntimeAlertService(runtimeAlerts)
	}
	_ = gateway
	return OpsRuntimeCustomization{}
}

type APIKeyCustomization struct{}

func ApplyAPIKeyCustomization(svc *APIKeyService, repo AccountRepository) APIKeyCustomization {
	svc.SetAccountRepository(repo)
	return APIKeyCustomization{}
}

type GatewayCustomization struct{}

func ApplyGatewayCustomization(svc *GatewayService, slotPool SlotPoolService, gemini *GeminiTokenProvider, antigravity *AntigravityTokenProvider) GatewayCustomization {
	svc.slotPoolService = slotPool
	svc.geminiTokenProvider = gemini
	svc.SetAntigravityTokenProvider(antigravity)
	return GatewayCustomization{}
}

type ChannelMonitorCustomization struct{}

func ApplyChannelMonitorCustomization(svc *ChannelMonitorService, repo APIKeyRepository) ChannelMonitorCustomization {
	svc.SetAPIKeyRepository(repo)
	return ChannelMonitorCustomization{}
}

func ProvideTicketService(ticketRepo TicketRepository, userRepo UserRepository, emailService *EmailService, settingRepo SettingRepository) *TicketService {
	svc := NewTicketService(ticketRepo, userRepo, emailService, settingRepo)
	svc.Start()
	return svc
}

func ProvideSlotPoolService(cache SlotPoolCache, schedulerCache SchedulerCache, concurrencyCache ConcurrencyCache, cfg *config.Config) SlotPoolService {
	svc := NewSlotPoolService(cache, schedulerCache, concurrencyCache, cfg)
	svc.Start()
	return svc
}

func ProvideAccountMonitorService(repo AccountMonitorRepository, account AccountRepository, recovery *RateLimitService) *AccountMonitorService {
	svc := NewAccountMonitorService(repo, account)
	svc.SetRecovery(recovery)
	if blocker, ok := account.(AccountMonitorFailureBlocker); ok {
		svc.SetFailureBlocker(blocker)
	}
	return svc
}

func ProvideTransientRecoveryProbeService(account AccountRepository, monitor *AccountMonitorService, recovery *RateLimitService) *TransientRecoveryProbeService {
	svc := NewTransientRecoveryProbeService(account, monitor, recovery)
	if recovery != nil {
		recovery.SetTransientRecoveryProbeScheduler(svc)
	}
	svc.Start()
	return svc
}

func ProvideGroupAutoSortService(admin AdminService, accountMonitorService *AccountMonitorService, lockCache LeaderLockCache, db *sql.DB) *GroupAutoSortService {
	adapter, ok := admin.(groupAutoSortAdmin)
	if !ok {
		logger.LegacyPrintf("service.group_auto_sort", "AdminService does not expose scheduling methods; auto-sort disabled")
		return nil
	}
	svc := NewGroupAutoSortService(adapter, accountMonitorService, groupAutoSortDefaultInterval)
	provider := newSQLGroupAutoSortExperienceProvider(db)
	svc.SetExperienceProvider(provider)
	svc.SetRateProvider(provider)
	svc.SetLeaderLock(lockCache, db)
	svc.Start()
	return svc
}

func ProvideAccountMonitorRunner(svc *AccountMonitorService, settingService *SettingService) *AccountMonitorRunner {
	r := NewAccountMonitorRunner(svc, settingService)
	svc.SetScheduler(r)
	r.Start()
	return r
}

type ConcurrencyCacheCustomization struct{}

type inlineCleanupOnReadConfigurator interface {
	SetInlineCleanupOnRead(bool)
}

func ApplyConcurrencyCacheCustomization(cache ConcurrencyCache, cfg *config.Config) ConcurrencyCacheCustomization {
	if configurable, ok := cache.(inlineCleanupOnReadConfigurator); ok {
		configurable.SetInlineCleanupOnRead(cfg != nil && cfg.Gateway.Scheduling.SlotCleanupInterval <= 0)
	}
	return ConcurrencyCacheCustomization{}
}
