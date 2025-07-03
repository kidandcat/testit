package fasttest

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type Runner struct {
	browser       *rod.Browser
	config        *Config
	tests         []Test
	results       []TestResult
	mu            sync.Mutex
	consoleErrors []ConsoleError
	screenshotCounter map[string]int
}

type Config struct {
	Headless          bool
	Timeout           time.Duration
	FailOnConsoleError bool
	ErrorFilter       func(error ConsoleError) bool
	ScreenshotDir     string
	UpdateScreenshots bool
	ScreenshotThreshold float64
}

type Test struct {
	Name  string
	Steps []Step
}

type Step struct {
	Action string
	Target string
	Value  string
}

type TestResult struct {
	Name     string
	Passed   bool
	Error    error
	Duration time.Duration
	Errors   []ConsoleError
}

type ConsoleError struct {
	Message   string
	Type      string
	Timestamp time.Time
	URL       string
}

func NewRunner(config *Config) *Runner {
	if config == nil {
		config = &Config{
			Headless:          true,
			Timeout:           30 * time.Second,
			FailOnConsoleError: true,
			ScreenshotDir:     "__screenshots__",
			ScreenshotThreshold: 0.0,
		}
	}
	if config.ScreenshotDir == "" {
		config.ScreenshotDir = "__screenshots__"
	}
	return &Runner{
		config: config,
		screenshotCounter: make(map[string]int),
	}
}

func (r *Runner) Start() error {
	l := launcher.New().Headless(r.config.Headless)
	url := l.MustLaunch()
	r.browser = rod.New().ControlURL(url).MustConnect()
	return nil
}

func (r *Runner) Stop() error {
	if r.browser != nil {
		return r.browser.Close()
	}
	return nil
}

func (r *Runner) AddTest(test Test) {
	r.tests = append(r.tests, test)
}

func (r *Runner) Run() []TestResult {
	r.results = make([]TestResult, 0, len(r.tests))
	
	for _, test := range r.tests {
		result := r.runTest(test)
		r.results = append(r.results, result)
	}
	
	return r.results
}

func (r *Runner) runTest(test Test) TestResult {
	start := time.Now()
	result := TestResult{
		Name:   test.Name,
		Passed: true,
	}
	
	page := r.browser.MustPage()
	defer page.MustClose()
	
	page = page.Timeout(r.config.Timeout)
	
	r.setupConsoleListener(page, &result)
	
	for _, step := range test.Steps {
		if err := r.executeStep(page, step, test.Name); err != nil {
			result.Passed = false
			result.Error = err
			break
		}
	}
	
	if r.config.FailOnConsoleError && len(result.Errors) > 0 {
		result.Passed = false
		if result.Error == nil {
			result.Error = fmt.Errorf("console errors detected: %d errors", len(result.Errors))
		}
	}
	
	result.Duration = time.Since(start)
	return result
}

func (r *Runner) setupConsoleListener(page *rod.Page, result *TestResult) {
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		if e.Type == proto.RuntimeConsoleAPICalledTypeError {
			consoleErr := ConsoleError{
				Message:   e.Args[0].Preview.Properties[0].Value,
				Type:      string(e.Type),
				Timestamp: time.Now(),
				URL:       page.MustInfo().URL,
			}
			
			if r.config.ErrorFilter == nil || !r.config.ErrorFilter(consoleErr) {
				r.mu.Lock()
				result.Errors = append(result.Errors, consoleErr)
				r.mu.Unlock()
			}
		}
	})()
}

