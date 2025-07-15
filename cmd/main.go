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
	// shane: 初始化LLM处理器
	ctx := context.Background()
	logger := logrus.New()
	llm := handler.NewLLMHandler(ctx, llmConfig.apiKey, llmConfig.url, llmConfig.systemPrompt, logger)

	r := gin.Default()
	// shane: 前端建立连接
	frontendServer := ws.NewFrontendServer(llm)
	frontendServer.Start(r, "8080")
	// shane: 后端建立连接
	backendServer := ws.NewBackendServer("ws://175.27.250.177:8080")
	_, err := backendServer.Connect("webrtc")
	if err != nil {
		log.Fatalf("Unable to connect to backend: %v", err)
	} else {
		log.Println("Connected to backend successfully!")
	}

	// shane: keep alive
	select {}
}
