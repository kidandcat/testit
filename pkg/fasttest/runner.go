package fasttest

import (
	"bytes"
	"context"
	"fmt"
	"image"
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
	allocCtx          context.Context
	allocCancel       context.CancelFunc
	config            *Config
	tests             []Test
	results           []TestResult
	mu                sync.Mutex
	consoleErrors     []ConsoleError
	screenshotCounter map[string]int
	snapshotCounter   map[string]int
}

type Config struct {
	Headless            bool
	Timeout             time.Duration
	FailOnConsoleError  bool
	ErrorFilter         func(error ConsoleError) bool
	ScreenshotDir       string
	UpdateScreenshots   bool
	ScreenshotThreshold float64
	SnapshotDir         string
	UpdateSnapshots     bool
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
			Headless:            true,
			Timeout:             10 * time.Second, // Reduced default timeout
			FailOnConsoleError:  true,
			ScreenshotDir:       "__screenshots__",
			ScreenshotThreshold: 0.0,
			SnapshotDir:         "__snapshots__",
			UpdateSnapshots:     false,
		}
	}
	if config.ScreenshotDir == "" {
		config.ScreenshotDir = "__screenshots__"
	}
	if config.SnapshotDir == "" {
		config.SnapshotDir = "__snapshots__"
	}
	return &Runner{
		config:            config,
		screenshotCounter: make(map[string]int),
		snapshotCounter:   make(map[string]int),
	}
}

