package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderSide 订单方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusOpen       OrderStatus = "open"
	OrderStatusTriggered  OrderStatus = "triggered"
	OrderStatusSubmitting OrderStatus = "submitting"
	OrderStatusFilled     OrderStatus = "filled"
	OrderStatusFailed     OrderStatus = "failed"
	OrderStatusCanceled   OrderStatus = "canceled"
)

// TriggerOp 触发操作
type TriggerOp string

const (
	TriggerOpGTE TriggerOp = "gte" // 大于等于
	TriggerOpLTE TriggerOp = "lte" // 小于等于
)

// LedgerEntryType 账本条目类型
type LedgerEntryType string

const (
	LedgerEntryTypeLockFunds    LedgerEntryType = "lock_funds"
	LedgerEntryTypeReleaseFunds LedgerEntryType = "release_funds"
	LedgerEntryTypeFilled       LedgerEntryType = "filled"
	LedgerEntryTypeCompensate   LedgerEntryType = "compensate"
)

// TxAttemptStatus 交易尝试状态
type TxAttemptStatus string

const (
	TxAttemptStatusPending   TxAttemptStatus = "pending"
	TxAttemptStatusSubmitted TxAttemptStatus = "submitted"
	TxAttemptStatusConfirmed TxAttemptStatus = "confirmed"
	TxAttemptStatusFailed    TxAttemptStatus = "failed"
)

// User 用户模型
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Wallet 钱包模型
type Wallet struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	UserID           uuid.UUID       `json:"user_id" db:"user_id"`
	Asset            string          `json:"asset" db:"asset"`
	AvailableDecimal decimal.Decimal `json:"available_decimal" db:"available_decimal"`
	LockedDecimal    decimal.Decimal `json:"locked_decimal" db:"locked_decimal"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// Order 订单模型
type Order struct {
	ID                  uuid.UUID       `json:"id" db:"id"`
	UserID              uuid.UUID       `json:"user_id" db:"user_id"`
	Base                string          `json:"base" db:"base"`
	Quote               string          `json:"quote" db:"quote"`
	Size                decimal.Decimal `json:"size" db:"size"`
	Side                OrderSide       `json:"side" db:"side"`
	TriggerPrice        decimal.Decimal `json:"trigger_price" db:"trigger_price"`
	TriggerOp           TriggerOp       `json:"trigger_op" db:"trigger_op"`
	Status              OrderStatus     `json:"status" db:"status"`
	MaxSlippagePct      decimal.Decimal `json:"max_slippage_pct" db:"max_slippage_pct"`
	PriorityFeeLamports int64           `json:"priority_fee_lamports" db:"priority_fee_lamports"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`
	TriggeredAt         *time.Time      `json:"triggered_at,omitempty" db:"triggered_at"`
	SubmittedAt         *time.Time      `json:"submitted_at,omitempty" db:"submitted_at"`
	TxSig               *string         `json:"tx_sig,omitempty" db:"tx_sig"`
	ErrorMessage        *string         `json:"error_message,omitempty" db:"error_message"`
	Version             int             `json:"version" db:"version"`
}

// LedgerEntry 账本条目模型
type LedgerEntry struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	UserID       uuid.UUID       `json:"user_id" db:"user_id"`
	Asset        string          `json:"asset" db:"asset"`
	Delta        decimal.Decimal `json:"delta" db:"delta"`
	BalanceAfter decimal.Decimal `json:"balance_after" db:"balance_after"`
	Type         LedgerEntryType `json:"type" db:"type"`
	RefID        uuid.UUID       `json:"ref_id" db:"ref_id"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// TxAttempt 交易尝试模型
