package ws

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

// FrontendServer 管理前端WebSocket连接
type FrontendServer struct {
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	conn     *websocket.Conn
}

func NewFrontendServer() *FrontendServer {
	return &FrontendServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // shane: 允许跨域
		}, // http的升级
		clients: make(map[*websocket.Conn]bool),
	}
}

// shane: 启动HTTP服务，提供WebSocket升级
func (s *FrontendServer) Start(port string) {
	http.HandleFunc("/ws", s.handleWebSocket) // shane: 第一个参数相当于是句柄，如果遇到这个句柄就会调用连接函数
	log.Printf("连接成功！")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// shane: 处理前端WebSocket连接
func (s *FrontendServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("升级连接失败:", err)
		return
	}
	defer conn.Close()
	s.clients[conn] = true // shane: 设置已经连接（状态信息）
}

func (s *FrontendServer) GetMessageChan() []byte {
	// shane: 接收前端发送的消息
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			log.Println("接收消息失败:", err)
			delete(s.clients, s.conn)
			break
		}
		return msg
	}
	return nil
}
