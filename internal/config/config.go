package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"app"`

	Window struct {
		Width  int `yaml:"width"`
		Height int `yaml:"height"`
	} `yaml:"window"`

	APIs struct {
		AprsKey         string `yaml:"aprs_key"`
		RepeaterBookKey string `yaml:"repeater_book_key"`
		BrandmeisterKey string `yaml:"brandmeister_key"`
	} `yaml:"apis"`
}

func LoadConfig() (*Config, error) {
	configPath := filepath.Join("configs", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func GetDefaultConfig() *Config {
	return &Config{
		App: struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		}{
			Name:    "DigiLogRT",
			Version: "0.1.0",
		},
		Window: struct {
			Width  int `yaml:"width"`
			Height int `yaml:"height"`
		}{
			Width:  1200,
			Height: 800,
		},
		APIs: struct {
			AprsKey         string `yaml:"aprs_key"`
			RepeaterBookKey string `yaml:"repeater_book_key"`
			BrandmeisterKey string `yaml:"brandmeister_key"`
		}{
			AprsKey: "126515.6ryMvtanTmJDG",
		},
	}
}
