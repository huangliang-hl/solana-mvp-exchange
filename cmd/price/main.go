package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"solana-limit-order-backend/internal/config"
	"solana-limit-order-backend/internal/logger"
	"solana-limit-order-backend/internal/price"
	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		// 使用默认日志级别初始化
		logger.Init("info", false)
		log.Fatal().Err(err).Msg("加载配置失败")
	}

	// 使用配置中的日志级别初始化日志
	logger.Init(cfg.LogLevel, cfg.IsDev())

	log.Info().Msg("启动 Solana 限价单价格服务")

	log.Info().
		Str("price_ws_bind", cfg.PriceWSBind).
		Str("mode", cfg.Mode).
		Msg("配置加载完成")

	// 初始化数据库连接（用于可选的价格持久化）
	db, err := repository.NewDB(cfg.DatabaseURL, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("数据库连接失败")
	}
	defer db.Close()

	// 初始化 Redis 队列（用于价格广播）
	queueClient, err := queue.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Redis 连接失败")
	}
	defer queueClient.Close()

	// 创建 WebSocket Hub
	hub := price.NewHub(cfg)
	go hub.Run()

	// 创建价格服务
	priceService := price.NewService(db, queueClient)
	priceService.SetHub(hub)

	// 启动价格模拟器
	basePrice, err := decimal.NewFromString(cfg.PriceSimulatorBasePrice)
	if err != nil {
		log.Fatal().Err(err).Msg("解析模拟器基础价格失败")
	}
	spread, err := decimal.NewFromString(cfg.PriceSimulatorSpread)
	if err != nil {
		log.Fatal().Err(err).Msg("解析模拟器价差失败")
	}

	simulatorConfig := price.SimulatorConfig{
		Symbol:       cfg.PriceSimulatorSymbol,
		BasePrice:    basePrice,
		Volatility:   cfg.PriceSimulatorVolatility,
		TickInterval: time.Duration(cfg.PriceSimulatorTickInterval) * time.Millisecond,
		Spread:       spread,
		Seed:         42,
	}
	simulator := price.NewSimulatorWithConfig(simulatorConfig)

	go func() {
		log.Info().
			Str("symbol", cfg.PriceSimulatorSymbol).
			Str("base_price", cfg.PriceSimulatorBasePrice).
			Float64("volatility", cfg.PriceSimulatorVolatility).
			Int("tick_interval_ms", cfg.PriceSimulatorTickInterval).
			Msg("启动价格模拟器")
		if err := simulator.Start(priceService.Broadcast); err != nil {
			log.Error().Err(err).Msg("价格模拟器启动失败")
		}
	}()

	// 创建路由器
	r := chi.NewRouter()

	// 基础中间件
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS 中间件
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Api-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// WebSocket 路由
	r.HandleFunc("/price", hub.HandleWebSocket)

	// 健康检查
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"status": "ok",
			"service": "price",
			"timestamp": "%s",
			"websocket_endpoint": "/price"
		}`, time.Now().UTC().Format(time.RFC3339))
	})

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         cfg.PriceWSBind,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.HTTPReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTPWriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.HTTPIdleTimeout) * time.Second,
	}

	// 启动服务器
	go func() {
		log.Info().
			Str("addr", cfg.PriceWSBind).
			Msg("价格 WebSocket 服务器启动")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("价格服务器启动失败")
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("收到停止信号，开始优雅关闭...")

	// 停止模拟器
	simulator.Stop()

	// 停止 hub
	hub.Stop()

	// 创建关闭超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭 HTTP 服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("价格服务器关闭失败")
	} else {
		log.Info().Msg("价格服务器已关闭")
	}

	log.Info().Msg("价格服务已停止")
}
