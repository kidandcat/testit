package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fasttest/fasttest/pkg/config"
	"github.com/fasttest/fasttest/pkg/fasttest"
	"github.com/fasttest/fasttest/pkg/parser"
)

func main() {
	var (
		headless          = flag.Bool("headless", true, "Run browser in headless mode")
		timeout           = flag.Duration("timeout", 30*time.Second, "Test timeout")
		failOnConsoleError = flag.Bool("fail-on-console-error", true, "Fail tests when console errors occur")
		pattern           = flag.String("pattern", "*.test", "File pattern for test files")
		configFile        = flag.String("config", "", "Config file path")
		screenshotDir     = flag.String("screenshot-dir", "", "Screenshot directory")
		updateScreenshots = flag.Bool("update-screenshots", false, "Update baseline screenshots")
	)
	
	flag.Parse()
	
	// Start with default config
	runnerConfig := &fasttest.Config{
		Headless:          *headless,
		Timeout:           *timeout,
		FailOnConsoleError: *failOnConsoleError,
	}
	
	// Load config file if available
	configPath := *configFile
	if configPath == "" {
		configPath = config.FindConfigFile()
	}
	
	if configPath != "" {
		fileConfig, err := config.LoadConfig(configPath)
		if err != nil {
			log.Printf("Warning: Failed to load config file %s: %v", configPath, err)
		} else {
			// Apply file config (CLI flags override file config)
			if !isFlagSet("headless") && fileConfig.Headless != nil {
				runnerConfig.Headless = *fileConfig.Headless
			}
			if !isFlagSet("timeout") && fileConfig.Timeout != nil {
				runnerConfig.Timeout = fileConfig.Timeout.Duration
			}
			if !isFlagSet("fail-on-console-error") && fileConfig.FailOnConsoleError != nil {
				runnerConfig.FailOnConsoleError = *fileConfig.FailOnConsoleError
			}
			if fileConfig.ScreenshotDir != "" && *screenshotDir == "" {
				runnerConfig.ScreenshotDir = fileConfig.ScreenshotDir
			}
			if fileConfig.UpdateScreenshots && !*updateScreenshots {
				runnerConfig.UpdateScreenshots = fileConfig.UpdateScreenshots
			}
			runnerConfig.ScreenshotThreshold = fileConfig.ScreenshotThreshold
		}
	}
	
	// CLI flags override everything
	if *screenshotDir != "" {
		runnerConfig.ScreenshotDir = *screenshotDir
	}
	if *updateScreenshots {
		runnerConfig.UpdateScreenshots = true
	}
	
	runner := fasttest.NewRunner(runnerConfig)
	if err := runner.Start(); err != nil {
		log.Fatal("Failed to start browser:", err)
	}
	defer runner.Stop()
	
	testFiles, err := findTestFiles(*pattern, flag.Args())
	if err != nil {
		log.Fatal("Failed to find test files:", err)
	}
	
	if len(testFiles) == 0 {
		log.Fatal("No test files found")
	}
	
	p := parser.New()
	totalTests := 0
	
	for _, file := range testFiles {
		tests, err := p.ParseFile(file)
		if err != nil {
			log.Printf("Failed to parse %s: %v", file, err)
			continue
		}
		
		for _, test := range tests {
			runner.AddTest(test)
			totalTests++
		}
	}
	
	fmt.Printf("Running %d tests from %d files...\n\n", totalTests, len(testFiles))
	
	results := runner.Run()
	
	passed := 0
	failed := 0
	
	for _, result := range results {
		if result.Passed {
			fmt.Printf("✓ %s (%.2fs)\n", result.Name, result.Duration.Seconds())
			passed++
		} else {
			fmt.Printf("✗ %s (%.2fs)\n", result.Name, result.Duration.Seconds())
			if result.Error != nil {
				fmt.Printf("  Error: %v\n", result.Error)
			}
			if len(result.Errors) > 0 {
				fmt.Printf("  Console errors:\n")
				for _, err := range result.Errors {
					fmt.Printf("    - %s at %s\n", err.Message, err.URL)
				}
			}
			failed++
		}
	}
	
	fmt.Printf("\n%d passed, %d failed\n", passed, failed)
	
	if failed > 0 {
		os.Exit(1)
	}
}

func findTestFiles(pattern string, args []string) ([]string, error) {
	var files []string
	
	if len(args) > 0 {
		for _, arg := range args {
			if strings.HasSuffix(arg, ".test") || strings.HasSuffix(arg, ".fasttest") {
				files = append(files, arg)
			} else {
				matches, err := filepath.Glob(filepath.Join(arg, pattern))
				if err != nil {
					return nil, err
				}
				files = append(files, matches...)
			}
		}
	} else {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		files = matches
	}
	
	return files, nil
}

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}