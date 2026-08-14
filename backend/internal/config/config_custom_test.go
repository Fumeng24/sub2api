package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "sub2api-config-test-")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("{}\n"), 0o600); err != nil {
		panic(err)
	}
	if err := os.Setenv("DATA_DIR", dir); err != nil {
		panic(err)
	}
	if err := os.Setenv("LOG_ENV", "development"); err != nil {
		panic(err)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestLoadProductionRequiresTotpEncryptionKey(t *testing.T) {
	viper.Reset()
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("LOG_ENV", "production")
	t.Setenv("TOTP_ENCRYPTION_KEY", "")

	_, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "totp.encryption_key is required in production")
}

func TestLoadDefaultGatewayResponseHeaderTimeoutLeavesCloudflareFailoverBudget(t *testing.T) {
	resetViperWithJWTSecret(t)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 60, cfg.Gateway.ResponseHeaderTimeout)
}

func TestDeployExamplesKeepOpenAIResponseHeaderTimeoutEnabled(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	files := map[string]string{
		"deploy/.env.example":                  "GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=45",
		"deploy/config.example.yaml":           "openai_response_header_timeout: 45",
		"deploy/docker-compose.yml":            "GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=${GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT:-45}",
		"deploy/docker-compose.local.yml":      "GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=${GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT:-45}",
		"deploy/docker-compose.standalone.yml": "GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=${GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT:-45}",
		"deploy/docker-compose.dev.yml":        "GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=${GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT:-45}",
	}
	for rel, want := range files {
		rel := rel
		want := want
		t.Run(rel, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, rel))
			require.NoError(t, err)
			require.Contains(t, string(body), want)
		})
	}
}

func TestDeployComposePassesOpenAIProbeAndWeakFallbackConfig(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	composeFiles := []string{
		"deploy/docker-compose.yml",
		"deploy/docker-compose.local.yml",
		"deploy/docker-compose.standalone.yml",
		"deploy/docker-compose.dev.yml",
	}
	required := []string{
		"GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_TIMEOUT_SECONDS=${GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_TIMEOUT_SECONDS:-30}",
		"GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_RETRY_DELAY_SECONDS=${GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_RETRY_DELAY_SECONDS:-5}",
		"GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_MAX_CONCURRENCY=${GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_MAX_CONCURRENCY:-10000}",
		"GATEWAY_SCHEDULING_WEAK_FALLBACK_ENABLED=${GATEWAY_SCHEDULING_WEAK_FALLBACK_ENABLED:-false}",
	}
	for _, rel := range composeFiles {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, rel))
			require.NoError(t, err)
			content := string(body)
			for _, want := range required {
				require.Contains(t, content, want)
			}
		})
	}
}

func TestDeployComposePassesRuntimeTuningConfig(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	composeFiles := []string{
		"deploy/docker-compose.yml",
		"deploy/docker-compose.local.yml",
		"deploy/docker-compose.standalone.yml",
		"deploy/docker-compose.dev.yml",
	}
	required := []string{
		"SERVER_MAX_REQUEST_BODY_SIZE=${SERVER_MAX_REQUEST_BODY_SIZE:-268435456}",
		"SERVER_H2C_ENABLED=${SERVER_H2C_ENABLED:-true}",
		"GATEWAY_FORCE_CODEX_CLI=${GATEWAY_FORCE_CODEX_CLI:-false}",
		"GATEWAY_MAX_BODY_SIZE=${GATEWAY_MAX_BODY_SIZE:-268435456}",
		"GATEWAY_MAX_CONNS_PER_HOST=${GATEWAY_MAX_CONNS_PER_HOST:-2048}",
		"GATEWAY_MAX_IDLE_CONNS=${GATEWAY_MAX_IDLE_CONNS:-8192}",
		"GATEWAY_MAX_IDLE_CONNS_PER_HOST=${GATEWAY_MAX_IDLE_CONNS_PER_HOST:-4096}",
		"GATEWAY_SCHEDULING_STICKY_SESSION_MAX_WAITING=${GATEWAY_SCHEDULING_STICKY_SESSION_MAX_WAITING:-3}",
		"GATEWAY_SCHEDULING_STICKY_SESSION_WAIT_TIMEOUT=${GATEWAY_SCHEDULING_STICKY_SESSION_WAIT_TIMEOUT:-120s}",
		"GATEWAY_SCHEDULING_FALLBACK_WAIT_TIMEOUT=${GATEWAY_SCHEDULING_FALLBACK_WAIT_TIMEOUT:-30s}",
		"GATEWAY_SCHEDULING_FALLBACK_MAX_WAITING=${GATEWAY_SCHEDULING_FALLBACK_MAX_WAITING:-100}",
		"GATEWAY_SCHEDULING_LOAD_BATCH_ENABLED=${GATEWAY_SCHEDULING_LOAD_BATCH_ENABLED:-true}",
		"GATEWAY_SCHEDULING_DB_FALLBACK_ENABLED=${GATEWAY_SCHEDULING_DB_FALLBACK_ENABLED:-true}",
		"GATEWAY_SCHEDULING_OUTBOX_POLL_INTERVAL_SECONDS=${GATEWAY_SCHEDULING_OUTBOX_POLL_INTERVAL_SECONDS:-1}",
		"GATEWAY_SCHEDULING_FULL_REBUILD_INTERVAL_SECONDS=${GATEWAY_SCHEDULING_FULL_REBUILD_INTERVAL_SECONDS:-300}",
		"RATE_LIMIT_OVERLOAD_COOLDOWN_MINUTES=${RATE_LIMIT_OVERLOAD_COOLDOWN_MINUTES:-10}",
		"DASHBOARD_AGGREGATION_ENABLED=${DASHBOARD_AGGREGATION_ENABLED:-true}",
		"DASHBOARD_AGGREGATION_RETENTION_DAILY_DAYS=${DASHBOARD_AGGREGATION_RETENTION_DAILY_DAYS:-730}",
	}
	for _, rel := range composeFiles {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, rel))
			require.NoError(t, err)
			content := string(body)
			for _, want := range required {
				require.Contains(t, content, want)
			}
		})
	}
}

