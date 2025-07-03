package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDurationUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "seconds",
			input: `"30s"`,
			want:  30 * time.Second,
		},
		{
			name:  "minutes",
			input: `"5m"`,
			want:  5 * time.Minute,
		},
		{
			name:  "complex duration",
			input: `"1h30m45s"`,
			want:  1*time.Hour + 30*time.Minute + 45*time.Second,
		},
		{
			name:    "invalid duration",
			input:   `"invalid"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.input), &d)
			if (err != nil) != tt.wantErr {
				t.Errorf("Duration.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && d.Duration != tt.want {
				t.Errorf("Duration = %v, want %v", d.Duration, tt.want)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
		wantErr  bool
		check    func(t *testing.T, cfg *FileConfig)
	}{
		{
			name:     "yaml config",
			filename: "test.yaml",
			content: `headless: false
timeout: 45s
screenshotDir: custom_dir
screenshotThreshold: 0.05
viewportWidth: 1920
viewportHeight: 1080`,
			check: func(t *testing.T, cfg *FileConfig) {
				if cfg.Headless == nil || *cfg.Headless != false {
					t.Error("Expected headless to be false")
				}
				if cfg.Timeout == nil || cfg.Timeout.Duration != 45*time.Second {
					t.Error("Expected timeout to be 45s")
				}
				if cfg.ScreenshotDir != "custom_dir" {
					t.Error("Expected custom screenshot dir")
				}
				if cfg.ScreenshotThreshold != 0.05 {
					t.Error("Expected threshold to be 0.05")
				}
				if cfg.ViewportWidth != 1920 {
					t.Error("Expected viewport width to be 1920")
				}
			},
		},
		{
			name:     "json config",
			filename: "test.json",
			content: `{
  "headless": true,
  "timeout": "30s",
  "failOnConsoleError": true,
  "screenshotDir": "screenshots",
  "updateScreenshots": true
}`,
			check: func(t *testing.T, cfg *FileConfig) {
				if cfg.Headless == nil || *cfg.Headless != true {
					t.Error("Expected headless to be true")
				}
				if cfg.FailOnConsoleError == nil || *cfg.FailOnConsoleError != true {
					t.Error("Expected failOnConsoleError to be true")
				}
				if cfg.UpdateScreenshots != true {
					t.Error("Expected updateScreenshots to be true")
				}
			},
		},
		{
			name:     "config with action timeouts",
			filename: "test.yaml",
			content: `timeout: 30s
actionTimeouts:
  navigate: 20s
  click: 10s
  type: 5s`,
			check: func(t *testing.T, cfg *FileConfig) {
				if cfg.ActionTimeouts == nil {
					t.Fatal("Expected actionTimeouts to be set")
				}
				if cfg.ActionTimeouts["navigate"] == nil || cfg.ActionTimeouts["navigate"].Duration != 20*time.Second {
					t.Error("Expected navigate timeout to be 20s")
				}
				if cfg.ActionTimeouts["click"] == nil || cfg.ActionTimeouts["click"].Duration != 10*time.Second {
					t.Error("Expected click timeout to be 10s")
				}
			},
		},
		{
			name:     "invalid yaml",
			filename: "test.yaml",
			content:  `invalid: yaml: content:`,
			wantErr:  true,
		},
		{
			name:     "invalid json",
			filename: "test.json",
			content:  `{invalid json}`,
			wantErr:  true,
		},
		{
			name:     "unsupported format",
			filename: "test.txt",
			content:  `some content`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			cfg, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestFindConfigFile(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Test no config file
	if found := FindConfigFile(); found != "" {
		t.Errorf("Expected empty string, got %s", found)
	}

	// Test finding different config files
	configFiles := []string{
		"fasttest.config.yaml",
		"fasttest.yml",
		".fasttest.json",
	}

	for _, filename := range configFiles {
		// Clean up previous files
		files, _ := filepath.Glob("*fasttest*")
		for _, f := range files {
			os.Remove(f)
		}

		// Create config file
		err := os.WriteFile(filename, []byte("test"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		found := FindConfigFile()
		if found != filename {
			t.Errorf("Expected to find %s, got %s", filename, found)
		}
	}
}

func TestFileConfigDefaults(t *testing.T) {
	cfg := &FileConfig{}
	
	// Test that zero values are as expected
	if cfg.Headless != nil {
		t.Error("Expected Headless to be nil by default")
	}
	if cfg.Timeout != nil {
		t.Error("Expected Timeout to be nil by default")
	}
	if cfg.ScreenshotDir != "" {
		t.Error("Expected ScreenshotDir to be empty by default")
	}
	if cfg.ScreenshotThreshold != 0 {
		t.Error("Expected ScreenshotThreshold to be 0 by default")
	}
	if cfg.UpdateScreenshots != false {
		t.Error("Expected UpdateScreenshots to be false by default")
	}
}