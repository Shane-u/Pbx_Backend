package ws

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"pbx_back_end"
	"pbx_back_end/internal/handler"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/shenjinti/go711"
	"github.com/shenjinti/go722"
)

// FrontendServer 管理前端WebSocket连接
type FrontendServer struct {
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]bool
	conn           *websocket.Conn
	RealTimeConn   *websocket.Conn
	llm            *handler.LLMHandler
	siliconFlowLLM *handler.SiliconFlowHandler // shane: siliconflow LLM handler
	backendConn    *websocket.Conn             // shane: 与后端的连接
	backendServer  *BackendServer              // shane: 后端服务实例
	codec          string                      // shane: codec for audio stream
	asrOption      *pbx_back_end.ASROption
	ttsOption      *pbx_back_end.TTSOption
}

func NewFrontendServer(llm *handler.LLMHandler, siliconFlowLLM *handler.SiliconFlowHandler, backendConn *websocket.Conn, backendServer *BackendServer, codec string, asrOption *pbx_back_end.ASROption, ttsOption *pbx_back_end.TTSOption) *FrontendServer {
	// func NewFrontendServer(llm *handler.LLMHandler, siliconFlowLLM *handler.SiliconFlowHandler, codec string, asrOption *pbx_back_end.ASROption, ttsOption *pbx_back_end.TTSOption) *FrontendServer {
	return &FrontendServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // shane: 允许跨域
		}, // http的升级
		clients:        make(map[*websocket.Conn]bool),
		llm:            llm,
		siliconFlowLLM: siliconFlowLLM, // shane: siliconflow LLM handler
		backendConn:    backendConn,
		backendServer:  backendServer,
		codec:          codec,
		asrOption:      asrOption,
		ttsOption:      ttsOption,
	}
}

// Start shane: http 升级为 websocket
func (s *FrontendServer) Start(r *gin.Engine, port string) {
	// shane: 使用gin处理WebSocket连接
	r.GET("/ws", func(c *gin.Context) {
		s.handleWebSocket(c.Writer, c.Request) // shane: http请求升级为WebSocket连接
	})
	// shane: 新增路由 /ws2 处理实时语音
	r.GET("/ws2", func(c *gin.Context) {
		s.handleWebSocket2(c.Writer, c.Request)
	})

	// shane: 监听后端消息
	if s.backendConn != nil {
		go s.receiveBackendMessages()
	}

	go func() {
		if err := r.Run(":" + port); err != nil {
			logrus.Error("Connection failed:", err)
		}
	}() // shane: 开协程防止阻塞
	logrus.Infof("Connected to the front end! Serve on %s", port)
}

// shane: 处理前端WebSocket连接
func (s *FrontendServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error("Upgrade Connection Failed:", err)
		return
	}
	defer func() {
		conn.Close()
		delete(s.clients, conn)
		logrus.Info("WebSocket Connection closed")
	}()
	s.clients[conn] = true // shane: 设置已经连接（状态信息）
	s.conn = conn          // shane: 一定要记住保存连接，后面需要用到

	done := make(chan struct{})
	go s.ReceiveMessages(conn, done) // shane: 启动接收消息的协程
	// shane: 阻塞当前函数
	<-done
}

func (s *FrontendServer) handleWebSocket2(w gin.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error("Upgrade Connection Failed:", err)
		return
	}
	defer func() {
		conn.Close()
		delete(s.clients, conn)
		s.RealTimeConn = nil
		logrus.Info("RealTime WebSocket Connection closed")
	}()
	s.clients[conn] = true // shane: 设置已经连接（状态信息）
	s.RealTimeConn = conn  // shane: 一定要记住保存连接，后面需要用到

	done := make(chan struct{})
	go s.ReceiveRealTimeMessage(conn, done) // shane: 启动接收实时消息的协程
	<-done
}

// convertAudio shane: parse audio data and convert to g722 or pcmu
func (s *FrontendServer) convertAudio(audioData []byte) ([]byte, error) {
	if s.codec == "g722" {
		encoder := go722.NewG722Encoder(go722.Rate64000, 0)
		return encoder.Encode(audioData), nil
	} else if s.codec == "pcmu" {
		return go711.EncodePCMU(audioData)
	} else if s.codec == "pcm" {
		return audioData, nil
	} else if s.codec == "wav" {
		return audioData, nil
	}

	return audioData, nil
}

