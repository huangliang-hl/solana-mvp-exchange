package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init 初始化日志配置 (别名)
func Init(level string, isDev bool) {
	Setup(level, isDev)
}

// Setup 初始化日志配置
func Setup(level string, isDev bool) {
	// 设置日志级别
	logLevel := parseLogLevel(level)
	zerolog.SetGlobalLevel(logLevel)

	// 开发模式使用人类可读格式，生产模式使用 JSON 格式
	if isDev {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// 添加调用者信息
	log.Logger = log.Logger.With().Caller().Logger()
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithOrderID 添加订单 ID 到日志上下文
func WithOrderID(orderID string) *zerolog.Event {
	return log.Info().Str("order_id", orderID)
}

// WithUserID 添加用户 ID 到日志上下文
func WithUserID(userID string) *zerolog.Event {
	return log.Info().Str("user_id", userID)
}

// WithTxSig 添加交易签名到日志上下文
func WithTxSig(txSig string) *zerolog.Event {
	return log.Info().Str("tx_sig", txSig)
}

// WithAttemptID 添加尝试 ID 到日志上下文
func WithAttemptID(attemptID string) *zerolog.Event {
	return log.Info().Str("attempt_id", attemptID)
}

// WithEvent 添加事件类型到日志上下文
func WithEvent(event string) *zerolog.Event {
	return log.Info().Str("event", event)
}

// Error 记录错误日志
func Error(err error) *zerolog.Event {
	return log.Error().Err(err)
}

// Info 记录信息日志
func Info() *zerolog.Event {
	return log.Info()
}

// Debug 记录调试日志
func Debug() *zerolog.Event {
	return log.Debug()
}

// Warn 记录警告日志
func Warn() *zerolog.Event {
	return log.Warn()
}
