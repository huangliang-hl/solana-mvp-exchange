package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    *Config
	}{
		{
			name: "默认配置",
			envVars: map[string]string{
				"RPC_URL":     "https://api.devnet.solana.com",
				"PRIVATE_KEY": "ca485ee78080c81cb6eaf6900d16ebdafd13ccc0d2e690f1c575c5add809989dfa1244d62020a51946191fe593894a1cc",
			},
			want: &Config{
				DatabaseURL:     "postgres://postgres:password@localhost:5432/solana_limit_order?sslmode=disable",
				RedisURL:        "redis://localhost:6379/0",
				RPCUrl:          "https://api.devnet.solana.com",
				PrivateKey:      "ca485ee78080c81cb6eaf6900d16ebdafd13ccc0d2e690f1c575c5add809989dfa1244d62020a51946191fe593894a1cc",
				JupiterEndpoint: "https://lite-api.jup.ag/swap/v1/",
				APIBind:         "0.0.0.0:8080",
				PriceWSBind:     "0.0.0.0:8081",
				Mode:            "dev",
				LogLevel:        "info",
				APIKeys:         []string{"test-key-1"},
			},
		},
		{
			name: "自定义配置",
			envVars: map[string]string{
				"DB_URL":           "postgres://test:test@localhost:5432/test?sslmode=disable",
				"REDIS_URL":        "redis://localhost:6379/1",
				"RPC_URL":          "https://api.mainnet-beta.solana.com",
				"PRIVATE_KEY":      "test-private-key",
				"JUPITER_ENDPOINT": "https://quote-api.jup.ag/v7",
				"API_BIND":         "127.0.0.1:9080",
				"PRICE_WS_BIND":    "127.0.0.1:9081",
				"MODE":             "prod",
				"LOG_LEVEL":        "error",
				"API_KEYS":         "key1,key2,key3",
			},
			want: &Config{
				DatabaseURL:     "postgres://test:test@localhost:5432/test?sslmode=disable",
				RedisURL:        "redis://localhost:6379/1",
				RPCUrl:          "https://api.mainnet-beta.solana.com",
				PrivateKey:      "test-private-key",
				JupiterEndpoint: "https://quote-api.jup.ag/v7",
				APIBind:         "127.0.0.1:9080",
				PriceWSBind:     "127.0.0.1:9081",
				Mode:            "prod",
				LogLevel:        "error",
				APIKeys:         []string{"key1", "key2", "key3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理环境变量
			clearEnvVars()

			// 设置测试环境变量
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// 加载配置
			config, err := Load()
			require.NoError(t, err)

			// 验证配置
			assert.Equal(t, tt.want.DatabaseURL, config.DatabaseURL)
			assert.Equal(t, tt.want.RedisURL, config.RedisURL)
			assert.Equal(t, tt.want.RPCUrl, config.RPCUrl)
			assert.Equal(t, tt.want.PrivateKey, config.PrivateKey)
			assert.Equal(t, tt.want.JupiterEndpoint, config.JupiterEndpoint)
			assert.Equal(t, tt.want.APIBind, config.APIBind)
			assert.Equal(t, tt.want.PriceWSBind, config.PriceWSBind)
			assert.Equal(t, tt.want.Mode, config.Mode)
			assert.Equal(t, tt.want.LogLevel, config.LogLevel)
			assert.Equal(t, tt.want.APIKeys, config.APIKeys)

			// 清理环境变量
			clearEnvVars()
		})
	}
}

