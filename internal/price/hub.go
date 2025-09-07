package price

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"solana-limit-order-backend/internal/config"
	"solana-limit-order-backend/internal/types"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// Hub WebSocket 连接管理中心
type Hub struct {
	// 客户端连接
	clients map[*Client]bool

	// 注册新客户端
	register chan *Client

	// 注销客户端
	unregister chan *Client

	// 广播消息到所有客户端
	broadcast chan []byte

	// 价格 tick 通道
	priceTicks chan *types.PriceTick

	// 互斥锁
	mutex sync.RWMutex

	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc

	// WebSocket 配置
	config *config.Config
}

// Client WebSocket 客户端
type Client struct {
	// WebSocket 连接
	conn *websocket.Conn

	// 发送消息缓冲通道
	send chan []byte

	// Hub 引用
	hub *Hub

	// 客户端 ID
	id string
}

// NewHub 创建新的 Hub
func NewHub(cfg *config.Config) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		priceTicks: make(chan *types.PriceTick, 1000),
		ctx:        ctx,
		cancel:     cancel,
		config:     cfg,
	}
}

// Run 运行 Hub
func (h *Hub) Run() {
	defer h.cancel()

	log.Info().Msg("价格 Hub 开始运行")

	for {
		select {
		case <-h.ctx.Done():
			log.Info().Msg("价格 Hub 停止运行")
			return

		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

			log.Info().
				Str("client_id", client.id).
				Int("total_clients", len(h.clients)).
				Msg("新客户端连接")

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()

			log.Info().
				Str("client_id", client.id).
				Int("total_clients", len(h.clients)).
				Msg("客户端断开连接")

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// 客户端发送缓冲区满，关闭连接
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mutex.RUnlock()

		case priceTick := <-h.priceTicks:
			// 将价格 tick 转换为 JSON 并广播
			if data, err := json.Marshal(priceTick); err == nil {
				select {
				case h.broadcast <- data:
				default:
					log.Warn().Msg("广播通道满，丢弃价格 tick")
				}
			} else {
				log.Error().Err(err).Msg("序列化价格 tick 失败")
			}
		}
	}
}

// PublishPriceTick 发布价格 tick
func (h *Hub) PublishPriceTick(tick *types.PriceTick) {
	select {
	case h.priceTicks <- tick:
	default:
		log.Warn().Msg("价格 tick 通道满，丢弃 tick")
	}
}

// GetPriceTickChannel 获取价格 tick 通道（只读）
func (h *Hub) GetPriceTickChannel() <-chan *types.PriceTick {
	return h.priceTicks
}

// GetClientCount 获取客户端数量
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// Stop 停止 Hub
func (h *Hub) Stop() {
	h.cancel()
}

// HandleWebSocket 处理 WebSocket 连接
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// 在生产环境中应该验证Origin
			return true
		},
		ReadBufferSize:   2048, // 增加缓冲区大小
		WriteBufferSize:  2048,
		HandshakeTimeout: 10 * time.Second, // 添加握手超时
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Msg("WebSocket升级失败")
		return
	}

	// 设置连接参数
	conn.SetReadLimit(2048) // 增加读取限制

	// 生成客户端ID
	clientID := generateClientID()

	log.Info().
		Str("client_id", clientID).
		Str("remote_addr", r.RemoteAddr).
		Msg("新的WebSocket连接建立")

	// 创建客户端
	client := &Client{
		conn: conn,
		send: make(chan []byte, 512), // 增加发送缓冲区
		hub:  h,
		id:   clientID,
	}

	// 注册客户端
	h.register <- client

	// 启动客户端处理协程
	go client.writePump()
	go client.readPump()
}

// generateClientID 生成客户端ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// writePump 客户端写数据泵
func (c *Client) writePump() {
	pingInterval := time.Duration(c.hub.config.WSPingInterval) * time.Second
	writeTimeout := time.Duration(c.hub.config.WSWriteTimeout) * time.Second

	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					// 连接可能已关闭，忽略错误
					_ = err
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}

			// 将队列中的其他消息一起发送
			n := len(c.send)
			for i := 0; i < n; i++ {
				if _, err := w.Write([]byte{'\n'}); err != nil {
					return
				}
				if _, err := w.Write(<-c.send); err != nil {
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 客户端读数据泵
func (c *Client) readPump() {
	readTimeout := time.Duration(c.hub.config.WSReadTimeout) * time.Second

	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	if err := c.conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			return err
		}
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// 检查关闭类型并记录相应日志
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				// 正常关闭 (1000) - 记录为 Info
				log.Info().
					Str("client_id", c.id).
					Str("close_code", "1000").
					Msg("WebSocket连接正常关闭")
			} else if websocket.IsCloseError(err, websocket.CloseGoingAway) {
				// 客户端离开 (1001) - 记录为 Info
				log.Info().
					Str("client_id", c.id).
					Str("close_code", "1001").
					Msg("WebSocket客户端离开")
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				// 连接关闭 - 记录为 Info
				log.Info().
					Str("client_id", c.id).
					Str("close_reason", err.Error()).
					Msg("WebSocket连接关闭")
			} else {
				// 其他关闭情况 - 记录为 Debug
				log.Debug().
					Str("client_id", c.id).
					Str("close_reason", err.Error()).
					Msg("WebSocket连接关闭")
			}
			break
		}

		// 收到消息时重置读取超时
		if err := c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.hub.config.WSReadTimeout) * time.Second)); err != nil {
			log.Error().Err(err).Msg("设置读取超时失败")
			break
		}
	}
}