func (r *Runner) Start() error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", r.config.Headless),
		chromedp.Flag("disable-gpu", r.config.Headless),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true), // Use /tmp instead of /dev/shm
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-features", "site-per-process"), // Faster navigation
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

	// Run tests with parallel execution
	testChan := make(chan Test, len(r.tests))
	resultChan := make(chan TestResult, len(r.tests))

	// Start workers
	numWorkers := 4
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for test := range testChan {
				result := r.runTest(test)
				resultChan <- result
			}
		}()
	}

	// Add tests to channel
	for _, test := range r.tests {
		testChan <- test
	}
	close(testChan)

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
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

	// Create a new browser context for this test with timeout
	ctx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	// Apply timeout from config
	ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	// Initialize the browser by navigating to about:blank
	if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
		result.Passed = false
		result.Error = fmt.Errorf("failed to initialize browser: %v", err)
		return result
	}

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

	case "snapshot":
		return r.takeSnapshot(ctx, step.Target, testName)

	case "wait_for_text":
		// First wait for element to be visible
		if err := chromedp.Run(ctx, chromedp.WaitVisible(step.Target)); err != nil {
			return err
		}
		// Then poll for text content with faster polling
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
			}, chromedp.WithPollingInterval(50*time.Millisecond)), // Reduced polling interval
		)

	case "wait_for_url":
		// Poll for URL to match with faster polling
		ticker := time.NewTicker(50 * time.Millisecond) // Reduced polling interval
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

	case "assert_text_visible":
		// First wait for the body element to be ready
		if err := chromedp.Run(ctx, chromedp.WaitReady("body")); err != nil {
			return fmt.Errorf("failed to wait for body element: %v", err)
		}

		// Get all visible text from the body element
		var text string
		err := chromedp.Run(ctx, chromedp.Text("body", &text))
		if err != nil {
			return fmt.Errorf("failed to get text from body: %v", err)
		}
		// Check if the expected text is contained anywhere in the page
		if !strings.Contains(text, step.Value) {
			return fmt.Errorf("text '%s' not found on page", step.Value)
		}
		return nil

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
		diff, diffImage, err := r.compareImages(baselineData, screenshot)
		if err != nil {
			return fmt.Errorf("failed to compare screenshots: %v", err)
		}

		if diff > r.config.ScreenshotThreshold {
			// Save the actual screenshot for reference
			actualPath := strings.TrimSuffix(path, ".png") + ".actual.png"
			os.WriteFile(actualPath, screenshot, 0644)

			// Save diff image showing the differences
			if diffImage != nil {
				diffPath := strings.TrimSuffix(path, ".png") + ".diff.png"
				os.WriteFile(diffPath, diffImage, 0644)
			}

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

func (r *Runner) compareImages(baseline, current []byte) (float64, []byte, error) {
	baselineImg, err := png.Decode(bytes.NewReader(baseline))
	if err != nil {
		return 0, nil, err
	}

	currentImg, err := png.Decode(bytes.NewReader(current))
	if err != nil {
		return 0, nil, err
	}

	// Quick check for exact match first
	if bytes.Equal(baseline, current) {
		return 0, nil, nil
	}

	bounds := baselineImg.Bounds()
	if bounds != currentImg.Bounds() {
		return 1.0, nil, nil // 100% different if sizes don't match
	}

	totalPixels := bounds.Dx() * bounds.Dy()
	differentPixels := 0

	// Create diff image only if needed
	var diffImg *image.RGBA
	var needsDiff bool

	// Sample comparison first - check every 10th pixel for quick estimation
	sampleStep := 10
	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleStep {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleStep {
			c1 := baselineImg.At(x, y)
			c2 := currentImg.At(x, y)
			if !colorsEqual(c1, c2) {
				needsDiff = true
				break
			}
		}
		if needsDiff {
			break
		}
	}

	// Only do full comparison if sample shows differences
	if !needsDiff {
		return 0, nil, nil
	}

	diffImg = image.NewRGBA(bounds)

	// Parallel processing for large images
	numWorkers := 4
	rowsPerWorker := bounds.Dy() / numWorkers
	var wg sync.WaitGroup
	var mu sync.Mutex

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		startY := bounds.Min.Y + w*rowsPerWorker
		endY := startY + rowsPerWorker
		if w == numWorkers-1 {
			endY = bounds.Max.Y
		}

		go func(startY, endY int) {
			defer wg.Done()
			localDiff := 0

			for y := startY; y < endY; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					c1 := baselineImg.At(x, y)
					c2 := currentImg.At(x, y)
					if !colorsEqual(c1, c2) {
						localDiff++
						// Highlight differences in red
						diffImg.Set(x, y, color.RGBA{255, 0, 0, 255})
					} else {
						// Show matching pixels as grayscale from baseline
						r1, g1, b1, _ := c1.RGBA()
						gray := uint8((r1 + g1 + b1) / 3 / 256)
						diffImg.Set(x, y, color.RGBA{gray, gray, gray, 128})
					}
				}
			}

			mu.Lock()
			differentPixels += localDiff
			mu.Unlock()
		}(startY, endY)
	}

	wg.Wait()

	// Encode diff image
	var diffBuf bytes.Buffer
	if err := png.Encode(&diffBuf, diffImg); err != nil {
		return 0, nil, err
	}

	return float64(differentPixels) / float64(totalPixels), diffBuf.Bytes(), nil
}

