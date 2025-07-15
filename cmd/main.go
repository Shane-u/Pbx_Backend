package main

import (
	"log"
	"pbx_back_end/internal/ws"
)

func main() {
	// shane: 前端建立连接
	frontendServer := ws.NewFrontendServer()
	frontendServer.Start("8080")
	// shane: 后端建立连接
	backendServer := ws.NewBackendServer("ws://175.27.250.177:8080")
	_, err := backendServer.Connect("webrtc")
	if err != nil {
		log.Fatalf("Unable to connect to backend: %v", err)
	} else {
		log.Println("Connected to backend successfully!")
	}
	//// shane: 从前端获取消息并发送到后端
	//go func() {
	//	for message := range frontendServer.GetMessageChan() {
	//		log.Printf("从前端收到消息: %s", string(message))
	//		backendServer.SendMessage(message)
	//	}
	//}()

	// shane: 处理从后端接收的消息
	//go func() {
	//	for message := range backendClient.GetReceiveChan() {
	//		log.Printf("从后端收到消息: %s", string(message))
	//	}
	//}()

	// shane: keep alive
	select {}
}
