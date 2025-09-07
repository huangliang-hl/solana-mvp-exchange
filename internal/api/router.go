package api

import (
	"encoding/json"
	"net/http"
	"time"

	"solana-limit-order-backend/internal/api/handlers"
	apimiddleware "solana-limit-order-backend/internal/api/middleware"
	"solana-limit-order-backend/internal/config"
	"solana-limit-order-backend/internal/order"
	"solana-limit-order-backend/internal/repository"
	"solana-limit-order-backend/internal/user"
	"solana-limit-order-backend/internal/wallet"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Router API 路由器
type Router struct {
	cfg           *config.Config
	db            *repository.DB
	orderService  order.Service
	userService   user.Service
	walletService wallet.Service
}

// NewRouter 创建新的路由器
func NewRouter(cfg *config.Config, db *repository.DB, orderService order.Service, userService user.Service, walletService wallet.Service) *Router {
	return &Router{
		cfg:           cfg,
		db:            db,
		orderService:  orderService,
		userService:   userService,
		walletService: walletService,
	}
}

// SetupRoutes 设置路由
func (rt *Router) SetupRoutes() http.Handler {
	r := chi.NewRouter()

	// 基础中间件
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS 中间件
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // 生产环境应该限制具体域名
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Api-Key", "Idempotency-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 健康检查（无需认证）
	r.Get("/health", rt.healthCheck)
	r.Get("/", rt.rootHandler)

	// API 路由（需要认证）
	r.Route("/api/v1", func(r chi.Router) {
		// 认证中间件
		r.Use(apimiddleware.Auth(rt.cfg))

		// 幂等性中间件（仅对 POST 请求）
		r.Use(apimiddleware.Idempotency(rt.db))

		// 订单相关路由
		rt.setupOrderRoutes(r)

		// 用户相关路由
		rt.setupUserRoutes(r)

		// 钱包相关路由
		rt.setupWalletRoutes(r)
	})

	return r
}

// setupOrderRoutes 设置订单路由
func (rt *Router) setupOrderRoutes(r chi.Router) {
	ordersHandler := handlers.NewOrdersHandler(rt.orderService)

	r.Route("/orders", func(r chi.Router) {
		r.Post("/", ordersHandler.CreateOrder)       // 创建订单
		r.Get("/", ordersHandler.GetOrders)          // 查询订单列表
		r.Get("/{id}", ordersHandler.GetOrderByID)   // 获取订单详情
		r.Delete("/{id}", ordersHandler.CancelOrder) // 取消订单
	})
}

// setupUserRoutes 设置用户路由
func (rt *Router) setupUserRoutes(r chi.Router) {
	usersHandler := handlers.NewUsersHandler(rt.userService)

	r.Route("/users", func(r chi.Router) {
		r.Post("/", usersHandler.CreateUser)                          // 创建用户
		r.Get("/{id}", usersHandler.GetUser)                          // 获取用户信息
		r.Get("/username/{username}", usersHandler.GetUserByUsername) // 根据用户名获取用户信息
		r.Put("/{id}", usersHandler.UpdateUser)                       // 更新用户信息
		r.Delete("/{id}", usersHandler.DeleteUser)                    // 删除用户
	})
}

// setupWalletRoutes 设置钱包路由
func (rt *Router) setupWalletRoutes(r chi.Router) {
	walletsHandler := handlers.NewWalletsHandler(rt.walletService)

	// 钱包管理路由
	r.Route("/wallets", func(r chi.Router) {
		r.Post("/", walletsHandler.CreateWallet)              // 创建钱包
		r.Get("/{user_id}", walletsHandler.GetWallets)        // 获取用户所有钱包
		r.Get("/{user_id}/{asset}", walletsHandler.GetWallet) // 获取特定钱包
	})

	// 余额查询路由
	r.Route("/balances", func(r chi.Router) {
		r.Get("/{user_id}", walletsHandler.GetBalances)        // 获取用户所有余额
		r.Get("/{user_id}/{asset}", walletsHandler.GetBalance) // 获取特定资产余额
	})

	// 充值提现路由
	r.Post("/deposits", walletsHandler.Deposit)     // 充值
	r.Post("/withdrawals", walletsHandler.Withdraw) // 提现
}

// healthCheck 健康检查
func (rt *Router) healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 检查数据库连接
	dbHealth := "ok"
	if err := rt.db.Health(ctx); err != nil {
		dbHealth = "error: " + err.Error()
	}

	// 构建健康状态响应
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"services": map[string]string{
			"database": dbHealth,
		},
		"version": "1.0.0",
	}

	// 如果有服务不健康，返回 503
	if dbHealth != "ok" {
		health["status"] = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// rootHandler 根路径处理器
func (rt *Router) rootHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service": "Solana Limit Order Backend",
		"version": "1.0.0",
		"status":  "running",
		"endpoints": map[string]interface{}{
			"health": "/health",
			"orders": map[string]string{
				"create": "POST /api/v1/orders",
				"list":   "GET /api/v1/orders",
				"get":    "GET /api/v1/orders/{id}",
				"cancel": "DELETE /api/v1/orders/{id}",
			},
			"users": map[string]string{
				"create":          "POST /api/v1/users",
				"get":             "GET /api/v1/users/{id}",
				"get_by_username": "GET /api/v1/users/username/{username}",
				"update":          "PUT /api/v1/users/{id}",
				"delete":          "DELETE /api/v1/users/{id}",
			},
			"wallets": map[string]string{
				"create": "POST /api/v1/wallets",
				"get":    "GET /api/v1/wallets/{user_id}/{asset}",
				"list":   "GET /api/v1/wallets/{user_id}",
			},
			"balances": map[string]string{
				"get":  "GET /api/v1/balances/{user_id}/{asset}",
				"list": "GET /api/v1/balances/{user_id}",
			},
			"transactions": map[string]string{
				"deposit":  "POST /api/v1/deposits",
				"withdraw": "POST /api/v1/withdrawals",
			},
		},
		"docs": "https://github.com/your-repo/solana-limit-order-backend",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// 已经写入了状态码，无法再次设置错误状态
		// 只记录错误，有意留空
		_ = err
	}
}
