package price

import (
	"context"
	"math"
	"math/rand"
	"time"

	"solana-limit-order-backend/internal/types"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

// Simulator 价格模拟器
type Simulator struct {
	// 配置
	symbol       string
	basePrice    decimal.Decimal
	volatility   float64
	tickInterval time.Duration
	spread       decimal.Decimal
	seed         int64

	// 运行状态
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
	sequenceID int64

	// 随机数生成器
	rng *rand.Rand

	// 广播回调
	broadcastFunc func(*types.PriceTick)
}

// SimulatorConfig 模拟器配置
type SimulatorConfig struct {
	Symbol       string          `json:"symbol"`
	BasePrice    decimal.Decimal `json:"base_price"`
	Volatility   float64         `json:"volatility"`
	TickInterval time.Duration   `json:"tick_interval"`
	Spread       decimal.Decimal `json:"spread"`
	Seed         int64           `json:"seed"`
}

// NewSimulator 创建价格模拟器
func NewSimulator() *Simulator {
	// 默认配置
	config := SimulatorConfig{
		Symbol:       "SOL/USDC",
		BasePrice:    decimal.NewFromFloat(30.0),
		Volatility:   0.02,
		TickInterval: 1 * time.Second,
		Spread:       decimal.NewFromFloat(0.1),
		Seed:         42,
	}

	return NewSimulatorWithConfig(config)
}

// NewSimulatorWithConfig 使用配置创建价格模拟器
func NewSimulatorWithConfig(config SimulatorConfig) *Simulator {
	ctx, cancel := context.WithCancel(context.Background())

	return &Simulator{
		symbol:       config.Symbol,
		basePrice:    config.BasePrice,
		volatility:   config.Volatility,
		tickInterval: config.TickInterval,
		spread:       config.Spread,
		seed:         config.Seed,
		ctx:          ctx,
		cancel:       cancel,
		sequenceID:   1,
		rng:          rand.New(rand.NewSource(config.Seed)),
	}
}

// Start 启动模拟器
func (s *Simulator) Start(broadcastFunc func(*types.PriceTick)) error {
	if s.running {
		return nil
	}

	s.broadcastFunc = broadcastFunc
	s.running = true

	log.Info().
		Str("symbol", s.symbol).
		Str("base_price", s.basePrice.String()).
		Float64("volatility", s.volatility).
		Dur("tick_interval", s.tickInterval).
		Msg("价格模拟器启动")

	go s.run()
	return nil
}

// Stop 停止模拟器
func (s *Simulator) Stop() {
	if !s.running {
		return
	}

	s.running = false
	s.cancel()

	log.Info().
		Str("symbol", s.symbol).
		Msg("价格模拟器停止")
}

// IsRunning 检查是否正在运行
func (s *Simulator) IsRunning() bool {
	return s.running
}

// GetCurrentPrice 获取当前价格
func (s *Simulator) GetCurrentPrice() decimal.Decimal {
	return s.basePrice
}

// run 运行模拟器主循环
func (s *Simulator) run() {
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	currentPrice := s.basePrice

	for {
		select {
		case <-s.ctx.Done():
			return

		case <-ticker.C:
			// 生成下一个价格
			currentPrice = s.generateNextPrice(currentPrice)

			// 计算买卖价
			halfSpread := s.spread.Div(decimal.NewFromInt(2))
			bidPrice := currentPrice.Sub(halfSpread)
			askPrice := currentPrice.Add(halfSpread)

			// 创建价格 tick
			tick := &types.PriceTick{
				SequenceID: s.sequenceID,
				Timestamp:  time.Now().UnixMilli(),
				Symbol:     s.symbol,
				BidPrice:   bidPrice,
				AskPrice:   askPrice,
				MidPrice:   currentPrice,
				Source:     "simulator",
			}

			// 广播价格 tick
			if s.broadcastFunc != nil {
				s.broadcastFunc(tick)
			}

			s.sequenceID++

			// 记录调试日志
			if s.sequenceID%60 == 0 { // 每分钟记录一次
				log.Debug().
					Str("symbol", s.symbol).
					Int64("sequence_id", s.sequenceID).
					Str("mid_price", currentPrice.String()).
					Str("bid_price", bidPrice.String()).
					Str("ask_price", askPrice.String()).
					Msg("价格 tick 生成")
			}
		}
	}
}

// generateNextPrice 生成下一个价格
func (s *Simulator) generateNextPrice(currentPrice decimal.Decimal) decimal.Decimal {
	// 使用几何布朗运动模型
	// P(t+1) = P(t) * exp((μ - σ²/2) * dt + σ * sqrt(dt) * Z)
	// 其中 Z 是标准正态分布随机数

	dt := s.tickInterval.Seconds() / 86400.0 // 转换为天为单位
	mu := 0.0                                // 漂移率（假设为0）
	sigma := s.volatility                    // 波动率

	// 生成标准正态分布随机数
	z := s.rng.NormFloat64()

	// 计算价格变化
	drift := (mu - sigma*sigma/2) * dt
	diffusion := sigma * math.Sqrt(dt) * z
	priceChange := math.Exp(drift + diffusion)

	// 应用价格变化
	newPrice := currentPrice.Mul(decimal.NewFromFloat(priceChange))

	// 确保价格在合理范围内（不低于基础价格的10%，不高于基础价格的10倍）
	minPrice := s.basePrice.Mul(decimal.NewFromFloat(0.1))
	maxPrice := s.basePrice.Mul(decimal.NewFromFloat(10.0))

	if newPrice.LessThan(minPrice) {
		newPrice = minPrice
	} else if newPrice.GreaterThan(maxPrice) {
		newPrice = maxPrice
	}

	return newPrice
}

// SetVolatility 设置波动率
func (s *Simulator) SetVolatility(volatility float64) {
	if volatility < 0 {
		volatility = 0
	}
	if volatility > 1.0 {
		volatility = 1.0
	}
	s.volatility = volatility

	log.Info().
		Str("symbol", s.symbol).
		Float64("volatility", volatility).
		Msg("模拟器波动率已更新")
}

// SetTickInterval 设置tick间隔
func (s *Simulator) SetTickInterval(interval time.Duration) {
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	s.tickInterval = interval

	log.Info().
		Str("symbol", s.symbol).
		Dur("tick_interval", interval).
		Msg("模拟器tick间隔已更新")
}

// GetStats 获取模拟器统计信息
func (s *Simulator) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"symbol":        s.symbol,
		"base_price":    s.basePrice.String(),
		"current_price": s.GetCurrentPrice().String(),
		"volatility":    s.volatility,
		"tick_interval": s.tickInterval.String(),
		"spread":        s.spread.String(),
		"sequence_id":   s.sequenceID,
		"running":       s.running,
		"seed":          s.seed,
	}
}