// ReceiveMessages shane: 接收前端发送的消息,没有返回值
func (s *FrontendServer) ReceiveMessages(conn *websocket.Conn, done chan struct{}) {
	// TODO: 设计读超时

	// shane: 接收前端发送的消息
	for {
		if conn == nil {
			logrus.Error("Connection is nil, waiting for connection")
			break
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logrus.Error("Receive from frontend failed:", err)
			delete(s.clients, s.conn)
			break
		}
		// shane: 主动关闭连接
		if string(msg) == "close" {
			logrus.Info("Frontend requested to close connection")
			break
		}
		logrus.Infof("Receive from frontend: %s", string(msg))
		// s.handleMessage(msg)
		s.SendMessages(conn, msg) // shane: 接收到消息之后发送消息
	}

	close(done) // shane: 关闭done通道，通知主协程结束
}

func (s *FrontendServer) ReceiveRealTimeMessage(conn *websocket.Conn, done chan struct{}) {
	for {
		if conn == nil {
			logrus.Info("Connection is nil, waiting for connection")
			break
		}
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			logrus.Error("Receive Frontend Message failed:", err)
			break
		}
		// shane: 主动关闭连接
		if string(msg) == "close" {
			logrus.Info("Frontend requested to close connection")
			break
		}

		if msgType == websocket.TextMessage {
			// shane: parse message
			var frontendEvent struct {
				Event     string          `json:"event"`
				Sdp       string          `json:"sdp"`
				Candidate json.RawMessage `json:"candidate"`
				Command   string          `json:"command"`
				Reason    string          `json:"reason"`
				Initiator string          `json:"initiator"`
			}
			if err := json.Unmarshal(msg, &frontendEvent); err != nil {
				logrus.Error("parse front end message failed:", err)
				continue
			}

			// shane: receive offer
			if frontendEvent.Event == "offer" && frontendEvent.Sdp != "" {
				logrus.Infof("receive front end offer message, sdp: %s", frontendEvent.Sdp)

				inviteCmd := pbx_back_end.InviteCommand{
					Command: "invite",
					Option: pbx_back_end.CallOption{
						Offer:  frontendEvent.Sdp,
						Caller: "frontend",
						Callee: "rust",
						ASR:    s.asrOption,
						TTS:    s.ttsOption,
					},
				}

				cmdBytes, err := json.Marshal(inviteCmd)
				if err != nil {
					logrus.Error("marshal invite command failed:", err)
					continue
				}
				if err := s.backendConn.WriteMessage(websocket.TextMessage, cmdBytes); err != nil {
					logrus.Infof("forward candidate command to rust backend err: %v, Command data: %s", err, string(cmdBytes))
					if s.backendConn == nil {
						logrus.Errorf("Backend connection is nil, trying to reconnect")
						err := s.backendServer.reconnect("webrtc")
						if err != nil {
							return
						} else {
							// shane: 重发invite
							logrus.Info("Reconnected to backend successfully, will retry sending invite command")
							s.backendConn = s.backendServer.Conn
							if err := s.backendConn.WriteMessage(websocket.TextMessage, cmdBytes); err != nil {
								logrus.Error("Retrying to forward invite command failed:", err)
							} else {
								logrus.Info("Successfully retried forwarding invite command to rust backend")
							}
						}
					} else {
						logrus.Error("Failed to forward invite command to rust backend, will retry later")
					}
				} else {
					logrus.Info("Forwarded invite command with ASR config to rust backend")
				}
			}
			// shane: handle hangup event
			if frontendEvent.Command == "hangup" {
				hangupCmd := pbx_back_end.HangupCommand{
					Command:   "hangup",
					Reason:    frontendEvent.Reason,
					Initiator: frontendEvent.Initiator,
				}

				cmdBytes, err := json.Marshal(hangupCmd)
				if err != nil {
					log.Println("marshal hangup command failed:", err)
					continue
				}
				if err := s.backendConn.WriteMessage(websocket.TextMessage, cmdBytes); err != nil {
					log.Println("forward hangup command to rust backend err:", err)
					err := s.backendServer.reconnect("webrtc")
					if err != nil {
						return
					} else {
						// shane: 重发hangup
						s.backendConn = s.backendServer.Conn
						if err := s.backendConn.WriteMessage(websocket.TextMessage, cmdBytes); err != nil {
							log.Println("Retrying to forward hangup command failed:", err)
						} else {
							log.Println("Successfully retried forwarding hangup command to rust backend")
						}
					} // shane: 重连后端
				} else {
					log.Println("Forwarded hangup command to rust backend")
				}
			}
		}
	}
	close(done)
}

