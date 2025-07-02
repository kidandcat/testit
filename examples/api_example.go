package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fasttest/fasttest/pkg/fasttest"
)

func main() {
	config := &fasttest.Config{
		Headless:          false,
		Timeout:           20 * time.Second,
		FailOnConsoleError: true,
		ErrorFilter: func(err fasttest.ConsoleError) bool {
			return false
		},
	}
	
	runner := fasttest.WithConfig(config)
	
	if err := runner.Start(); err != nil {
		log.Fatal("Failed to start browser:", err)
	}
	defer runner.Stop()
	
	result := runner.Test("Google Search").
		Navigate("https://www.google.com").
		Type("input[name='q']", "go-rod testing").
		Click("input[name='btnK']").
		WaitFor("#search").
		Run()
	
	if result.Passed {
		fmt.Printf("Test passed in %.2fs\n", result.Duration.Seconds())
	} else {
		fmt.Printf("Test failed: %v\n", result.Error)
		if len(result.Errors) > 0 {
			fmt.Println("Console errors:")
			for _, err := range result.Errors {
				fmt.Printf("  - %s\n", err.Message)
			}
		}
	}
	
	runner2 := fasttest.New()
	if err := runner2.Start(); err != nil {
		log.Fatal("Failed to start browser:", err)
	}
	defer runner2.Stop()
	
	runner2.Test("Test 1").
		Navigate("https://example.com").
		Click("#button1").
		Add()
	
	runner2.Test("Test 2").
		Navigate("https://example.com/page2").
		Type("#input", "test").
		Add()
	
	results := runner2.Run()
	
	for _, result := range results {
		fmt.Printf("%s: %v\n", result.Name, result.Passed)
	}
}