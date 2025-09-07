package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"solana-limit-order-backend/internal/api"
	"solana-limit-order-backend/internal/config"
	"solana-limit-order-backend/internal/ledger"
	"solana-limit-order-backend/internal/logger"
	"solana-limit-order-backend/internal/order"
	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/user"
	"solana-limit-order-backend/internal/wallet"

	"github.com/rs/zerolog/log"
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

	log.Info().Msg("启动 Solana 限价单 API 服务")

	log.Info().
		Str("api_bind", cfg.APIBind).
		Str("mode", cfg.Mode).
		Str("log_level", cfg.LogLevel).
		Msg("配置加载完成")

	// 初始化数据库连接
	db, err := repository.NewDB(cfg.DatabaseURL, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("数据库连接失败")
	}
	defer db.Close()

	// 初始化 Redis 队列
	queueClient, err := queue.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Redis 连接失败")
	}
	defer queueClient.Close()

	// 初始化仓储层
	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)

	// 初始化服务层
	ledgerService := ledger.NewService(db, walletRepo, ledgerRepo)
	userService := user.NewService(db, userRepo)
	walletService := wallet.NewService(db, walletRepo, userRepo, ledgerService)
	orderService := order.NewService(orderRepo, userRepo, ledgerService)

	// 创建 API 路由器
	router := api.NewRouter(cfg, db, orderService, userService, walletService)
	handler := router.SetupRoutes()

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         cfg.APIBind,
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.HTTPReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTPWriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.HTTPIdleTimeout) * time.Second,
	}

	// 启动服务器
	go func() {
		log.Info().
			Str("addr", cfg.APIBind).
			Msg("HTTP API 服务器启动")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP 服务器启动失败")
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("收到停止信号，开始优雅关闭...")

	// 创建关闭超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭 HTTP 服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP 服务器关闭失败")
	} else {
		log.Info().Msg("HTTP 服务器已关闭")
	}

	log.Info().Msg("API 服务已停止")
}
