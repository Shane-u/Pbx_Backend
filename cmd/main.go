package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"pbx_back_end"
	"pbx_back_end/config"
	"pbx_back_end/internal/api"
	"pbx_back_end/internal/handler"
	"pbx_back_end/internal/ws"
	"pbx_back_end/repository"
)

func main() {
	cfg, err := config.LoadConfig("../config.yaml") // shane: ! 注意修改配置文件
	if err != nil {
		logrus.Errorf("failed to load config: %v", err)
	}

	asrOption := &pbx_back_end.ASROption{
		Provider:   cfg.ASR.Provider,
		Language:   cfg.ASR.Language,
		SampleRate: cfg.ASR.SampleRate,
		AppID:      cfg.ASR.AppID,
		SecretID:   cfg.ASR.SecretID,
		SecretKey:  cfg.ASR.SecretKey,
		Endpoint:   cfg.ASR.Endpoint,
		ModelType:  cfg.ASR.ModelType,
	}

	ttsOption := &pbx_back_end.TTSOption{
		Provider:   cfg.TTS.Provider,
		Samplerate: cfg.TTS.SampleRate,
		Speaker:    cfg.TTS.Speaker,
		Speed:      cfg.TTS.Speed,
		Volume:     cfg.TTS.Volume,
		AppID:      cfg.TTS.AppID,
		SecretID:   cfg.TTS.SecretID,
		SecretKey:  cfg.TTS.SecretKey,
		Codec:      cfg.TTS.Codec,
		Endpoint:   cfg.TTS.Endpoint,
	}

	ctx := context.Background()
	logger := logrus.New()
	llm := handler.NewLLMHandler(ctx, cfg.LLM.APIKey, cfg.LLM.URL, cfg.LLM.SystemPrompt, logger)
	siliconFlowLLM := handler.NewSiliconFlowHandler(ctx, cfg.LLM.SiliconFlow.APIKey, cfg.LLM.SiliconFlow.URL, cfg.LLM.SiliconFlow.Model, logger, cfg.BigModel.SearchApiUrl, cfg.BigModel.SearchApiKey, cfg.BigModel.SearchApiModel)

	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	// shane: 初始化数据库
	repo, err := repository.NewRobotRepository(cfg.Database.DSN)
	if err != nil {
		logrus.Errorf("database connection failed: %v", err)
	} else {
		logrus.Info("database connected successfully")
	}

	api.Routers(r, repo)

	// shane: 后端建立连接
	backendServer := ws.NewBackendServer(cfg.Backend.URL)
	backendConn, err := backendServer.Connect(cfg.Backend.CallType)
	if err != nil {
		logrus.Errorf("Unable to connect to backend: %v", err)
	} else {
		logrus.Info("Connected to backend successfully!")
	}
	// shane: 前端建立连接
	frontendServer := ws.NewFrontendServer(llm, siliconFlowLLM, backendConn, backendServer, cfg.Audio.Codec, asrOption, ttsOption)
	frontendServer.Start(r, cfg.Server.Port)

	select {}
}