func TestParseAPIKeys(t *testing.T) {
	tests := []struct {
		name     string
		keysStr  string
		expected []string
	}{
		{
			name:     "空字符串",
			keysStr:  "",
			expected: []string{},
		},
		{
			name:     "单个密钥",
			keysStr:  "key1",
			expected: []string{"key1"},
		},
		{
			name:     "多个密钥",
			keysStr:  "key1,key2,key3",
			expected: []string{"key1", "key2", "key3"},
		},
		{
			name:     "带空格的密钥",
			keysStr:  " key1 , key2 , key3 ",
			expected: []string{"key1", "key2", "key3"},
		},
		{
			name:     "包含空密钥",
			keysStr:  "key1,,key3",
			expected: []string{"key1", "key3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAPIKeys(tt.keysStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidAPIKey(t *testing.T) {
	config := &Config{
		APIKeys: []string{"valid-key-1", "valid-key-2"},
	}

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "有效密钥1",
			key:      "valid-key-1",
			expected: true,
		},
		{
			name:     "有效密钥2",
			key:      "valid-key-2",
			expected: true,
		},
		{
			name:     "无效密钥",
			key:      "invalid-key",
			expected: false,
		},
		{
			name:     "空密钥",
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.IsValidAPIKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDev(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected bool
	}{
		{
			name:     "开发模式",
			mode:     "dev",
			expected: true,
		},
		{
			name:     "生产模式",
			mode:     "prod",
			expected: false,
		},
		{
			name:     "测试模式",
			mode:     "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{Mode: tt.mode}
			result := config.IsDev()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "环境变量存在",
			key:          "TEST_ENV_VAR",
			defaultValue: "default",
			envValue:     "actual",
			expected:     "actual",
		},
		{
			name:         "环境变量不存在",
			key:          "NON_EXISTENT_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理环境变量
			os.Unsetenv(tt.key)

			// 设置环境变量（如果有值）
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)

			// 清理
			os.Unsetenv(tt.key)
		})
	}
}

func TestIsDevnet(t *testing.T) {
	tests := []struct {
		name     string
		rpcURL   string
		expected bool
	}{
		{
			name:     "devnet URL",
			rpcURL:   "https://api.devnet.solana.com",
			expected: true,
		},
		{
			name:     "devnet URL with different case",
			rpcURL:   "https://api.DEVNET.solana.com",
			expected: true,
		},
		{
			name:     "mainnet URL",
			rpcURL:   "https://api.mainnet-beta.solana.com",
			expected: false,
		},
		{
			name:     "testnet URL",
			rpcURL:   "https://api.testnet.solana.com",
			expected: false,
		},
		{
			name:     "localhost URL",
			rpcURL:   "http://localhost:8899",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{RPCUrl: tt.rpcURL}
			result := cfg.IsDevnet()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldUseJupiterMock(t *testing.T) {
	tests := []struct {
		name        string
		mockEnabled bool
		rpcURL      string
		expected    bool
	}{
		{
			name:        "显式启用模拟器",
			mockEnabled: true,
			rpcURL:      "https://api.mainnet-beta.solana.com",
			expected:    true,
		},
		{
			name:        "devnet 环境自动启用",
			mockEnabled: false,
			rpcURL:      "https://api.devnet.solana.com",
			expected:    true,
		},
		{
			name:        "mainnet 环境不启用",
			mockEnabled: false,
			rpcURL:      "https://api.mainnet-beta.solana.com",
			expected:    false,
		},
		{
			name:        "显式启用优先级高于环境检测",
			mockEnabled: true,
			rpcURL:      "https://api.mainnet-beta.solana.com",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				JupiterMockEnabled: tt.mockEnabled,
				RPCUrl:             tt.rpcURL,
			}
			result := cfg.ShouldUseJupiterMock()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// clearEnvVars 清理测试环境变量
func clearEnvVars() {
	envVars := []string{
		"DB_URL", "REDIS_URL", "RPC_URL", "PRIVATE_KEY", "JUPITER_ENDPOINT",
		"API_BIND", "PRICE_WS_BIND", "MODE", "LOG_LEVEL", "API_KEYS",
		"JUPITER_MOCK_ENABLED", "JUPITER_MOCK_SUCCESS_RATE", "JUPITER_MOCK_LATENCY_MS",
		"JUPITER_MOCK_MIN_SLIPPAGE", "JUPITER_MOCK_MAX_SLIPPAGE",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}
