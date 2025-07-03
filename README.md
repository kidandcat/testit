# FastTest

A simple and powerful browser testing framework built on [go-rod/rod](https://github.com/go-rod/rod) with a focus on DSL-based test files for easy, code-free testing.

## Features

- **DSL Test Files**: Write tests in simple, readable DSL files (no programming required)
- **Visual Testing**: Screenshot capture and comparison for visual regression testing
- **Rich Assertions**: Element existence, text content, URLs, attributes, and more
- **Console Error Detection**: Automatically fail tests when console errors occur
- **Configuration Files**: YAML/JSON config files for project-wide settings
- **Flexible CLI**: Run tests from command line with various options
- **Fast Execution**: Built on the efficient go-rod library

## Installation

```bash
go get github.com/fasttest/fasttest
```

## Quick Start

### Using DSL Test Files (Recommended)

Create a file `login.test`:

```
test "User can login"
  navigate "https://example.com/login"
  type "#username" "user@example.com"
  type "#password" "password123"
  click "#login-button"
  wait_for ".dashboard"
  assert_text ".welcome" "Welcome!"
```

Run it with the CLI:

```bash
fasttest login.test
```

## DSL Commands

### Navigation & Interaction
- `navigate "url"` - Navigate to a URL
- `click "selector"` - Click an element
- `type "selector" "text"` - Type text into an input
- `select "selector" "value"` - Select dropdown option
- `check "selector"` - Check a checkbox
- `uncheck "selector"` - Uncheck a checkbox
- `hover "selector"` - Hover over an element

### Waiting
- `wait_for "selector"` - Wait for an element to appear
- `wait_for_text "selector" "text"` - Wait for specific text in element
- `wait_for_url "pattern"` - Wait for URL to contain pattern

### Assertions
- `assert_text "selector" "expected"` - Assert exact text match
- `assert_text_contains "selector" "text"` - Assert text contains substring
- `assert_element_exists "selector"` - Assert element exists
- `assert_element_not_exists "selector"` - Assert element doesn't exist
- `assert_url "expected_url"` - Assert current URL
- `assert_title "expected_title"` - Assert page title
- `assert_attribute "selector" "attribute" "value"` - Assert attribute value

### Screenshots
- `screenshot` - Take a screenshot (auto-names with test name + number). If screenshot already exists, compares against it and fails if different
- `screenshot "filename"` - Take a screenshot with specific filename

## Configuration

### Configuration File

Create a `fasttest.config.yaml` in your project root:

```yaml
# Browser settings
headless: true

# Timeouts
timeout: 30s
actionTimeouts:
  navigate: 20s
  click: 10s
  type: 5s

# Console error handling
failOnConsoleError: true

# Screenshot settings
screenshotDir: "__screenshots__"
updateScreenshots: false
screenshotThreshold: 0.01  # 1% pixel difference allowed
```

Or use JSON format (`fasttest.config.json`):

```json
{
  "headless": true,
  "timeout": "30s",
  "screenshotDir": "__screenshots__",
  "screenshotThreshold": 0.01
}
```

FastTest automatically finds config files in this order:
- `fasttest.config.yaml`, `fasttest.config.yml`, `fasttest.config.json`
- `fasttest.yaml`, `fasttest.yml`, `fasttest.json`
- `.fasttest.yaml`, `.fasttest.yml`, `.fasttest.json`

## CLI Usage

```bash
# Run all .test files in current directory
fasttest

# Run specific files
fasttest login.test search.test

# Run with options
fasttest -headless=false -timeout=60s *.test

# Run tests in a directory
fasttest tests/
```

### CLI Options

- `-headless` (default: true) - Run browser in headless mode
- `-timeout` (default: 30s) - Test timeout duration
- `-fail-on-console-error` (default: true) - Fail tests when console errors occur
- `-pattern` (default: "*.test") - File pattern for test files
- `-config` - Path to config file (auto-detected if not specified)
- `-screenshot-dir` - Directory for screenshots
- `-update-screenshots` - Update baseline screenshots

## Advanced Usage

### Visual Regression Testing

Use screenshot command to catch visual regressions:

```
test "Homepage visual test"
  navigate "https://mysite.com"
  wait_for ".main-content"
  screenshot
```

First run creates baseline screenshots. Subsequent runs compare against baselines and fail if they differ.

To update baselines when intentional changes are made, delete the old screenshot file and run the test again
```

### Complex Form Testing

```
test "Complete form submission"
  navigate "https://mysite.com/form"
  
  # Fill text inputs
  type "#name" "John Doe"
  type "#email" "john@example.com"
  
  # Select from dropdown
  select "#country" "United States"
  
  # Check checkboxes
  check "#terms"
  check "#newsletter"
  
  # Hover and click submit
  hover "#submit-btn"
  screenshot "form-before-submit.png"
  click "#submit-btn"
  
  # Verify success
  wait_for ".success-message"
  assert_text_contains ".success-message" "Thank you"
```

### Testing Dynamic Content

```
test "Search autocomplete"
  navigate "https://mysite.com"
  
  # Type and wait for suggestions
  type "#search" "prod"
  wait_for_text ".suggestions" "Products"
  
  # Verify suggestion exists
  assert_element_exists ".suggestion-item"
  
  # Click first suggestion
  click ".suggestion-item:first-child"
  
  # Verify navigation
  wait_for_url "/search"
  assert_url "https://mysite.com/search?q=Products"
```

## Examples

See the `examples/` directory for complete examples:
- `simple.test` - Basic navigation test
- `login.test` - Login flow testing
- `search.test` - Search functionality testing  
- `advanced.test` - Demonstrates all DSL commands
- `fasttest.config.yaml` - Example configuration file

## Using as a Go Library

While FastTest is designed primarily for DSL-based testing, it can also be used as a Go library:

```go
package main

import (
    "fmt"
    "log"
    "github.com/fasttest/fasttest/pkg/fasttest"
)

func main() {
    runner := fasttest.New()
    if err := runner.Start(); err != nil {
        log.Fatal(err)
    }
    defer runner.Stop()
    
    result := runner.Test("My Test").
        Navigate("https://example.com").
        Click("#button").
        AssertText(".result", "Success").
        Run()
    
    fmt.Printf("Test passed: %v\n", result.Passed)
}
```

## License

MIT