package executor

import (
	"fmt"
	"testing"

	"solana-limit-order-backend/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestCreateJupiterClient(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		expectedType  string
		shouldUseMock bool
	}{
		{
			name: "显式启用模拟器",
			config: &config.Config{
				JupiterMockEnabled:     true,
				RPCUrl:                 "https://api.mainnet-beta.solana.com",
				JupiterEndpoint:        "https://lite-api.jup.ag/swap/v1/",
				JupiterMockSuccessRate: 0.95,
				JupiterMockLatencyMs:   200,
				JupiterMockMinSlippage: 0.1,
				JupiterMockMaxSlippage: 2.0,
			},
			expectedType:  "*executor.JupiterMockClient",
			shouldUseMock: true,
		},
		{
			name: "devnet 环境自动启用模拟器",
			config: &config.Config{
				JupiterMockEnabled:     false,
				RPCUrl:                 "https://api.devnet.solana.com",
				JupiterEndpoint:        "https://lite-api.jup.ag/swap/v1/",
				JupiterMockSuccessRate: 0.95,
				JupiterMockLatencyMs:   200,
				JupiterMockMinSlippage: 0.1,
				JupiterMockMaxSlippage: 2.0,
			},
			expectedType:  "*executor.JupiterMockClient",
			shouldUseMock: true,
		},
		{
			name: "mainnet 环境使用真实客户端",
			config: &config.Config{
				JupiterMockEnabled:     false,
				RPCUrl:                 "https://api.mainnet-beta.solana.com",
				JupiterEndpoint:        "https://lite-api.jup.ag/swap/v1/",
				JupiterMockSuccessRate: 0.95,
				JupiterMockLatencyMs:   200,
				JupiterMockMinSlippage: 0.1,
				JupiterMockMaxSlippage: 2.0,
			},
			expectedType:  "*executor.JupiterClient",
			shouldUseMock: false,
		},
		{
			name: "testnet 环境使用真实客户端",
			config: &config.Config{
				JupiterMockEnabled:     false,
				RPCUrl:                 "https://api.testnet.solana.com",
				JupiterEndpoint:        "https://lite-api.jup.ag/swap/v1/",
				JupiterMockSuccessRate: 0.95,
				JupiterMockLatencyMs:   200,
				JupiterMockMinSlippage: 0.1,
				JupiterMockMaxSlippage: 2.0,
			},
			expectedType:  "*executor.JupiterClient",
			shouldUseMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := CreateJupiterClient(tt.config)

			// 验证客户端不为空
			assert.NotNil(t, client)

			// 验证客户端类型
			clientType := getClientType(client)
			assert.Equal(t, tt.expectedType, clientType)

			// 验证配置决策逻辑
			assert.Equal(t, tt.shouldUseMock, tt.config.ShouldUseJupiterMock())
		})
	}
}

func TestCreateJupiterClient_InterfaceCompliance(t *testing.T) {
	// 测试返回的客户端都实现了 JupiterClientInterface 接口
	configs := []*config.Config{
		{
			JupiterMockEnabled:     true,
			RPCUrl:                 "https://api.devnet.solana.com",
			JupiterEndpoint:        "https://lite-api.jup.ag/swap/v1/",
			JupiterMockSuccessRate: 0.95,
			JupiterMockLatencyMs:   200,
			JupiterMockMinSlippage: 0.1,
			JupiterMockMaxSlippage: 2.0,
		},
		{
			JupiterMockEnabled:     false,
			RPCUrl:                 "https://api.mainnet-beta.solana.com",
			JupiterEndpoint:        "https://lite-api.jup.ag/swap/v1/",
			JupiterMockSuccessRate: 0.95,
			JupiterMockLatencyMs:   200,
			JupiterMockMinSlippage: 0.1,
			JupiterMockMaxSlippage: 2.0,
		},
	}

	for i, cfg := range configs {
		t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
			client := CreateJupiterClient(cfg)

			// 验证客户端实现了接口
			var _ JupiterClientInterface = client

			// 验证客户端不为空
			assert.NotNil(t, client)
		})
	}
}

// getClientType 获取客户端类型的字符串表示
func getClientType(client JupiterClientInterface) string {
	switch client.(type) {
	case *JupiterMockClient:
		return "*executor.JupiterMockClient"
	case *JupiterClient:
		return "*executor.JupiterClient"
	default:
		return "unknown"
	}
}
