package ws

import (
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type BackendServer struct {
	BackendUrl string
	clients    map[*websocket.Conn]bool // shane: 存储连接的客户端
	Conn       *websocket.Conn
}

func NewBackendServer(Url string) *BackendServer {
	return &BackendServer{
		BackendUrl: Url,
		clients:    make(map[*websocket.Conn]bool),
	}
}

func (s *BackendServer) Connect(callType string) (*websocket.Conn, error) {
	url := s.BackendUrl
	url += "/call/" + callType
	var err error
	if s.Conn, _, err = websocket.DefaultDialer.Dial(url, nil); err != nil {
		logrus.Fatal(err)
		return nil, err
	} else {
		logrus.Info("Connected to backend")
		return s.Conn, nil
	}

}
