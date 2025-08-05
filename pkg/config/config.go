package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// FileConfig represents the configuration loaded from a file
type FileConfig struct {
	Headless            *bool                `yaml:"headless" json:"headless"`
	Timeout             *Duration            `yaml:"timeout" json:"timeout"`
	FailOnConsoleError  *bool                `yaml:"failOnConsoleError" json:"failOnConsoleError"`
	ScreenshotDir       string               `yaml:"screenshotDir" json:"screenshotDir"`
	UpdateScreenshots   bool                 `yaml:"updateScreenshots" json:"updateScreenshots"`
	ScreenshotThreshold float64              `yaml:"screenshotThreshold" json:"screenshotThreshold"`
	ViewportWidth       int                  `yaml:"viewportWidth" json:"viewportWidth"`
	ViewportHeight      int                  `yaml:"viewportHeight" json:"viewportHeight"`
	BrowserType         string               `yaml:"browserType" json:"browserType"`
	ActionTimeouts      map[string]*Duration `yaml:"actionTimeouts" json:"actionTimeouts"`
}

// Duration is a custom type for unmarshaling duration strings
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

// LoadConfig loads configuration from file
func LoadConfig(filename string) (*FileConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config FileConfig
	ext := filepath.Ext(filename)

	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	case ".json":
		err = json.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

// FindConfigFile searches for a config file in the current directory
func FindConfigFile() string {
	configNames := []string{
		"testit.config.yaml",
		"testit.config.yml",
		"testit.config.json",
		"testit.yaml",
		"testit.yml",
		"testit.json",
		".testit.yaml",
		".testit.yml",
		".testit.json",
	}

	for _, name := range configNames {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}

	return ""
}
