package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"solana-limit-order-backend/internal/config"
	"solana-limit-order-backend/internal/executor"
	"solana-limit-order-backend/internal/ledger"
	"solana-limit-order-backend/internal/logger"
	"solana-limit-order-backend/internal/order"
	"solana-limit-order-backend/internal/queue"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/trigger"

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

	log.Info().Msg("启动 Solana 限价单执行器服务")

	log.Info().
		Str("rpc_url", cfg.RPCUrl).
		Str("jupiter_endpoint", cfg.JupiterEndpoint).
		Bool("jupiter_mock_enabled", cfg.ShouldUseJupiterMock()).
		Str("mode", cfg.Mode).
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

	// 初始化队列和消费组
	queueManager := queue.NewQueueManager(queueClient)
	if err := queueManager.InitializeStreams(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("初始化队列失败")
	}

	// 初始化仓储层
	userRepo := repository.NewUserRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)

	// 初始化服务层
	ledgerService := ledger.NewService(db, walletRepo, ledgerRepo)
	_ = order.NewService(orderRepo, userRepo, ledgerService) // 这里暂时不使用

	// 初始化 Jupiter 客户端（根据配置选择真实客户端或模拟器）
	jupiterClient := executor.CreateJupiterClient(cfg)

	// 初始化 Solana 客户端
	solanaClient, err := executor.NewSolanaClient(cfg.RPCUrl, cfg.PrivateKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Solana 客户端初始化失败")
	}

	// 创建执行器服务
	executorService := executor.NewService(
		cfg,
		queueClient,
		orderRepo,
		ledgerService,
		jupiterClient,
		solanaClient,
	)

	// 创建触发引擎
	triggerEngine := trigger.NewEngine(
		queueClient,
		orderRepo,
	)

	// 创建上下文用于控制服务生命周期
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动触发引擎
	go func() {
		log.Info().Msg("启动触发引擎")
		if err := triggerEngine.Start(ctx); err != nil {
			log.Error().Err(err).Msg("触发引擎启动失败")
		}
	}()

	// 启动执行器工作器
	workerCount := cfg.WorkerCount // 从配置文件获取工作器数量
	for i := 0; i < workerCount; i++ {
		go func(workerID int) {
			log.Info().Int("worker_id", workerID).Msg("启动执行器工作器")
			if err := executorService.StartWorker(ctx, workerID); err != nil {
				log.Error().
					Err(err).
					Int("worker_id", workerID).
					Msg("执行器工作器启动失败")
			}
		}(i)
	}

	// 启动订单恢复服务（系统重启后恢复未完成的订单）
	go func() {
		log.Info().Msg("启动订单恢复服务")
		if err := executorService.RecoverPendingOrders(ctx); err != nil {
			log.Error().Err(err).Msg("订单恢复失败")
		}
	}()

	// 启动定期健康检查
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.HealthCheckInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 检查数据库健康状态
				if err := db.Health(ctx); err != nil {
					log.Error().Err(err).Msg("数据库健康检查失败")
				}

				// 检查 Redis 健康状态
				if err := queueClient.Health(ctx); err != nil {
					log.Error().Err(err).Msg("Redis 健康检查失败")
				}

				log.Debug().Msg("健康检查完成")
			}
		}
	}()

	log.Info().Msg("所有服务已启动，等待停止信号...")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("收到停止信号，开始优雅关闭...")

	// 取消上下文，停止所有服务
	cancel()

	// 等待一段时间让服务完成当前任务
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 等待关闭完成或超时
	done := make(chan struct{})
	go func() {
		// 这里可以添加更精细的关闭逻辑
		time.Sleep(5 * time.Second)
		close(done)
	}()

	select {
	case <-shutdownCtx.Done():
		log.Warn().Msg("强制关闭：超时")
	case <-done:
		log.Info().Msg("优雅关闭完成")
	}

	log.Info().Msg("执行器服务已停止")
}