// SendMessages shane: 接收到消息之后发送消息
func (s *FrontendServer) SendMessages(conn *websocket.Conn, msg []byte) {
	if conn == nil {
		logrus.Error("Connection is nil, waiting for connection")
		return
	}

	// shane: 发送LLM处理后的消息到前端
	//response, _, err := s.llm.Query("qwen-turbo", string(msg))
	// shane: 发送SiliconFlowLLM处理后的消息到前端
	response, err := s.siliconFlowLLM.Query(string(msg))
	if err != nil {
		logrus.Error("LLM query failed:", err)
		return
	}
	// shane: 发送回复到前端
	if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
		logrus.Error("sendMessage failed:", err)
	}
}

// receiveBackendMessages shane: 接收并打印后端发送的消息
func (s *FrontendServer) receiveBackendMessages() {
	callType := "webrtc"
	for {
		if s.backendConn == nil {
			// shane: reconnect the backend connection
			err := s.backendServer.reconnect(callType)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			s.backendConn = s.backendServer.Conn
		}
		// shane: read message from backend
		messageType, msg, err := s.backendConn.ReadMessage()
		if err != nil {
			// shane: backend connection is closed, set it to nil, and continue the loop
			s.backendConn = nil
			continue
		}
		// shane: type down message fron rust backend
		logrus.Infof("Received from rust backend (type %d): %s", messageType, string(msg))

		// shane: parse the message
		var event struct {
			Event string `json:"event"`
			Text  string `json:"text"`
		}
		if err := json.Unmarshal(msg, &event); err == nil {
			// shane: handle asrFinal and send ASR result to LLM handler
			if event.Event == "asrFinal" && event.Text != "" {
				logrus.Infof("received ASR response: %s", event.Text)

				// shane: use LLM to handle ASR result
				if s.llm != nil {
					logrus.Info("handle ASR result via LLM...")
					//response, _, err := s.llm.Query("qwen-turbo", event.Text)
					response, err := s.siliconFlowLLM.Query(event.Text)
					if err != nil {
						logrus.Error("LLM handle ASR result failed:", err)
					} else {
						logrus.Infof("LLM response: %s", response)
						if s.RealTimeConn != nil {
							if err := s.RealTimeConn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
								logrus.Errorf("Failed to send LLM response to frontend: %v", err)
								if websocket.IsCloseError(err, websocket.CloseGoingAway) {
									s.RealTimeConn = nil
								}
							}
						} else {
							logrus.Error("RealTime conn is nil, cannot send LLM response")
						}

						// shane: send TTS command to Rust backend
						ttsCmd := pbx_back_end.TtsCommand{
							Command: "tts",
							Text:    response,
							Speaker: s.ttsOption.Speaker,
							Option:  s.ttsOption,
						}

						cmdBytes, err := json.Marshal(ttsCmd)
						if err != nil {
							logrus.Error("generate TTS Command failed:", err)
						} else {
							if err := s.backendConn.WriteMessage(websocket.TextMessage, cmdBytes); err != nil {
								logrus.Error("send TTS command to Rust backend failed:", err)
							} else {
								logrus.Info("TTS command sent to Rust backend successfully")
							}
						}
					}
				}
			} else if event.Event == "asrDelta" {
				// shane: handle ASR delta event
				logrus.Infof("ASR realtime recognize: %s", event.Text)
			} else if event.Event == "speaking" {
				logrus.Info("detecting speaking")
			} else if event.Event == "silence" {
				logrus.Info("detecting silence")
			} else if event.Event == "trackStart" {
				logrus.Info("track started")
			} else if event.Event == "trackEnd" {
				logrus.Info("track ended")
			}
		}

		// shane: forward the message to the frontend
		s.forwardRustMessageToFrontend(msg)
	}

	// log.Println("Stopped listening for backend messages") // shane: 自动重连监听
}

// forwardRustMessageToFrontend shane: 转发后端消息给前端
func (s *FrontendServer) forwardRustMessageToFrontend(msg []byte) {
	if s.RealTimeConn != nil {
		if err := s.RealTimeConn.WriteMessage(websocket.TextMessage, msg); err != nil {
			logrus.Error("Failed to forward backend message to frontend:", err)
		} else {
			logrus.Info("Successfully forwarded backend message to frontend")
		}
	} else {
		logrus.Error("Frontend connection is nil, cannot forward message")
	}
}
