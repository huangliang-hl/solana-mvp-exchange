package executor

import (
	"time"

	"solana-limit-order-backend/internal/config"

	"github.com/rs/zerolog/log"
)

// CreateJupiterClient 根据配置创建 Jupiter 客户端
func CreateJupiterClient(cfg *config.Config) JupiterClientInterface {
	if cfg.ShouldUseJupiterMock() {
		log.Info().
			Bool("mock_enabled", cfg.JupiterMockEnabled).
			Bool("is_devnet", cfg.IsDevnet()).
			Float64("success_rate", cfg.JupiterMockSuccessRate).
			Int("latency_ms", cfg.JupiterMockLatencyMs).
			Msg("使用 Jupiter 模拟器")

		mockConfig := &JupiterMockConfig{
			Enabled:     true,
			SuccessRate: cfg.JupiterMockSuccessRate,
			LatencyMs:   time.Duration(cfg.JupiterMockLatencyMs) * time.Millisecond,
			MinSlippage: cfg.JupiterMockMinSlippage,
			MaxSlippage: cfg.JupiterMockMaxSlippage,
		}

		return NewJupiterMockClient(mockConfig)
	}

	log.Info().
		Str("endpoint", cfg.JupiterEndpoint).
		Msg("使用真实 Jupiter API")

	return NewJupiterClient(cfg.JupiterEndpoint)
}
