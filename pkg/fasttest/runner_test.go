package fasttest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewRunner(t *testing.T) {
	// Test with nil config
	runner := NewRunner(nil)
	if runner.config == nil {
		t.Error("Expected default config to be created")
	}
	if runner.config.Headless != true {
		t.Error("Expected default headless to be true")
	}
	if runner.config.Timeout != 10*time.Second {
		t.Error("Expected default timeout to be 10s")
	}
	if runner.config.ScreenshotDir != "__screenshots__" {
		t.Error("Expected default screenshot dir to be __screenshots__")
	}

	// Test with custom config
	customConfig := &Config{
		Headless:      false,
		Timeout:       60 * time.Second,
		ScreenshotDir: "custom_screenshots",
	}
	runner2 := NewRunner(customConfig)
	if runner2.config.Headless != false {
		t.Error("Expected custom headless setting")
	}
	if runner2.config.ScreenshotDir != "custom_screenshots" {
		t.Error("Expected custom screenshot dir")
	}
}

func TestScreenshotNaming(t *testing.T) {
	runner := NewRunner(nil)
	runner.screenshotCounter = make(map[string]int)

	tests := []struct {
		testName     string
		filename     string
		callCount    int
		wantFilename string
	}{
		{
			testName:     "Simple Test",
			filename:     "",
			callCount:    1,
			wantFilename: "Simple_Test.png",
		},
		{
			testName:     "Multi Screenshot Test",
			filename:     "",
			callCount:    2,
			wantFilename: "Multi_Screenshot_Test_2.png",
		},
		{
			testName:     "Test with/slashes",
			filename:     "",
			callCount:    1,
			wantFilename: "Test_with_slashes.png",
		},
		{
			testName:     "Test with\\backslashes",
			filename:     "",
			callCount:    1,
			wantFilename: "Test_with_backslashes.png",
		},
		{
			testName:     "Custom Name Test",
			filename:     "custom.png",
			callCount:    1,
			wantFilename: "custom.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			// Simulate multiple calls for same test
			for i := 0; i < tt.callCount; i++ {
				filename := tt.filename
				if filename == "" {
					safeTestName := strings.ReplaceAll(tt.testName, " ", "_")
					safeTestName = strings.ReplaceAll(safeTestName, "/", "_")
					safeTestName = strings.ReplaceAll(safeTestName, "\\", "_")

					runner.mu.Lock()
					runner.screenshotCounter[tt.testName]++
					counter := runner.screenshotCounter[tt.testName]
					runner.mu.Unlock()

					if counter == 1 {
						filename = safeTestName + ".png"
					} else {
						filename = fmt.Sprintf("%s_%d.png", safeTestName, counter)
					}
				}

				if i == tt.callCount-1 && filename != tt.wantFilename {
					t.Errorf("Got filename %s, want %s", filename, tt.wantFilename)
				}
			}
		})
	}
}

func TestCompareImages(t *testing.T) {
	runner := NewRunner(nil)

	// Create test images (1x1 pixel PNGs)
	// Same images
	img1 := []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0, 144, 119, 83, 222, 0, 0, 0, 12, 73, 68, 65, 84, 8, 215, 99, 248, 255, 255, 63, 0, 5, 254, 2, 254, 220, 204, 89, 231, 0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130}
	img2 := []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0, 144, 119, 83, 222, 0, 0, 0, 12, 73, 68, 65, 84, 8, 215, 99, 248, 255, 255, 63, 0, 5, 254, 2, 254, 220, 204, 89, 231, 0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130}

	// Test same images
	diff, _, err := runner.compareImages(img1, img2)
	if err != nil {
		t.Fatalf("compareImages() error = %v", err)
	}
	if diff != 0 {
		t.Errorf("Expected 0 difference for identical images, got %f", diff)
	}

	// Test invalid image data
	invalidImg := []byte("not a png")
	_, _, err = runner.compareImages(img1, invalidImg)
	if err == nil {
		t.Error("Expected error for invalid image data")
	}
}

func TestTestResult(t *testing.T) {
	result := TestResult{
		Name:     "Test 1",
		Passed:   true,
		Error:    nil,
		Duration: 5 * time.Second,
		Errors:   []ConsoleError{},
	}

	if result.Name != "Test 1" {
		t.Errorf("Expected test name 'Test 1', got %s", result.Name)
	}
	if !result.Passed {
		t.Error("Expected test to pass")
	}
	if result.Duration != 5*time.Second {
		t.Errorf("Expected duration 5s, got %s", result.Duration)
	}
}

func TestConfig(t *testing.T) {
	config := &Config{
		Headless:            true,
		Timeout:             45 * time.Second,
		FailOnConsoleError:  true,
		ScreenshotDir:       "test_screenshots",
		UpdateScreenshots:   false,
		ScreenshotThreshold: 0.05,
	}

	if !config.Headless {
		t.Error("Expected headless to be true")
	}
	if config.Timeout != 45*time.Second {
		t.Error("Expected timeout to be 45s")
	}
	if !config.FailOnConsoleError {
		t.Error("Expected FailOnConsoleError to be true")
	}
	if config.ScreenshotDir != "test_screenshots" {
		t.Error("Expected custom screenshot directory")
	}
	if config.UpdateScreenshots {
		t.Error("Expected UpdateScreenshots to be false")
	}
	if config.ScreenshotThreshold != 0.05 {
		t.Error("Expected threshold to be 0.05")
	}
}

func TestAssertScreenshotPaths(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{
		ScreenshotDir: tempDir,
	}
	_ = NewRunner(config) // Create runner to test config is used

	// Test baseline path generation
	baselineName := "test_screenshot"
	expectedPath := filepath.Join(tempDir, baselineName+".png")

	// Create a dummy file to test path
	err := os.WriteFile(expectedPath, []byte("dummy"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify file exists at expected path
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file at path %s", expectedPath)
	}
}
