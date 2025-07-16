package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"log"
	"pbx_back_end/internal/handler"
	"pbx_back_end/internal/ws"
)

func main() {
	// TODO:shane: 加载配置文件
	// shane: 硬编码
	llmConfig := struct {
		apiKey       string
		model        string
		url          string
		systemPrompt string
	}{
		apiKey:       "sk-0cea2b4306694a7d96ddeee439f02401",
		model:        "qwen-turbo",
		url:          "https://dashscope.aliyuncs.com/compatible-mode/v1",
		systemPrompt: "You are a helpful assistant for a my health system.",
	}
	// shane: create LLM handler
	ctx := context.Background()
	logger := logrus.New()
	llm := handler.NewLLMHandler(ctx, llmConfig.apiKey, llmConfig.url, llmConfig.systemPrompt, logger)

	// shane: create media handler
	mediaHandler, err := handler.NewMediaHandler(ctx, logger)
	if err != nil {
		logger.Fatalf("Failed to create media handler: %v", err)
	}
	defer mediaHandler.Stop()

	r := gin.Default()
	// shane: 后端建立连接
	backendServer := ws.NewBackendServer("ws://175.27.250.177:8080")
	backendConn, err := backendServer.Connect("webrtc")
	if err != nil {
		log.Fatalf("Unable to connect to backend: %v", err)
	} else {
		log.Println("Connected to backend successfully!")
	}
	// shane: 前端建立连接
	frontendServer := ws.NewFrontendServer(llm, backendConn)
	frontendServer.Start(r, "8080")

	// shane: keep alive
	select {}
}