func (r *Runner) executeStep(page *rod.Page, step Step, testName string) error {
	switch step.Action {
	case "navigate":
		return page.Navigate(step.Target)
	case "click":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		return element.Click(proto.InputMouseButtonLeft, 1)
	case "type":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		return element.Input(step.Value)
	case "wait_for":
		_, err := page.Element(step.Target)
		return err
	case "assert_text":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		text, err := element.Text()
		if err != nil {
			return err
		}
		if text != step.Value {
			return fmt.Errorf("expected text '%s', got '%s'", step.Value, text)
		}
		return nil
		
	case "assert_element_exists":
		_, err := page.Element(step.Target)
		if err != nil {
			return fmt.Errorf("element not found: %s", step.Target)
		}
		return nil
		
	case "assert_element_not_exists":
		_, err := page.Element(step.Target)
		if err == nil {
			return fmt.Errorf("element should not exist: %s", step.Target)
		}
		return nil
		
	case "assert_text_contains":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		text, err := element.Text()
		if err != nil {
			return err
		}
		if !strings.Contains(text, step.Value) {
			return fmt.Errorf("expected text to contain '%s', got '%s'", step.Value, text)
		}
		return nil
		
	case "assert_url":
		currentURL := page.MustInfo().URL
		if currentURL != step.Target {
			return fmt.Errorf("expected URL '%s', got '%s'", step.Target, currentURL)
		}
		return nil
		
	case "assert_title":
		title := page.MustInfo().Title
		if title != step.Target {
			return fmt.Errorf("expected title '%s', got '%s'", step.Target, title)
		}
		return nil
		
	case "assert_attribute":
		// Target format: "selector|attribute"
		parts := strings.Split(step.Target, "|")
		if len(parts) != 2 {
			return fmt.Errorf("invalid assert_attribute format")
		}
		selector, attribute := parts[0], parts[1]
		element, err := page.Element(selector)
		if err != nil {
			return err
		}
		value, err := element.Attribute(attribute)
		if err != nil {
			return err
		}
		if value == nil || *value != step.Value {
			actual := "<nil>"
			if value != nil {
				actual = *value
			}
			return fmt.Errorf("expected attribute '%s' to be '%s', got '%s'", attribute, step.Value, actual)
		}
		return nil
		
	case "screenshot":
		return r.takeScreenshot(page, step.Target, testName)
		
	case "wait_for_text":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		err = element.WaitStable(r.config.Timeout)
		if err != nil {
			return err
		}
		text, err := element.Text()
		if err != nil {
			return err
		}
		if !strings.Contains(text, step.Value) {
			return fmt.Errorf("timeout waiting for text '%s' in element", step.Value)
		}
		return nil
		
	case "wait_for_url":
		start := time.Now()
		for {
			currentURL := page.MustInfo().URL
			if strings.Contains(currentURL, step.Target) {
				return nil
			}
			if time.Since(start) > r.config.Timeout {
				return fmt.Errorf("timeout waiting for URL to contain '%s'", step.Target)
			}
			time.Sleep(100 * time.Millisecond)
		}
		
	case "select":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		return element.Select([]string{step.Value}, true, rod.SelectorTypeText)
		
	case "check":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		// Click the checkbox to check it
		return element.Click(proto.InputMouseButtonLeft, 1)
		
	case "uncheck":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		// Click the checkbox to uncheck it
		return element.Click(proto.InputMouseButtonLeft, 1)
		
	case "hover":
		element, err := page.Element(step.Target)
		if err != nil {
			return err
		}
		return element.Hover()
		
	default:
		return fmt.Errorf("unknown action: %s", step.Action)
	}
}

func (r *Runner) takeScreenshot(page *rod.Page, filename string, testName string) error {
	if filename == "" {
		// Sanitize test name for filename
		safeTestName := strings.ReplaceAll(testName, " ", "_")
		safeTestName = strings.ReplaceAll(safeTestName, "/", "_")
		safeTestName = strings.ReplaceAll(safeTestName, "\\", "_")
		
		// Get counter for this test
		r.mu.Lock()
		r.screenshotCounter[testName]++
		counter := r.screenshotCounter[testName]
		r.mu.Unlock()
		
		if counter == 1 {
			filename = fmt.Sprintf("%s.png", safeTestName)
		} else {
			filename = fmt.Sprintf("%s_%d.png", safeTestName, counter)
		}
	}
	
	// Create screenshot directory if it doesn't exist
	err := os.MkdirAll(r.config.ScreenshotDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create screenshot directory: %v", err)
	}
	
	// Take current screenshot
	screenshot, err := page.Screenshot(true, nil)
	if err != nil {
		return fmt.Errorf("failed to take screenshot: %v", err)
	}
	
	path := filepath.Join(r.config.ScreenshotDir, filename)
	
	// Check if screenshot already exists
	if _, err := os.Stat(path); err == nil {
		// Screenshot exists, load and compare
		baselineData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read existing screenshot: %v", err)
		}
		
		// Compare screenshots
		diff, err := r.compareImages(baselineData, screenshot)
		if err != nil {
			return fmt.Errorf("failed to compare screenshots: %v", err)
		}
		
		if diff > r.config.ScreenshotThreshold {
			// Save diff screenshot for debugging
			diffPath := strings.TrimSuffix(path, ".png") + ".diff.png"
			os.WriteFile(diffPath, screenshot, 0644)
			
			return fmt.Errorf("screenshot differs from baseline by %.2f%% (threshold: %.2f%%). Delete the old screenshot at %s to save the new one", diff*100, r.config.ScreenshotThreshold*100, path)
		}
		
		// Screenshots match, no need to save
		return nil
	}
	
	// Screenshot doesn't exist, save it
	err = os.WriteFile(path, screenshot, 0644)
	if err != nil {
		return fmt.Errorf("failed to save screenshot: %v", err)
	}
	
	return nil
}


func (r *Runner) compareImages(baseline, current []byte) (float64, error) {
	baselineImg, err := png.Decode(bytes.NewReader(baseline))
	if err != nil {
		return 0, err
	}
	
	currentImg, err := png.Decode(bytes.NewReader(current))
	if err != nil {
		return 0, err
	}
	
	// Simple pixel-by-pixel comparison
	bounds := baselineImg.Bounds()
	if bounds != currentImg.Bounds() {
		return 1.0, nil // 100% different if sizes don't match
	}
	
	totalPixels := bounds.Dx() * bounds.Dy()
	differentPixels := 0
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c1 := baselineImg.At(x, y)
			c2 := currentImg.At(x, y)
			if !colorsEqual(c1, c2) {
				differentPixels++
			}
		}
	}
	
	return float64(differentPixels) / float64(totalPixels), nil
}

func colorsEqual(c1, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}