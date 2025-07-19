package ws

import (
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"math"
	"time"
)

type BackendServer struct {
	BackendUrl   string
	clients      map[*websocket.Conn]bool // shane: 存储连接的客户端
	Conn         *websocket.Conn
	reconnecting bool // shane: add reconnecting 重连
}

func NewBackendServer(Url string) *BackendServer {
	return &BackendServer{
		BackendUrl:   Url,
		clients:      make(map[*websocket.Conn]bool),
		reconnecting: false,
	}
}

func (s *BackendServer) Connect(callType string) (*websocket.Conn, error) {
	url := s.BackendUrl
	url += "/call/" + callType
	var err error
	if s.Conn != nil {
		err := s.Conn.Close()
		if err != nil {
			return nil, err
		}
	}
	if s.Conn, _, err = websocket.DefaultDialer.Dial(url, nil); err != nil {
		logrus.Fatal(err)
		return nil, err
	} else {
		logrus.Info("Connected to backend")
		return s.Conn, nil
	}
}

func (s *BackendServer) reconnect(callType string) error {
	if s.reconnecting {
		return nil
	}
	s.reconnecting = true
	defer func() { s.reconnecting = false }()
	var maxAttempts = 5
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		logrus.Infof("Trying to reconnect to backend (attempt %d/%d)", i+1, maxAttempts)
		conn, err := s.Connect(callType)
		if err == nil {
			s.Conn = conn
			logrus.Info("Reconnected to backend successfully")
			return nil
		}
		// shane: capture the last error
		lastErr = err
		logrus.Warnf("Reconnection attempt %d failed: %v", i+1, err)
		time.Sleep(time.Second * time.Duration(math.Pow(2, float64(i))))
	}

	logrus.Error("Failed to reconnect after multiple attempts")
	return lastErr
}
