package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config 应用配置结构
type Config struct {
	// 数据库配置
	DatabaseURL       string
	DBMaxConns        int
	DBMinConns        int
	DBMaxConnLifetime int // 小时
	DBMaxConnIdleTime int // 分钟

	// Redis 配置
	RedisURL string

	// Solana 配置
	RPCUrl     string
	PrivateKey string

	// Jupiter 配置
	JupiterEndpoint string

	// Jupiter 模拟器配置
	JupiterMockEnabled     bool
	JupiterMockSuccessRate float64
	JupiterMockLatencyMs   int
	JupiterMockMinSlippage float64
	JupiterMockMaxSlippage float64

	// 服务绑定地址
	APIBind     string
	PriceWSBind string

	// HTTP 服务器配置
	HTTPReadTimeout  int // 秒
	HTTPWriteTimeout int // 秒
	HTTPIdleTimeout  int // 秒

	// 执行器配置
	WorkerCount           int
	WorkerConsumerTimeout int // 秒
	OrderExecutionTimeout int // 秒
	HealthCheckInterval   int // 秒

	// 价格模拟器配置
	PriceSimulatorSymbol       string
	PriceSimulatorBasePrice    string
	PriceSimulatorVolatility   float64
	PriceSimulatorTickInterval int // 毫秒
	PriceSimulatorSpread       string

	// WebSocket 配置
	WSReadTimeout  int // 秒
	WSWriteTimeout int // 秒
	WSPingInterval int // 秒

	// 运行模式
	Mode string

	// 日志级别
	LogLevel string

	// API 密钥列表
	APIKeys []string
}

// Load 加载配置
func Load() (*Config, error) {
	// 尝试加载 .env 文件，如果不存在则忽略
	// 支持通过 ENV_FILE 环境变量指定配置文件
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)

	config := &Config{
		// 数据库配置
		DatabaseURL:       getEnv("DB_URL", "postgres://postgres:password@localhost:5432/solana_limit_order?sslmode=disable"),
		DBMaxConns:        getEnvInt("DB_MAX_CONNS", 20),
		DBMinConns:        getEnvInt("DB_MIN_CONNS", 5),
		DBMaxConnLifetime: getEnvInt("DB_MAX_CONN_LIFETIME", 1),   // 小时
		DBMaxConnIdleTime: getEnvInt("DB_MAX_CONN_IDLE_TIME", 30), // 分钟

		// Redis 配置
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379/0"),

		// Solana 配置
		RPCUrl:     getEnv("RPC_URL", "https://api.devnet.solana.com"),
		PrivateKey: getEnv("PRIVATE_KEY", ""),

		// Jupiter 配置
		JupiterEndpoint: getEnv("JUPITER_ENDPOINT", "https://lite-api.jup.ag/swap/v1/"),

		// Jupiter 模拟器配置
		JupiterMockEnabled:     getEnvBool("JUPITER_MOCK_ENABLED", false),
		JupiterMockSuccessRate: getEnvFloat("JUPITER_MOCK_SUCCESS_RATE", 0.95),
		JupiterMockLatencyMs:   getEnvInt("JUPITER_MOCK_LATENCY_MS", 200),
		JupiterMockMinSlippage: getEnvFloat("JUPITER_MOCK_MIN_SLIPPAGE", 0.1),
		JupiterMockMaxSlippage: getEnvFloat("JUPITER_MOCK_MAX_SLIPPAGE", 2.0),

		// 服务绑定地址
		APIBind:     getEnv("API_BIND", "0.0.0.0:8080"),
		PriceWSBind: getEnv("PRICE_WS_BIND", "0.0.0.0:8081"),

		// HTTP 服务器配置
		HTTPReadTimeout:  getEnvInt("HTTP_READ_TIMEOUT", 15),  // 秒
		HTTPWriteTimeout: getEnvInt("HTTP_WRITE_TIMEOUT", 15), // 秒
		HTTPIdleTimeout:  getEnvInt("HTTP_IDLE_TIMEOUT", 60),  // 秒

		// 执行器配置
		WorkerCount:           getEnvInt("WORKER_COUNT", 3),
		WorkerConsumerTimeout: getEnvInt("WORKER_CONSUMER_TIMEOUT", 5),  // 秒
		OrderExecutionTimeout: getEnvInt("ORDER_EXECUTION_TIMEOUT", 30), // 秒
		HealthCheckInterval:   getEnvInt("HEALTH_CHECK_INTERVAL", 30),   // 秒

		// 价格模拟器配置
		PriceSimulatorSymbol:       getEnv("PRICE_SIMULATOR_SYMBOL", "SOL/USDC"),
		PriceSimulatorBasePrice:    getEnv("PRICE_SIMULATOR_BASE_PRICE", "30.0"),
		PriceSimulatorVolatility:   getEnvFloat("PRICE_SIMULATOR_VOLATILITY", 0.02),
		PriceSimulatorTickInterval: getEnvInt("PRICE_SIMULATOR_TICK_INTERVAL", 1000), // 毫秒
		PriceSimulatorSpread:       getEnv("PRICE_SIMULATOR_SPREAD", "0.1"),

		// WebSocket 配置
		WSReadTimeout:  getEnvInt("WS_READ_TIMEOUT", 120), // 秒 - 增加读取超时时间
		WSWriteTimeout: getEnvInt("WS_WRITE_TIMEOUT", 30), // 秒 - 增加写入超时时间
		WSPingInterval: getEnvInt("WS_PING_INTERVAL", 25), // 秒 - 减少ping间隔以保持连接活跃

		// 运行模式和日志
		Mode:     getEnv("MODE", "dev"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		APIKeys:  parseAPIKeys(getEnv("API_KEYS", "test-key-1")),
	}

	return config, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数环境变量，如果不存在或解析失败则返回默认值
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvFloat 获取浮点数环境变量，如果不存在或解析失败则返回默认值
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvBool 获取布尔值环境变量，如果不存在或解析失败则返回默认值
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// parseAPIKeys 解析 API 密钥列表
func parseAPIKeys(keysStr string) []string {
	if keysStr == "" {
		return []string{}
	}

	keys := strings.Split(keysStr, ",")
	result := make([]string, 0, len(keys))

	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			result = append(result, key)
		}
	}

	return result
}

// IsValidAPIKey 检查 API 密钥是否有效
func (c *Config) IsValidAPIKey(key string) bool {
	for _, validKey := range c.APIKeys {
		if validKey == key {
			return true
		}
	}
	return false
}

// IsDev 是否为开发模式
func (c *Config) IsDev() bool {
	return c.Mode == "dev"
}

// IsDevnet 是否为 devnet 环境
func (c *Config) IsDevnet() bool {
	return strings.Contains(strings.ToLower(c.RPCUrl), "devnet")
}

// ShouldUseJupiterMock 是否应该使用 Jupiter 模拟器
func (c *Config) ShouldUseJupiterMock() bool {
	// 如果显式启用了模拟器，则使用
	if c.JupiterMockEnabled {
		return true
	}

	// 如果是 devnet 环境，自动启用模拟器
	return c.IsDevnet()
}