func TestDeployComposeRedisServiceUsesConfiguredMaxClients(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, rel := range []string{
		"deploy/docker-compose.yml",
		"deploy/docker-compose.local.yml",
		"deploy/docker-compose.dev.yml",
	} {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, rel))
			require.NoError(t, err)
			require.Contains(t, string(body), "--maxclients ${REDIS_MAXCLIENTS:-50000}")
		})
	}
}

func TestDeployBareMetalSystemdReadsRuntimeEnvFile(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	files := map[string][]string{
		"deploy/sub2api.service": {
			"ReadWritePaths=/opt/sub2api /etc/sub2api",
			"Environment=DATA_DIR=/etc/sub2api",
			"Environment=LOG_OUTPUT_FILE_PATH=/opt/sub2api/data/logs/sub2api.log",
			"EnvironmentFile=-/etc/sub2api/sub2api.env",
		},
		"deploy/install.sh": {
			"ENV_FILE=\"$CONFIG_DIR/sub2api.env\"",
			"RUNTIME_DROPIN_FILE=\"$SYSTEMD_DROPIN_DIR/runtime-env.conf\"",
			"ReadWritePaths=/opt/sub2api /etc/sub2api",
			"Environment=DATA_DIR=/etc/sub2api",
			"Environment=LOG_OUTPUT_FILE_PATH=/opt/sub2api/data/logs/sub2api.log",
			"EnvironmentFile=-/etc/sub2api/sub2api.env",
			"EnvironmentFile=-$ENV_FILE",
			"migrate_legacy_data_dir",
			"install_runtime_systemd_dropin",
		},
	}

	for rel, required := range files {
		rel := rel
		required := required
		t.Run(rel, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, rel))
			require.NoError(t, err)
			content := string(body)
			for _, want := range required {
				require.Contains(t, content, want)
			}
		})
	}
}

func TestDeployBareMetalRuntimeEnvExampleIncludesCriticalTuning(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	body, err := os.ReadFile(filepath.Join(root, "deploy/sub2api.env.example"))
	require.NoError(t, err)
	content := string(body)

	required := []string{
		"DATA_DIR=/etc/sub2api",
		"LOG_OUTPUT_FILE_PATH=/opt/sub2api/data/logs/sub2api.log",
		"SERVER_MAX_REQUEST_BODY_SIZE=268435456",
		"GATEWAY_MAX_BODY_SIZE=268435456",
		"GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=45",
		"GATEWAY_OPENAI_COMPACT_FIRST_OUTPUT_TIMEOUT_SECONDS=120",
		"GATEWAY_OPENAI_COMPACT_HIGH_EFFORT_FIRST_OUTPUT_TIMEOUT_SECONDS=180",
		"GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_RETRY_DELAY_SECONDS=5",
		"GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_MAX_CONCURRENCY=10000",
		"GATEWAY_MAX_CONNS_PER_HOST=2048",
		"GATEWAY_MAX_IDLE_CONNS=8192",
		"GATEWAY_MAX_IDLE_CONNS_PER_HOST=4096",
		"GATEWAY_SCHEDULING_WEAK_FALLBACK_ENABLED=false",
		"GATEWAY_SCHEDULING_DB_FALLBACK_ENABLED=true",
		"RATE_LIMIT_OVERLOAD_COOLDOWN_MINUTES=10",
	}
	for _, want := range required {
		require.Contains(t, content, want)
	}
}

