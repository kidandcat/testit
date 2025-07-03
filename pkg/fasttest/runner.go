package fasttest

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type Runner struct {
	allocCtx      context.Context
	allocCancel   context.CancelFunc
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
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", r.config.Headless),
		chromedp.Flag("disable-gpu", r.config.Headless),
		chromedp.Flag("no-sandbox", true),
	)
	
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	r.allocCtx = allocCtx
	r.allocCancel = cancel
	
	return nil
}

func (r *Runner) Stop() error {
	if r.allocCancel != nil {
		r.allocCancel()
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
		Errors: []ConsoleError{},
	}
	
	// Check if browser has been started
	if r.allocCtx == nil {
		result.Passed = false
		result.Error = fmt.Errorf("browser not started")
		return result
	}
	
	// Create a new browser context for this test
	ctx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()
	
	// Initialize the browser by starting it
	// We don't need to navigate to about:blank - chromedp handles this
	
	// Set up console listener
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			if ev.Type == runtime.APITypeError {
				var message string
				if len(ev.Args) > 0 && ev.Args[0].Value != nil {
					message = string(ev.Args[0].Value)
				}
				
				consoleErr := ConsoleError{
					Message:   message,
					Type:      string(ev.Type),
					Timestamp: time.Now(),
				}
				
				// Get current URL
				var url string
				chromedp.Run(ctx, chromedp.Location(&url))
				consoleErr.URL = url
				
				if r.config.ErrorFilter == nil || !r.config.ErrorFilter(consoleErr) {
					r.mu.Lock()
					result.Errors = append(result.Errors, consoleErr)
					r.mu.Unlock()
				}
			}
		}
	})
	
	// Run steps
	for _, step := range test.Steps {
		if err := r.executeStep(ctx, step, test.Name); err != nil {
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

func (r *Runner) executeStep(ctx context.Context, step Step, testName string) error {
	switch step.Action {
	case "navigate":
		return chromedp.Run(ctx, chromedp.Navigate(step.Target))
		
	case "click":
		return chromedp.Run(ctx, chromedp.Click(step.Target, chromedp.NodeVisible))
		
	case "type":
		return chromedp.Run(ctx, chromedp.SendKeys(step.Target, step.Value, chromedp.NodeVisible))
		
	case "wait_for":
		return chromedp.Run(ctx, chromedp.WaitVisible(step.Target))
		
	case "assert_text":
		var text string
		err := chromedp.Run(ctx, chromedp.Text(step.Target, &text, chromedp.NodeVisible))
		if err != nil {
			return err
		}
		if text != step.Value {
			return fmt.Errorf("expected text '%s', got '%s'", step.Value, text)
		}
		return nil
		
	case "assert_element_exists":
		var nodes []*cdp.Node
		err := chromedp.Run(ctx, chromedp.Nodes(step.Target, &nodes))
		if err != nil || len(nodes) == 0 {
			return fmt.Errorf("element not found: %s", step.Target)
		}
		return nil
		
	case "assert_element_not_exists":
		var nodes []*cdp.Node
		err := chromedp.Run(ctx, chromedp.Nodes(step.Target, &nodes))
		if err == nil && len(nodes) > 0 {
			return fmt.Errorf("element should not exist: %s", step.Target)
		}
		return nil
		
	case "assert_text_contains":
		var text string
		err := chromedp.Run(ctx, chromedp.Text(step.Target, &text, chromedp.NodeVisible))
		if err != nil {
			return err
		}
		if !strings.Contains(text, step.Value) {
			return fmt.Errorf("expected text to contain '%s', got '%s'", step.Value, text)
		}
		return nil
		
	case "assert_url":
		var currentURL string
		err := chromedp.Run(ctx, chromedp.Location(&currentURL))
		if err != nil {
			return err
		}
		if currentURL != step.Target {
			return fmt.Errorf("expected URL '%s', got '%s'", step.Target, currentURL)
		}
		return nil
		
	case "assert_title":
		var title string
		err := chromedp.Run(ctx, chromedp.Title(&title))
		if err != nil {
			return err
		}
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
		
		var value string
		var ok bool
		err := chromedp.Run(ctx, chromedp.AttributeValue(selector, attribute, &value, &ok))
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("attribute '%s' not found", attribute)
		}
		if value != step.Value {
			return fmt.Errorf("expected attribute '%s' to be '%s', got '%s'", attribute, step.Value, value)
		}
		return nil
		
	case "screenshot":
		return r.takeScreenshot(ctx, step.Target, testName)
		
	case "wait_for_text":
		// First wait for element to be visible
		if err := chromedp.Run(ctx, chromedp.WaitVisible(step.Target)); err != nil {
			return err
		}
		// Then poll for text content
		return chromedp.Run(ctx,
			chromedp.Poll(step.Target, func(ctx context.Context, node *runtime.RemoteObject) error {
				var text string
				if err := chromedp.Run(ctx, chromedp.Text(step.Target, &text, chromedp.NodeVisible)); err != nil {
					return err
				}
				if strings.Contains(text, step.Value) {
					return nil
				}
				return fmt.Errorf("waiting for text")
			}, chromedp.WithPollingInterval(100*time.Millisecond)),
		)
		
	case "wait_for_url":
		// Poll for URL to match
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		deadline, ok := ctx.Deadline()
		if !ok {
			deadline = time.Now().Add(r.config.Timeout)
		}
		
		for time.Now().Before(deadline) {
			var currentURL string
			if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err == nil {
				if strings.Contains(currentURL, step.Target) {
					return nil
				}
			}
			select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				return fmt.Errorf("timeout waiting for URL to contain '%s'", step.Target)
			}
		}
		return fmt.Errorf("timeout waiting for URL to contain '%s'", step.Target)
		
	case "select":
		// First click to open dropdown
		if err := chromedp.Run(ctx, chromedp.Click(step.Target, chromedp.NodeVisible)); err != nil {
			return err
		}
		// Then select the option
		return chromedp.Run(ctx, chromedp.SetValue(step.Target, step.Value))
		
	case "check":
		// Click checkbox to check it
		return chromedp.Run(ctx, chromedp.Click(step.Target, chromedp.NodeVisible))
		
	case "uncheck":
		// Click checkbox to uncheck it
		return chromedp.Run(ctx, chromedp.Click(step.Target, chromedp.NodeVisible))
		
	case "hover":
		// Move mouse over element
		var nodes []*cdp.Node
		if err := chromedp.Run(ctx, chromedp.Nodes(step.Target, &nodes)); err != nil {
			return err
		}
		if len(nodes) == 0 {
			return fmt.Errorf("element not found: %s", step.Target)
		}
		return chromedp.Run(ctx, chromedp.MouseClickNode(nodes[0]))
		
	default:
		return fmt.Errorf("unknown action: %s", step.Action)
	}
}

func (r *Runner) takeScreenshot(ctx context.Context, filename string, testName string) error {
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
	var screenshot []byte
	err = chromedp.Run(ctx,
		chromedp.FullScreenshot(&screenshot, 100),
	)
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