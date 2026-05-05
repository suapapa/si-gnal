package main

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Server struct {
		Listen string `yaml:"listen"`
	} `yaml:"server"`
	PoemQueue struct {
		Batch     int  `yaml:"batch"`
		UseMemory bool `yaml:"use_memory"`
	} `yaml:"poem_queue"`
	OpenAI struct {
		BaseURL string `yaml:"base_url"`
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
	} `yaml:"openai"`
	TTS struct {
		Engine     string `yaml:"engine"`
		Supertonic struct {
			ONNXDir    string `yaml:"onnx_dir"`
			VoiceStyle string `yaml:"voice_style"`
		} `yaml:"supertonic"`
	} `yaml:"tts"`
	Wirephone struct {
		Enabled  bool `yaml:"enabled"`
		AddNoise bool `yaml:"add_noise"`
	} `yaml:"wirephone"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	normalizeConfig(&cfg)
	cfg.OpenAI.APIKey = os.ExpandEnv(cfg.OpenAI.APIKey)
	cfg.OpenAI.BaseURL = os.ExpandEnv(cfg.OpenAI.BaseURL)
	cfg.OpenAI.Model = os.ExpandEnv(cfg.OpenAI.Model)
	cfg.TTS.Supertonic.ONNXDir = os.ExpandEnv(cfg.TTS.Supertonic.ONNXDir)
	cfg.TTS.Supertonic.VoiceStyle = os.ExpandEnv(cfg.TTS.Supertonic.VoiceStyle)
	return &cfg, nil
}

func normalizeConfig(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	if cfg.PoemQueue.Batch <= 0 {
		cfg.PoemQueue.Batch = 5
	}
	if cfg.OpenAI.Model == "" {
		cfg.OpenAI.Model = "gpt-4o-mini"
	}
	if cfg.TTS.Engine == "" {
		cfg.TTS.Engine = "supertonic"
	}
	if cfg.TTS.Supertonic.ONNXDir == "" {
		cfg.TTS.Supertonic.ONNXDir = "assets/supertonic2/onnx"
	}
	if cfg.TTS.Supertonic.VoiceStyle == "" {
		cfg.TTS.Supertonic.VoiceStyle = "assets/supertonic2/voice_styles/F5.json"
	}
}
