package ws

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"pbx_back_end/internal/handler"
)

// FrontendServer 管理前端WebSocket连接
type FrontendServer struct {
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	conn     *websocket.Conn
	llm      *handler.LLMHandler
}

func NewFrontendServer() *FrontendServer {
	return &FrontendServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // shane: 允许跨域
		}, // http的升级
		clients: make(map[*websocket.Conn]bool),
		//llm:     llm,
	}
}

// Start shane: http 升级为 websocket
func (s *FrontendServer) Start(r *gin.Engine, port string) {
	// shane: 使用gin处理WebSocket连接
	r.GET("/ws", func(c *gin.Context) {
		s.handleWebSocket(c.Writer, c.Request) // shane: http请求升级为WebSocket连接
	})
	go func() {
		if err := r.Run(":" + port); err != nil {
			log.Println("Connection failed:", err)
		}
	}() // shane: 开协程防止阻塞
	log.Printf("Connected to the front end! Serve on %s", port)
}

// shane: 处理前端WebSocket连接
func (s *FrontendServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("升级连接失败:", err)
		return
	}
	defer func() {
		conn.Close()
		delete(s.clients, conn)
		log.Println("WebSocket Connection closed")
	}()
	s.clients[conn] = true // shane: 设置已经连接（状态信息）
	s.conn = conn          // shane: 一定要记住保存连接，后面需要用到

	done := make(chan struct{})
	go s.ReceiveMessages(conn, done) // shane: 启动接收消息的协程
	// shane: 阻塞当前函数
	<-done
}

// ReceiveMessages shane: 接收前端发送的消息,没有返回值
func (s *FrontendServer) ReceiveMessages(conn *websocket.Conn, done chan struct{}) {
	// TODO: 设计读超时

	// shane: 接收前端发送的消息
	for {
		if conn == nil {
			log.Println("Connection is nil, waiting for connection")
			break
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Receive from frontend failed:", err)
			delete(s.clients, s.conn)
			break
		}
		// shane: 主动关闭连接
		if string(msg) == "close" {
			log.Println("Frontend requested to close connection")
			break
		}
		log.Printf("Receive from frontend: %s", string(msg))
		s.SendMessages(conn, msg) // shane: 接收到消息之后发送消息
	}

	close(done) // shane: 关闭done通道，通知主协程结束
}

func (s *FrontendServer) SendMessages(conn *websocket.Conn, msg []byte) {
	if conn == nil {
		log.Println("Connection is nil, waiting for connection")
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Println("Sending message failed:", err)
	}
}
