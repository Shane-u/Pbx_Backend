package main

import (
	"context"
	"log"
	"pbx_back_end"
	"pbx_back_end/internal/handler"
	"pbx_back_end/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// shane: AppConfig
type AppConfig struct {
	Server struct {
		Port string
	}
	Backend struct {
		URL      string
		CallType string
	}
	Audio struct {
		Codec string // shane: g722
	}
	ASR struct {
		Provider   string
		Language   string
		SampleRate uint32
		AppID      string
		SecretID   string
		SecretKey  string
		Endpoint   string
		ModelType  string
	}
	TTS struct {
		Provider   string
		SampleRate int32
		Speaker    string
		Speed      float32
		Volume     int32
		Emotion    string
		AppID      string
		SecretID   string
		SecretKey  string
		Codec      string
		Endpoint   string
	}
	LLM struct {
		APIKey       string
		Model        string
		URL          string
		SystemPrompt string
	}
	VAD struct {
		Model     string
		Endpoint  string
		SecretKey string
	}
	Call struct {
		BreakOnVAD bool
		WithSIP    bool
		Record     bool
		Caller     string
		Callee     string
	}
	WebHook struct {
		Addr   string
		Prefix string
	}
	EOU struct {
		Type     string
		Endpoint string
	}
}

func main() {
	// shane: 硬编码
	endpoint := "ws://175.27.250.177:8080"
	codec := "g722"
	openaiKey := "yours"
	openaiModel := "qwen-turbo"
	openaiEndpoint := "https://dashscope.aliyuncs.com/compatible-mode/v1"
	systemPrompt := "You are a helpful assistant. Provide concise responses. Use 'hangup' tool when the conversation is complete."
	breakOnVad := false
	speaker := "101016"
	callWithSip := false
	record := false
	ttsProvider := "tencent"
	asrProvider := "tencent"
	caller := ""
	callee := ""
	asrEndpoint := "asr.tencentcloudapi.com"
	asrAppID := "yours"
	asrSecretID := "yours"
	asrSecretKey := "yours"
	asrModelType := "16k_zh"
	ttsEndpoint := "tts.tencentcloudapi.com"
	ttsAppID := "yours"
	ttsSecretID := "yours"
	ttsSecretKey := "yours"
	vadModel := "silero"
	vadEndpoint := ""
	vadSecretKey := ""
	webhookAddr := ""
	webhookPrefix := "/webhook"
	eouType := ""
	eouEndpoint := ""

	// shane: init Config
	config := AppConfig{}
	config.Server.Port = "8080"
	config.Backend.URL = endpoint
	config.Backend.CallType = "webrtc"
	config.Audio.Codec = codec
	config.Call.BreakOnVAD = breakOnVad
	config.Call.WithSIP = callWithSip
	config.Call.Record = record
	config.Call.Caller = caller
	config.Call.Callee = callee
	config.ASR.Provider = asrProvider
	config.ASR.Language = "zh-CN"
	config.ASR.SampleRate = 16000
	config.ASR.AppID = asrAppID
	config.ASR.SecretID = asrSecretID
	config.ASR.SecretKey = asrSecretKey
	config.ASR.Endpoint = asrEndpoint
	config.ASR.ModelType = asrModelType
	config.TTS.Provider = ttsProvider
	config.TTS.SampleRate = 16000
	config.TTS.Speaker = speaker
	config.TTS.Speed = 1.0
	config.TTS.Volume = 10
	config.TTS.AppID = ttsAppID
	config.TTS.SecretID = ttsSecretID
	config.TTS.SecretKey = ttsSecretKey
	config.TTS.Endpoint = ttsEndpoint
	config.TTS.Codec = "g722"
	config.VAD.Model = vadModel
	config.VAD.Endpoint = vadEndpoint
	config.VAD.SecretKey = vadSecretKey
	config.WebHook.Addr = webhookAddr
	config.WebHook.Prefix = webhookPrefix
	config.EOU.Type = eouType
	config.EOU.Endpoint = eouEndpoint
	config.LLM.APIKey = openaiKey
	config.LLM.Model = openaiModel
	config.LLM.URL = openaiEndpoint
	config.LLM.SystemPrompt = systemPrompt

	// shane: create ASR option
	asrOption := &pbx_back_end.ASROption{
		Provider:   config.ASR.Provider,
		Language:   config.ASR.Language,
		SampleRate: config.ASR.SampleRate,
		AppID:      config.ASR.AppID,
		SecretID:   config.ASR.SecretID,
		SecretKey:  config.ASR.SecretKey,
		Endpoint:   config.ASR.Endpoint,
		ModelType:  config.ASR.ModelType,
	}

	// shane: create TTS config
	ttsOption := &pbx_back_end.TTSOption{
		Provider:   config.TTS.Provider,
		Samplerate: config.TTS.SampleRate,
		Speaker:    config.TTS.Speaker,
		Speed:      config.TTS.Speed,
		Volume:     config.TTS.Volume,
		AppID:      config.TTS.AppID,
		SecretID:   config.TTS.SecretID,
		SecretKey:  config.TTS.SecretKey,
		Codec:      config.TTS.Codec,
		Endpoint:   config.TTS.Endpoint,
	}

	// shane: create LLM handler
	ctx := context.Background()
	logger := logrus.New()
	llm := handler.NewLLMHandler(ctx, config.LLM.APIKey, config.LLM.URL, config.LLM.SystemPrompt, logger)

	// shane: create media handler
	mediaHandler, err := handler.NewMediaHandler(ctx, logger)
	if err != nil {
		logger.Fatalf("Failed to create media handler: %v", err)
	}
	defer mediaHandler.Stop()

	r := gin.Default()
	// shane: 后端建立连接
	backendServer := ws.NewBackendServer(config.Backend.URL)
	backendConn, err := backendServer.Connect(config.Backend.CallType)
	if err != nil {
		log.Fatalf("Unable to connect to backend: %v", err)
	} else {
		log.Println("Connected to backend successfully!")
	}

	// shane: 前端建立连接
	frontendServer := ws.NewFrontendServer(llm, backendConn, config.Audio.Codec, asrOption, ttsOption)
	frontendServer.Start(r, config.Server.Port)

	// shane: keep alive
	select {}
}
