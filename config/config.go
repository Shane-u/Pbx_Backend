package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// Config shane: 配置文件
type Config struct {
	OpenAI OpenAIConfig `yaml:"openai"`
}

// OpenAIConfig shane: OpenAI相关配置
type OpenAIConfig struct {
	APIKey       string `yaml:"api_key"`
	Model        string `yaml:"model"`
	Url          string `yaml:"url"`
	SystemPrompt string `yaml:"system_prompt"`
	Streaming    bool   `yaml:"streaming,omitempty"`
}

// LoadConfig shane: 加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	// shane: parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}