func colorsEqual(c1, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

func (r *Runner) takeSnapshot(ctx context.Context, filename string, testName string) error {
	if filename == "" {
		// Sanitize test name for filename
		safeTestName := strings.ReplaceAll(testName, " ", "_")
		safeTestName = strings.ReplaceAll(safeTestName, "/", "_")
		safeTestName = strings.ReplaceAll(safeTestName, "\\", "_")

		// Get counter for this test
		r.mu.Lock()
		r.snapshotCounter[testName]++
		counter := r.snapshotCounter[testName]
		r.mu.Unlock()

		if counter == 1 {
			filename = fmt.Sprintf("%s.html", safeTestName)
		} else {
			filename = fmt.Sprintf("%s_%d.html", safeTestName, counter)
		}
	}

	// Create snapshot directory if it doesn't exist
	err := os.MkdirAll(r.config.SnapshotDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create snapshot directory: %v", err)
	}

	// Capture current HTML
	var html string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.documentElement.outerHTML`, &html),
	)
	if err != nil {
		return fmt.Errorf("failed to capture HTML: %v", err)
	}

	path := filepath.Join(r.config.SnapshotDir, filename)

	// Check if snapshot already exists
	if _, err := os.Stat(path); err == nil {
		// Snapshot exists, load and compare
		baselineData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read existing snapshot: %v", err)
		}

		// Compare snapshots
		if !r.compareSnapshots(string(baselineData), html) {
			// Save the actual snapshot for reference
			actualPath := strings.TrimSuffix(path, ".html") + ".actual.html"
			os.WriteFile(actualPath, []byte(html), 0644)

			// Generate and save diff
			diffHTML := r.generateHTMLDiff(string(baselineData), html)
			diffPath := strings.TrimSuffix(path, ".html") + ".diff.html"
			os.WriteFile(diffPath, []byte(diffHTML), 0644)

			return fmt.Errorf("snapshot differs from baseline. Delete the old snapshot at %s to save the new one", path)
		}

		// Snapshots match, no need to save
		return nil
	}

	// Snapshot doesn't exist, save it
	err = os.WriteFile(path, []byte(html), 0644)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %v", err)
	}

	return nil
}

func (r *Runner) compareSnapshots(baseline, current string) bool {
	// Normalize HTML for comparison
	baseline = r.normalizeHTML(baseline)
	current = r.normalizeHTML(current)

	return baseline == current
}

func (r *Runner) normalizeHTML(html string) string {
	// Remove extra whitespace between tags
	html = strings.ReplaceAll(html, "\n", " ")
	html = strings.ReplaceAll(html, "\r", " ")
	html = strings.ReplaceAll(html, "\t", " ")

	// Collapse multiple spaces into single space
	for strings.Contains(html, "  ") {
		html = strings.ReplaceAll(html, "  ", " ")
	}

	// Remove spaces between tags
	html = strings.ReplaceAll(html, "> <", "><")
	html = strings.ReplaceAll(html, "> ", ">")
	html = strings.ReplaceAll(html, " <", "<")

	return strings.TrimSpace(html)
}

func (r *Runner) generateHTMLDiff(baseline, current string) string {
	// Simple diff visualization
	// In a real implementation, you might want to use a proper diff library
	diffHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Snapshot Diff</title>
    <style>
        body { font-family: monospace; white-space: pre-wrap; }
        .added { background-color: #90EE90; }
        .removed { background-color: #FFB6C1; }
        .header { font-weight: bold; margin: 20px 0 10px 0; }
    </style>
</head>
<body>
    <div class="header">Snapshot Diff</div>
    <div class="header">Expected:</div>
    <div class="removed">` + escapeHTML(r.normalizeHTML(baseline)) + `</div>
    <div class="header">Actual:</div>
    <div class="added">` + escapeHTML(r.normalizeHTML(current)) + `</div>
</body>
</html>`

	return diffHTML
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

func (r *Runner) RunWithProgress(resultsChan chan<- TestResult, wg *sync.WaitGroup) []TestResult {
	r.results = make([]TestResult, 0, len(r.tests))

	testChan := make(chan Test, len(r.tests))
	resultCollector := make(chan TestResult, len(r.tests))

	// Use 4 parallel workers
	numWorkers := 4

	var workerWg sync.WaitGroup
	workerWg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer workerWg.Done()
			for test := range testChan {
				result := r.runTest(test)
				resultCollector <- result
			}
		}()
	}

	// Add tests to channel
	for _, test := range r.tests {
		testChan <- test
	}
	close(testChan)

	// Wait for all workers to complete
	go func() {
		workerWg.Wait()
		close(resultCollector)
	}()

	// Collect results and forward to UI
	for result := range resultCollector {
		r.results = append(r.results, result)
		wg.Add(1)
		resultsChan <- result
	}

	return r.results
}
