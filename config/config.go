package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Backend  BackendConfig  `yaml:"backend"`
	Audio    AudioConfig    `yaml:"audio"`
	ASR      ASRConfig      `yaml:"asr"`
	TTS      TTSConfig      `yaml:"tts"`
	LLM      LLMConfig      `yaml:"llm"`
	VAD      VADConfig      `yaml:"vad"`
	Call     CallConfig     `yaml:"call"`
	WebHook  WebHookConfig  `yaml:"webhook"`
	EOU      EOUConfig      `yaml:"eou"`
	Database DatabaseConfig `yaml:"database"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}
type BackendConfig struct {
	URL      string `yaml:"url"`
	CallType string `yaml:"call_type"`
}
type AudioConfig struct {
	Codec string `yaml:"codec"`
}
type ASRConfig struct {
	Provider   string `yaml:"provider"`
	Language   string `yaml:"language"`
	SampleRate uint32 `yaml:"sample_rate"`
	AppID      string `yaml:"app_id"`
	SecretID   string `yaml:"secret_id"`
	SecretKey  string `yaml:"secret_key"`
	Endpoint   string `yaml:"endpoint"`
	ModelType  string `yaml:"model_type"`
}
type TTSConfig struct {
	Provider        string  `yaml:"provider"`
	SampleRate      int32   `yaml:"sample_rate"`
	Speaker         string  `yaml:"speaker"`
	Speed           float32 `yaml:"speed"`
	Volume          int32   `yaml:"volume"`
	EmotionCategory string  `yaml:"emotion"`
	AppID           string  `yaml:"app_id"`
	SecretID        string  `yaml:"secret_id"`
	SecretKey       string  `yaml:"secret_key"`
	Codec           string  `yaml:"codec"`
	Endpoint        string  `yaml:"endpoint"`
}
type LLMConfig struct {
	APIKey       string `yaml:"api_key"`
	Model        string `yaml:"model"`
	URL          string `yaml:"url"`
	SystemPrompt string `yaml:"system_prompt"`
}
type VADConfig struct {
	Model     string `yaml:"model"`
	Endpoint  string `yaml:"endpoint"`
	SecretKey string `yaml:"secret_key"`
}
type CallConfig struct {
	BreakOnVAD bool   `yaml:"break_on_vad"`
	WithSIP    bool   `yaml:"with_sip"`
	Record     bool   `yaml:"record"`
	Caller     string `yaml:"caller"`
	Callee     string `yaml:"callee"`
}
type WebHookConfig struct {
	Addr   string `yaml:"addr"`
	Prefix string `yaml:"prefix"`
}
type EOUConfig struct {
	Type     string `yaml:"type"`
	Endpoint string `yaml:"endpoint"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}
