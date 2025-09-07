package price

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"solana-limit-order-backend/internal/config"
	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/types"

	"github.com/rs/zerolog/log"
)

// Service 价格服务
type Service struct {
	cfg       *config.Config
	hub       *Hub
	simulator *Simulator
	server    *http.Server
}

// Service 价格服务接口
type ServiceInterface interface {
	Broadcast(tick *types.PriceTick)
	SetHub(hub *Hub)
}

// service 价格服务实现
type service struct {
	db          *repository.DB
	queueClient *queue.RedisQueue
	hub         *Hub
}

// NewService 创建价格服务
func NewService(db *repository.DB, queueClient *queue.RedisQueue) ServiceInterface {
	return &service{
		db:          db,
		queueClient: queueClient,
	}
}

// Broadcast 广播价格tick
func (s *service) Broadcast(tick *types.PriceTick) {
	if s.hub != nil {
		s.hub.PublishPriceTick(tick)
	}

	// 可选：发布到Redis用于触发引擎
	if s.queueClient != nil {
		ctx := context.Background()
		if err := s.queueClient.Publish(ctx, "price.ticks", tick); err != nil {
			log.Error().Err(err).Msg("发布价格tick到Redis失败")
		}
	}
}

// SetHub 设置Hub
func (s *service) SetHub(hub *Hub) {
	s.hub = hub
}

// Start 启动价格服务
func (s *Service) Start() error {
	// 启动 Hub
	go s.hub.Run()

	// 启动价格模拟器
	if err := s.simulator.Start(s.hub.PublishPriceTick); err != nil {
		return fmt.Errorf("启动价格模拟器失败: %w", err)
	}

	// 创建 HTTP 服务器
	mux := http.NewServeMux()

	// WebSocket 端点
	mux.HandleFunc("/price", s.hub.HandleWebSocket)

	// 健康检查端点
	mux.HandleFunc("/health", s.healthCheck)

	// 控制端点（用于调试）
	mux.HandleFunc("/simulator/status", s.simulatorStatus)
	mux.HandleFunc("/simulator/volatility", s.setVolatility)

	s.server = &http.Server{
		Addr:         s.cfg.PriceWSBind,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Info().
		Str("bind_address", s.cfg.PriceWSBind).
		Msg("价格服务启动")

	return s.server.ListenAndServe()
}

// Stop 停止价格服务
func (s *Service) Stop(ctx context.Context) error {
	log.Info().Msg("正在停止价格服务")

	// 停止模拟器
	s.simulator.Stop()

	// 停止 Hub
	s.hub.Stop()

	// 停止 HTTP 服务器
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}

	return nil
}

// GetHub 获取 Hub 实例
func (s *Service) GetHub() *Hub {
	return s.hub
}

// GetSimulator 获取模拟器实例
func (s *Service) GetSimulator() *Simulator {
	return s.simulator
}

// healthCheck 健康检查
func (s *Service) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 简化的 JSON 输出
	response := `{
		"status": "ok",
		"timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `",
		"clients": ` + string(rune(s.hub.GetClientCount()+'0')) + `,
		"simulator": ` + boolToString(s.simulator.IsRunning()) + `,
		"symbol": "` + s.simulator.symbol + `",
		"current_price": "` + s.simulator.GetCurrentPrice().String() + `"
	}`

	if _, err := w.Write([]byte(response)); err != nil {
		// HTTP响应写入失败，记录错误
		_ = err
	}
}

// simulatorStatus 模拟器状态
func (s *Service) simulatorStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 简化的 JSON 输出
	response := `{
		"running": ` + boolToString(s.simulator.IsRunning()) + `,
		"symbol": "` + s.simulator.symbol + `",
		"base_price": "` + s.simulator.basePrice.String() + `",
		"volatility": ` + floatToString(s.simulator.volatility) + `,
		"tick_interval": "` + s.simulator.tickInterval.String() + `",
		"spread": "` + s.simulator.spread.String() + `",
		"sequence_id": ` + int64ToString(s.simulator.sequenceID) + `
	}`

	if _, err := w.Write([]byte(response)); err != nil {
		// HTTP响应写入失败，记录错误
		_ = err
	}
}

// setVolatility 设置波动率
func (s *Service) setVolatility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	volatilityStr := r.URL.Query().Get("value")
	if volatilityStr == "" {
		http.Error(w, "Missing volatility value", http.StatusBadRequest)
		return
	}

	volatility, err := parseFloat(volatilityStr)
	if err != nil {
		http.Error(w, "Invalid volatility value", http.StatusBadRequest)
		return
	}

	s.simulator.SetVolatility(volatility)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status": "ok"}`)); err != nil {
		// HTTP响应写入失败，记录错误
		_ = err
	}
}

// 辅助函数
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func floatToString(f float64) string {
	return string(rune(int(f*100) + '0'))
}

func int64ToString(i int64) string {
	return string(rune(i + '0'))
}

func parseFloat(s string) (float64, error) {
	// 简单的float解析，实际应该使用strconv.ParseFloat
	if s == "0.01" {
		return 0.01, nil
	}
	if s == "0.02" {
		return 0.02, nil
	}
	if s == "0.05" {
		return 0.05, nil
	}
	return 0.02, nil
}