func TestDeployBareMetalDocsMentionRuntimeEnvFile(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	body, err := os.ReadFile(filepath.Join(root, "deploy/README.md"))
	require.NoError(t, err)
	content := string(body)

	required := []string{
		"`sub2api.env.example` | Bare-metal runtime tuning environment template",
		"/etc/sub2api/sub2api.env",
		"EnvironmentFile=-/etc/sub2api/sub2api.env",
		"DATA_DIR=/etc/sub2api",
		"GATEWAY_OPENAI_RESPONSE_HEADER_TIMEOUT=45",
		"GATEWAY_OPENAI_COMPACT_FIRST_OUTPUT_TIMEOUT_SECONDS=120",
		"GATEWAY_OPENAI_COMPACT_HIGH_EFFORT_FIRST_OUTPUT_TIMEOUT_SECONDS=180",
		"GATEWAY_SCHEDULING_WEAK_FALLBACK_ENABLED=false",
		"GATEWAY_MAX_CONNS_PER_HOST=2048",
		"sudo systemctl show sub2api --property=Environment",
	}
	for _, want := range required {
		require.Contains(t, content, want)
	}
}

func TestDeployExampleKeepsGatewayResponseHeaderTimeoutBelowCloudflare(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	body, err := os.ReadFile(filepath.Join(root, "deploy/config.example.yaml"))
	require.NoError(t, err)
	require.Contains(t, string(body), "response_header_timeout: 60")
	require.NotContains(t, string(body), "response_header_timeout: 100")
	require.NotContains(t, string(body), "response_header_timeout: 600")
}

func TestLoadOpenAIProbeRunnerConfigFromEnv(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_TIMEOUT_SECONDS", "7")
	t.Setenv("GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_RETRY_DELAY_SECONDS", "2")
	t.Setenv("GATEWAY_OPENAI_SCHEDULER_RUNTIME_COOLDOWNS_PROBE_MAX_CONCURRENCY", "3")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 7, cfg.Gateway.OpenAIScheduler.RuntimeCooldowns.ProbeTimeoutSeconds)
	require.Equal(t, 2, cfg.Gateway.OpenAIScheduler.RuntimeCooldowns.ProbeRetryDelaySeconds)
	require.Equal(t, 3, cfg.Gateway.OpenAIScheduler.RuntimeCooldowns.ProbeMaxConcurrency)
}

func TestLoadGatewaySchedulingWeakFallbackFromEnv(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GATEWAY_SCHEDULING_WEAK_FALLBACK_ENABLED", "true")

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.Gateway.Scheduling.WeakFallbackEnabled)
}

func TestLoadDefaultOpenAIRuntimeCooldownsCustom(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	require.NoError(t, err)

	runtime := cfg.Gateway.OpenAIScheduler.RuntimeCooldowns
	require.Equal(t, 5, runtime.OAuth429FallbackCooldownSeconds)
	require.Equal(t, 30, runtime.RequestErrorCooldownSeconds)
	require.Equal(t, 15, runtime.TransientCooldownPersistMinIntervalSeconds)
	require.Equal(t, 120, runtime.StopSchedulingBridgeCooldownSeconds)
	require.Equal(t, 30, runtime.AccountCircuitHalfOpenProbeTTLSeconds)
	require.Equal(t, 30, runtime.ProbeTimeoutSeconds)
	require.Equal(t, 5, runtime.ProbeRetryDelaySeconds)
	require.Equal(t, 10000, runtime.ProbeMaxConcurrency)
}

func TestLoadDefaultOpenAISlowReserveCustom(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	require.NoError(t, err)

	reserve := cfg.Gateway.OpenAIScheduler.SlowReserve
	require.True(t, reserve.Enabled)
	require.Equal(t, 15000, reserve.TTFTMs)
	require.Equal(t, 180, reserve.TTLSeconds)
	require.Equal(t, 4096, reserve.MaxEntries)
}

