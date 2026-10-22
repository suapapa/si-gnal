package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// OpenAIConfig is populated from the environment (see cmd/server/.env.example),
// not from config.yaml.
type OpenAIConfig struct {
	Endpoint string // OPENAI_ENDPOINT
	APIKey   string // OPENAI_APIKEY
	Model    string // OPENAI_MODEL
	TTSModel string // OPENAI_TTS_MODEL
}

type Config struct {
	Server struct {
		Listen string `yaml:"listen"`
	} `yaml:"server"`
	PoemQueue struct {
		Batch     int  `yaml:"batch"`
		UseMemory bool `yaml:"use_memory"`
	} `yaml:"poem_queue"`
	Speaker struct {
		WirephoneEffect bool `yaml:"wirephone_effect"`
		AddNoise        bool `yaml:"add_noise"`
	} `yaml:"speaker"`
	// Voices lists OpenAI TTS voice names; one is picked at random per poem.
	// Empty means the TTS engine's default voice.
	Voices []string `yaml:"voices"`

	OpenAI OpenAIConfig `yaml:"-"`
}

func loadConfig(path string) (*Config, error) {
	loadDotEnv(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.OpenAI = openAIConfigFromEnv()
	normalizeConfig(&cfg)
	return &cfg, nil
}

func openAIConfigFromEnv() OpenAIConfig {
	return OpenAIConfig{
		Endpoint: os.Getenv("OPENAI_ENDPOINT"),
		APIKey:   os.Getenv("OPENAI_APIKEY"),
		Model:    os.Getenv("OPENAI_MODEL"),
		TTSModel: os.Getenv("OPENAI_TTS_MODEL"),
	}
}

func normalizeConfig(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	if cfg.PoemQueue.Batch <= 0 {
		cfg.PoemQueue.Batch = 5
	}
	if cfg.OpenAI.Endpoint == "" {
		cfg.OpenAI.Endpoint = "https://api.openai.com/v1"
	}
	if cfg.OpenAI.Model == "" {
		cfg.OpenAI.Model = "gpt-4o-mini"
	}
	if cfg.OpenAI.TTSModel == "" {
		cfg.OpenAI.TTSModel = "gpt-4o-mini-tts"
	}
	voices := make([]string, 0, len(cfg.Voices))
	for _, v := range cfg.Voices {
		if v = strings.TrimSpace(v); v != "" {
			voices = append(voices, v)
		}
	}
	cfg.Voices = voices
}

// loadDotEnv loads KEY=VALUE pairs from a .env file into the process
// environment without overriding variables that are already set.
// It looks for .env next to the config file and in the working directory.
func loadDotEnv(configPath string) {
	candidates := []string{".env"}
	if dir := filepath.Dir(configPath); dir != "" && dir != "." {
		candidates = append([]string{filepath.Join(dir, ".env")}, candidates...)
	}
	for _, path := range candidates {
		if err := parseDotEnv(path); err == nil {
			return
		}
	}
}

func parseDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		if key == "" {
			continue
		}
		if _, alreadySet := os.LookupEnv(key); alreadySet {
			continue
		}
		_ = os.Setenv(key, value)
	}
	return scanner.Err()
}
