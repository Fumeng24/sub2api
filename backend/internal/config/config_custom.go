package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const maxDurationSeconds = int64(time.Duration(1<<63-1) / time.Second)
const maxDurationMilliseconds = int64(time.Duration(1<<63-1) / time.Millisecond)

type GatewayOpenAISchedulerConfigCustom struct {
	RuntimeCooldowns GatewayOpenAIRuntimeCooldownsConfig `mapstructure:"runtime_cooldowns"`
	SlowReserve      GatewayOpenAISlowReserveConfig      `mapstructure:"slow_reserve"`
}

type GatewayOpenAIRuntimeCooldownsConfig struct {
	OAuth429FallbackCooldownSeconds            int `mapstructure:"oauth_429_fallback_cooldown_seconds"`
	RequestErrorCooldownSeconds                int `mapstructure:"request_error_cooldown_seconds"`
	TransientCooldownPersistMinIntervalSeconds int `mapstructure:"transient_cooldown_persist_min_interval_seconds"`
	StopSchedulingBridgeCooldownSeconds        int `mapstructure:"stop_scheduling_bridge_cooldown_seconds"`
	AccountCircuitHalfOpenProbeTTLSeconds      int `mapstructure:"account_circuit_half_open_probe_ttl_seconds"`
	ProbeTimeoutSeconds                        int `mapstructure:"probe_timeout_seconds"`
	ProbeRetryDelaySeconds                     int `mapstructure:"probe_retry_delay_seconds"`
	ProbeMaxConcurrency                        int `mapstructure:"probe_max_concurrency"`
}

// GatewayOpenAISlowReserveConfig keeps slow-but-usable OpenAI accounts as
// model-scoped reserve candidates instead of turning them into account errors.
type GatewayOpenAISlowReserveConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	TTFTMs     int  `mapstructure:"ttft_ms"`
	TTLSeconds int  `mapstructure:"ttl_seconds"`
	MaxEntries int  `mapstructure:"max_entries"`
}

type GatewaySchedulingConfigCustom struct {
	WeakFallbackEnabled bool           `mapstructure:"weak_fallback_enabled"`
	SlotPool            SlotPoolConfig `mapstructure:"slot_pool"`
}

type SlotPoolConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	RebuildInterval time.Duration `mapstructure:"rebuild_interval"`
	MaxRetries      int           `mapstructure:"max_retries"`
	FallbackOnError bool          `mapstructure:"fallback_on_error"`
}

func setCustomDefaults() {
	viper.SetDefault("gateway.response_header_timeout", 60)
	viper.SetDefault("gateway.openai_response_header_timeout", 45)
	// Remote compaction handles substantially larger request bodies than ordinary
	// Responses calls. Keep its semantic timeout independent from normal streams.
	viper.SetDefault("gateway.openai_compact_first_output_timeout_seconds", 120)
	viper.SetDefault("gateway.openai_compact_high_effort_first_output_timeout_seconds", 180)
	viper.SetDefault("gateway.failover_on_400", true)
	viper.SetDefault("gateway.openai_scheduler.sticky_escape_enabled", true)
	// Only a near-timeout slow response should break an active sticky session;
	// hard provider errors still escape immediately.
	viper.SetDefault("gateway.openai_scheduler.sticky_escape_ttft_ms", 25000)
	viper.SetDefault("gateway.openai_scheduler.sticky_escape_error_rate", 0.5)
	viper.SetDefault("gateway.openai_scheduler.runtime_cooldowns.oauth_429_fallback_cooldown_seconds", 5)
	viper.SetDefault("gateway.openai_scheduler.runtime_cooldowns.request_error_cooldown_seconds", 30)
	viper.SetDefault("gateway.openai_scheduler.runtime_cooldowns.transient_cooldown_persist_min_interval_seconds", 15)
	viper.SetDefault("gateway.openai_scheduler.runtime_cooldowns.stop_scheduling_bridge_cooldown_seconds", 120)
	viper.SetDefault("gateway.openai_scheduler.runtime_cooldowns.account_circuit_half_open_probe_ttl_seconds", 30)
	viper.SetDefault("gateway.openai_scheduler.runtime_cooldowns.probe_timeout_seconds", 30)
	viper.SetDefault("gateway.openai_scheduler.runtime_cooldowns.probe_retry_delay_seconds", 5)
	viper.SetDefault("gateway.openai_scheduler.runtime_cooldowns.probe_max_concurrency", 10000)
	viper.SetDefault("gateway.openai_scheduler.slow_reserve.enabled", true)
	// The first slow success is a pending signal; the second one in this short
	// window promotes the account/model to the reserve pool.
	viper.SetDefault("gateway.openai_scheduler.slow_reserve.ttft_ms", 15000)
	viper.SetDefault("gateway.openai_scheduler.slow_reserve.ttl_seconds", 180)
	viper.SetDefault("gateway.openai_scheduler.slow_reserve.max_entries", 4096)
	viper.SetDefault("gateway.max_body_size", int64(32*1024*1024))
	viper.SetDefault("gateway.scheduling.weak_fallback_enabled", false)
	viper.SetDefault("gateway.scheduling.slot_pool.enabled", true)
	viper.SetDefault("gateway.scheduling.slot_pool.rebuild_interval", 5*time.Minute)
	viper.SetDefault("gateway.scheduling.slot_pool.max_retries", 10)
	viper.SetDefault("gateway.scheduling.slot_pool.fallback_on_error", true)
}