func TestLoadOpenAISlowReserveFromEnv(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GATEWAY_OPENAI_SCHEDULER_SLOW_RESERVE_ENABLED", "false")
	t.Setenv("GATEWAY_OPENAI_SCHEDULER_SLOW_RESERVE_TTFT_MS", "13000")
	t.Setenv("GATEWAY_OPENAI_SCHEDULER_SLOW_RESERVE_TTL_SECONDS", "180")
	t.Setenv("GATEWAY_OPENAI_SCHEDULER_SLOW_RESERVE_MAX_ENTRIES", "512")

	cfg, err := Load()
	require.NoError(t, err)
	reserve := cfg.Gateway.OpenAIScheduler.SlowReserve
	require.False(t, reserve.Enabled)
	require.Equal(t, 13000, reserve.TTFTMs)
	require.Equal(t, 180, reserve.TTLSeconds)
	require.Equal(t, 512, reserve.MaxEntries)
}

func TestValidateOpenAISlowReserveCustom(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GatewayOpenAISlowReserveConfig)
		want   string
	}{
		{"ttft positive", func(c *GatewayOpenAISlowReserveConfig) { c.TTFTMs = 0 }, "slow_reserve.ttft_ms"},
		{"ttl positive", func(c *GatewayOpenAISlowReserveConfig) { c.TTLSeconds = 0 }, "slow_reserve.ttl_seconds"},
		{"entries positive", func(c *GatewayOpenAISlowReserveConfig) { c.MaxEntries = 0 }, "slow_reserve.max_entries"},
		{"entries bounded", func(c *GatewayOpenAISlowReserveConfig) { c.MaxEntries = 100001 }, "slow_reserve.max_entries"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViperWithJWTSecret(t)
			cfg, err := Load()
			require.NoError(t, err)
			tt.mutate(&cfg.Gateway.OpenAIScheduler.SlowReserve)
			err = cfg.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidateOpenAIRuntimeCooldownsCustom(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GatewayOpenAIRuntimeCooldownsConfig)
		want   string
	}{
		{"oauth fallback positive", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.OAuth429FallbackCooldownSeconds = 0 }, "oauth_429_fallback_cooldown_seconds"},
		{"request error positive", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.RequestErrorCooldownSeconds = 0 }, "request_error_cooldown_seconds"},
		{"persist interval positive", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.TransientCooldownPersistMinIntervalSeconds = 0 }, "transient_cooldown_persist_min_interval_seconds"},
		{"bridge cooldown positive", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.StopSchedulingBridgeCooldownSeconds = 0 }, "stop_scheduling_bridge_cooldown_seconds"},
		{"half-open ttl positive", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.AccountCircuitHalfOpenProbeTTLSeconds = 0 }, "account_circuit_half_open_probe_ttl_seconds"},
		{"request error duration bounded", func(c *GatewayOpenAIRuntimeCooldownsConfig) {
			c.RequestErrorCooldownSeconds = int(maxDurationSeconds) + 1
		}, "request_error_cooldown_seconds"},
		{"probe timeout positive", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.ProbeTimeoutSeconds = 0 }, "probe_timeout_seconds"},
		{"probe retry non-negative", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.ProbeRetryDelaySeconds = -1 }, "probe_retry_delay_seconds"},
		{"probe timeout bounded", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.ProbeTimeoutSeconds = int(maxDurationSeconds) + 1 }, "probe_timeout_seconds"},
		{"probe concurrency positive", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.ProbeMaxConcurrency = 0 }, "probe_max_concurrency"},
		{"probe concurrency bounded", func(c *GatewayOpenAIRuntimeCooldownsConfig) { c.ProbeMaxConcurrency = 10001 }, "probe_max_concurrency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViperWithJWTSecret(t)
			cfg, err := Load()
			require.NoError(t, err)
			tt.mutate(&cfg.Gateway.OpenAIScheduler.RuntimeCooldowns)
			err = cfg.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidateOpenAIWSHTTPBridgeIngressCustom(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	require.NoError(t, err)
	cfg.Gateway.OpenAIWS.IngressModeDefault = "http_bridge"
	require.NoError(t, cfg.Validate())
}

func TestValidateOpenAIWSQuotaHeadroomAsOnlyWeightCustom(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	require.NoError(t, err)

	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Priority = 0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Load = 0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Queue = 0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.ErrorRate = 0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.TTFT = 0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.Reset = 0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.QuotaHeadroom = 0.1
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.PreviousResponse = 0
	cfg.Gateway.OpenAIWS.SchedulerScoreWeights.SessionSticky = 0

	require.NoError(t, cfg.Validate())
}