type TxAttempt struct {
	AttemptID    uuid.UUID       `json:"attempt_id" db:"attempt_id"`
	OrderID      uuid.UUID       `json:"order_id" db:"order_id"`
	Params       string          `json:"params" db:"params"` // JSON 字符串
	Status       TxAttemptStatus `json:"status" db:"status"`
	TxSig        *string         `json:"tx_sig,omitempty" db:"tx_sig"`
	ErrorMessage *string         `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// PriceTick 价格 tick 模型
type PriceTick struct {
	SequenceID int64           `json:"sequence_id"`
	Timestamp  int64           `json:"timestamp"`
	Symbol     string          `json:"symbol"`
	BidPrice   decimal.Decimal `json:"bid_price"`
	AskPrice   decimal.Decimal `json:"ask_price"`
	MidPrice   decimal.Decimal `json:"mid_price"`
	Source     string          `json:"source"`
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	UserID              string `json:"user_id" validate:"required"`
	Base                string `json:"base" validate:"required"`
	Quote               string `json:"quote" validate:"required"`
	Size                string `json:"size" validate:"required"`
	Side                string `json:"side" validate:"required,oneof=buy sell"`
	TriggerPrice        string `json:"trigger_price" validate:"required"`
	TriggerOp           string `json:"trigger_op" validate:"required,oneof=gte lte"`
	MaxSlippagePct      string `json:"max_slippage_pct" validate:"required"`
	PriorityFeeLamports int64  `json:"priority_fee_lamports"`
	IdempotencyKey      string `json:"idempotency_key" validate:"required"`
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	OrderID           string            `json:"order_id"`
	Status            string            `json:"status"`
	AvailableBalances map[string]string `json:"available_balances"`
}

// CancelOrderResponse 取消订单响应
type CancelOrderResponse struct {
	OrderID           string            `json:"order_id"`
	Status            string            `json:"status"`
	AvailableBalances map[string]string `json:"available_balances"`
	LockedBalances    map[string]string `json:"locked_balances"`
}

// OrderQueryResponse 订单查询响应
type OrderQueryResponse struct {
	Orders []OrderInfo `json:"orders"`
}

// QueryOrdersResponse 查询订单响应
type QueryOrdersResponse = OrderQueryResponse

// ExecuteOrderTask 执行订单任务
type ExecuteOrderTask struct {
	OrderID             string `json:"order_id"`
	Timestamp           int64  `json:"timestamp"`
	Price               string `json:"price"`
	Side                string `json:"side"`
	MaxSlippagePct      string `json:"max_slippage_pct"`
	PriorityFeeLamports int64  `json:"priority_fee_lamports"`
	IdempotencyKey      string `json:"idempotency_key"`
}

// OrderInfo 订单信息
type OrderInfo struct {
	OrderID          string  `json:"order_id"`
	BaseAsset        string  `json:"base_asset"`
	QuoteAsset       string  `json:"quote_asset"`
	Size             string  `json:"size"`
	Side             string  `json:"side"`
	TriggerPrice     string  `json:"trigger_price"`
	TriggerCondition string  `json:"trigger_condition"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	TriggeredAt      *string `json:"triggered_at,omitempty"`
	ErrorMessage     *string `json:"error_message,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// ExecuteOrderMessage 执行订单消息
type ExecuteOrderMessage struct {
	OrderID             string `json:"order_id"`
	UserID              string `json:"user_id"`
	MaxSlippagePct      string `json:"max_slippage_pct"`
	PriorityFeeLamports int64  `json:"priority_fee_lamports"`
	IdempotencyKey      string `json:"idempotency_key"`
	Timestamp           int64  `json:"timestamp"`
}

// TriggerCheckMessage 触发检查消息
type TriggerCheckMessage struct {
	PriceTick PriceTick `json:"price_tick"`
}

// BalanceSnapshot 余额快照
type BalanceSnapshot struct {
	Available map[string]decimal.Decimal `json:"available"`
	Locked    map[string]decimal.Decimal `json:"locked"`
}

// 用户管理相关结构体

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username       string `json:"username" validate:"required,min=3,max=50"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

// CreateUserResponse 创建用户响应
type CreateUserResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// GetUserResponse 获取用户响应
type GetUserResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
}