func validateCustomTOTPEnvironment(cfg *Config) error {
	if cfg == nil || !isProductionLikeEnvironment(cfg.Log.Environment) {
		return nil
	}
	return fmt.Errorf("totp.encryption_key is required in production; set TOTP_ENCRYPTION_KEY to a stable 32-byte hex key")
}

func isProductionLikeEnvironment(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func validateCustomOpenAIRuntimeCooldowns(c *Config) error {
	runtimeCooldowns := c.Gateway.OpenAIScheduler.RuntimeCooldowns
	checks := []struct {
		path    string
		seconds int
	}{
		{"gateway.openai_scheduler.runtime_cooldowns.oauth_429_fallback_cooldown_seconds", runtimeCooldowns.OAuth429FallbackCooldownSeconds},
		{"gateway.openai_scheduler.runtime_cooldowns.request_error_cooldown_seconds", runtimeCooldowns.RequestErrorCooldownSeconds},
		{"gateway.openai_scheduler.runtime_cooldowns.transient_cooldown_persist_min_interval_seconds", runtimeCooldowns.TransientCooldownPersistMinIntervalSeconds},
		{"gateway.openai_scheduler.runtime_cooldowns.stop_scheduling_bridge_cooldown_seconds", runtimeCooldowns.StopSchedulingBridgeCooldownSeconds},
		{"gateway.openai_scheduler.runtime_cooldowns.account_circuit_half_open_probe_ttl_seconds", runtimeCooldowns.AccountCircuitHalfOpenProbeTTLSeconds},
		{"gateway.openai_scheduler.runtime_cooldowns.probe_timeout_seconds", runtimeCooldowns.ProbeTimeoutSeconds},
		{"gateway.openai_scheduler.runtime_cooldowns.probe_retry_delay_seconds", runtimeCooldowns.ProbeRetryDelaySeconds},
	}
	for _, check := range checks {
		if err := validatePositiveDurationSeconds(check.path, check.seconds); err != nil {
			return err
		}
	}
	if runtimeCooldowns.ProbeMaxConcurrency <= 0 {
		return fmt.Errorf("gateway.openai_scheduler.runtime_cooldowns.probe_max_concurrency must be positive")
	}
	if runtimeCooldowns.ProbeMaxConcurrency > 10000 {
		return fmt.Errorf("gateway.openai_scheduler.runtime_cooldowns.probe_max_concurrency must be <= 10000")
	}
	return nil
}

func validateCustomOpenAISlowReserve(c *Config) error {
	if c == nil {
		return nil
	}
	reserve := c.Gateway.OpenAIScheduler.SlowReserve
	if reserve.TTFTMs <= 0 {
		return fmt.Errorf("gateway.openai_scheduler.slow_reserve.ttft_ms must be positive")
	}
	if int64(reserve.TTFTMs) > maxDurationMilliseconds {
		return fmt.Errorf("gateway.openai_scheduler.slow_reserve.ttft_ms must be <= %d", maxDurationMilliseconds)
	}
	if err := validatePositiveDurationSeconds("gateway.openai_scheduler.slow_reserve.ttl_seconds", reserve.TTLSeconds); err != nil {
		return err
	}
	if reserve.MaxEntries <= 0 {
		return fmt.Errorf("gateway.openai_scheduler.slow_reserve.max_entries must be positive")
	}
	if reserve.MaxEntries > 100000 {
		return fmt.Errorf("gateway.openai_scheduler.slow_reserve.max_entries must be <= 100000")
	}
	return nil
}

func validateCustomOpenAIWSScoreWeights(c *Config) error {
	if c.Gateway.OpenAIWS.SchedulerScoreWeights.Reset < 0 {
		return fmt.Errorf("gateway.openai_ws.scheduler_score_weights values must be non-negative")
	}
	return nil
}

func addCustomOpenAIWSScoreWeights(c *Config, base float64) float64 {
	weights := c.Gateway.OpenAIWS.SchedulerScoreWeights
	return base + weights.Reset + weights.PreviousResponse + weights.SessionSticky
}

func validateCustomSlotPool(c *Config) error {
	slotPool := c.Gateway.Scheduling.SlotPool
	if !slotPool.Enabled {
		return nil
	}
	if slotPool.RebuildInterval < 0 {
		return fmt.Errorf("gateway.scheduling.slot_pool.rebuild_interval must be non-negative")
	}
	if slotPool.MaxRetries <= 0 {
		return fmt.Errorf("gateway.scheduling.slot_pool.max_retries must be positive when slot_pool is enabled")
	}
	return nil
}

func validatePositiveDurationSeconds(path string, seconds int) error {
	if seconds <= 0 {
		return fmt.Errorf("%s must be positive", path)
	}
	if int64(seconds) > maxDurationSeconds {
		return fmt.Errorf("%s must be <= %d", path, maxDurationSeconds)
	}
	return nil
}
