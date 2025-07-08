package fasttest

import (
	"context"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type TestBuilder struct {
	runner *Runner
	test   Test
}

type StepBuilder struct {
	builder *TestBuilder
	ctx     context.Context
}

func New() *Runner {
	return NewRunner(nil)
}

func WithConfig(config *Config) *Runner {
	return NewRunner(config)
}

func (r *Runner) Test(name string) *TestBuilder {
	return &TestBuilder{
		runner: r,
		test: Test{
			Name: name,
		},
	}
}

func (tb *TestBuilder) Navigate(url string) *TestBuilder {
	tb.test.Steps = append(tb.test.Steps, Step{
		Action: "navigate",
		Target: url,
	})
	return tb
}

func (tb *TestBuilder) Click(selector string) *TestBuilder {
	tb.test.Steps = append(tb.test.Steps, Step{
		Action: "click",
		Target: selector,
	})
	return tb
}

func (tb *TestBuilder) Type(selector, text string) *TestBuilder {
	tb.test.Steps = append(tb.test.Steps, Step{
		Action: "type",
		Target: selector,
		Value:  text,
	})
	return tb
}

func (tb *TestBuilder) WaitFor(selector string) *TestBuilder {
	tb.test.Steps = append(tb.test.Steps, Step{
		Action: "wait_for",
		Target: selector,
	})
	return tb
}

func (tb *TestBuilder) AssertText(selector, expected string) *TestBuilder {
	tb.test.Steps = append(tb.test.Steps, Step{
		Action: "assert_text",
		Target: selector,
		Value:  expected,
	})
	return tb
}

func (tb *TestBuilder) AssertTextVisible(expected string) *TestBuilder {
	tb.test.Steps = append(tb.test.Steps, Step{
		Action: "assert_text_visible",
		Value:  expected,
	})
	return tb
}

func (tb *TestBuilder) Run() TestResult {
	tb.runner.AddTest(tb.test)
	results := tb.runner.Run()
	if len(results) > 0 {
		return results[0]
	}
	return TestResult{
		Name:   tb.test.Name,
		Passed: false,
		Error:  ErrNoTestResults,
	}
}

func (tb *TestBuilder) Add() *TestBuilder {
	tb.runner.AddTest(tb.test)
	return tb
}

type PageTester struct {
	ctx     context.Context
	result  *TestResult
	timeout time.Duration
}

func NewPageTester(ctx context.Context, timeout time.Duration) *PageTester {
	return &PageTester{
		ctx:     ctx,
		timeout: timeout,
		result: &TestResult{
			Passed: true,
		},
	}
}

func (pt *PageTester) Navigate(url string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}

	timeoutCtx, cancel := context.WithTimeout(pt.ctx, pt.timeout)
	defer cancel()

	pt.result.Error = chromedp.Navigate(url).Do(timeoutCtx)
	return pt
}

func (pt *PageTester) Click(selector string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}

	timeoutCtx, cancel := context.WithTimeout(pt.ctx, pt.timeout)
	defer cancel()

	pt.result.Error = chromedp.Click(selector, chromedp.NodeVisible).Do(timeoutCtx)
	return pt
}

func (pt *PageTester) Type(selector, text string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}

	timeoutCtx, cancel := context.WithTimeout(pt.ctx, pt.timeout)
	defer cancel()

	pt.result.Error = chromedp.SendKeys(selector, text, chromedp.NodeVisible).Do(timeoutCtx)
	return pt
}

func (pt *PageTester) WaitFor(selector string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}

	timeoutCtx, cancel := context.WithTimeout(pt.ctx, pt.timeout)
	defer cancel()

	pt.result.Error = chromedp.WaitVisible(selector).Do(timeoutCtx)
	return pt
}

func (pt *PageTester) AssertText(selector, expected string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}

	timeoutCtx, cancel := context.WithTimeout(pt.ctx, pt.timeout)
	defer cancel()

	var text string
	err := chromedp.Text(selector, &text, chromedp.NodeVisible).Do(timeoutCtx)
	if err != nil {
		pt.result.Error = err
		return pt
	}

	if text != expected {
		pt.result.Error = &AssertionError{
			Expected: expected,
			Actual:   text,
			Message:  "text assertion failed",
		}
	}
	return pt
}

func (pt *PageTester) AssertTextVisible(expected string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}

	timeoutCtx, cancel := context.WithTimeout(pt.ctx, pt.timeout)
	defer cancel()

	var text string
	// Get all visible text from the body element
	err := chromedp.Text("body", &text, chromedp.NodeVisible).Do(timeoutCtx)
	if err != nil {
		pt.result.Error = err
		return pt
	}

	// Check if the expected text is contained anywhere in the page
	if !strings.Contains(text, expected) {
		pt.result.Error = &AssertionError{
			Expected: expected,
			Actual:   "[text not found in page]",
			Message:  "text visibility assertion failed",
		}
	}
	return pt
}

func (pt *PageTester) Result() TestResult {
	if pt.result.Error != nil {
		pt.result.Passed = false
	}
	return *pt.result
}