// UpdateUserResponse 更新用户响应
type UpdateUserResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 钱包管理相关结构体

// CreateWalletRequest 创建钱包请求
type CreateWalletRequest struct {
	UserID         string `json:"user_id" validate:"required"`
	Asset          string `json:"asset" validate:"required"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

// CreateWalletResponse 创建钱包响应
type CreateWalletResponse struct {
	WalletID         string          `json:"wallet_id"`
	UserID           string          `json:"user_id"`
	Asset            string          `json:"asset"`
	AvailableBalance decimal.Decimal `json:"available_balance"`
	LockedBalance    decimal.Decimal `json:"locked_balance"`
	CreatedAt        time.Time       `json:"created_at"`
}

// GetWalletResponse 获取钱包响应
type GetWalletResponse struct {
	WalletID         string          `json:"wallet_id"`
	UserID           string          `json:"user_id"`
	Asset            string          `json:"asset"`
	AvailableBalance decimal.Decimal `json:"available_balance"`
	LockedBalance    decimal.Decimal `json:"locked_balance"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// GetWalletsResponse 获取用户所有钱包响应
type GetWalletsResponse struct {
	Wallets []WalletInfo `json:"wallets"`
}

// WalletInfo 钱包信息
type WalletInfo struct {
	WalletID         string `json:"wallet_id"`
	Asset            string `json:"asset"`
	AvailableBalance string `json:"available_balance"`
	LockedBalance    string `json:"locked_balance"`
	UpdatedAt        string `json:"updated_at"`
}

// DepositRequest 充值请求
type DepositRequest struct {
	UserID         string `json:"user_id" validate:"required"`
	Asset          string `json:"asset" validate:"required"`
	Amount         string `json:"amount" validate:"required"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

// DepositResponse 充值响应
type DepositResponse struct {
	TransactionID    string          `json:"transaction_id"`
	UserID           string          `json:"user_id"`
	Asset            string          `json:"asset"`
	Amount           decimal.Decimal `json:"amount"`
	AvailableBalance decimal.Decimal `json:"available_balance"`
	LockedBalance    decimal.Decimal `json:"locked_balance"`
	CreatedAt        time.Time       `json:"created_at"`
}

// WithdrawRequest 提现请求
type WithdrawRequest struct {
	UserID         string `json:"user_id" validate:"required"`
	Asset          string `json:"asset" validate:"required"`
	Amount         string `json:"amount" validate:"required"`
	ToAddress      string `json:"to_address" validate:"required"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

// WithdrawResponse 提现响应
type WithdrawResponse struct {
	TransactionID    string          `json:"transaction_id"`
	UserID           string          `json:"user_id"`
	Asset            string          `json:"asset"`
	Amount           decimal.Decimal `json:"amount"`
	ToAddress        string          `json:"to_address"`
	AvailableBalance decimal.Decimal `json:"available_balance"`
	LockedBalance    decimal.Decimal `json:"locked_balance"`
	Status           string          `json:"status"`
	CreatedAt        time.Time       `json:"created_at"`
}

// GetBalanceResponse 获取余额响应
type GetBalanceResponse struct {
	UserID           string          `json:"user_id"`
	Asset            string          `json:"asset"`
	AvailableBalance decimal.Decimal `json:"available_balance"`
	LockedBalance    decimal.Decimal `json:"locked_balance"`
	TotalBalance     decimal.Decimal `json:"total_balance"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// GetBalancesResponse 获取所有余额响应
type GetBalancesResponse struct {
	UserID   string        `json:"user_id"`
	Balances []BalanceInfo `json:"balances"`
}

// BalanceInfo 余额信息
type BalanceInfo struct {
	Asset            string `json:"asset"`
	AvailableBalance string `json:"available_balance"`
	LockedBalance    string `json:"locked_balance"`
	TotalBalance     string `json:"total_balance"`
	UpdatedAt        string `json:"updated_at"`
}
