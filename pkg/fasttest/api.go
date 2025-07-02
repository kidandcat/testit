package fasttest

import (
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type TestBuilder struct {
	runner *Runner
	test   Test
}

type StepBuilder struct {
	builder *TestBuilder
	page    *rod.Page
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
	page    *rod.Page
	result  *TestResult
	timeout time.Duration
}

func NewPageTester(page *rod.Page, timeout time.Duration) *PageTester {
	return &PageTester{
		page:    page,
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
	pt.result.Error = pt.page.Timeout(pt.timeout).Navigate(url)
	return pt
}

func (pt *PageTester) Click(selector string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}
	element, err := pt.page.Timeout(pt.timeout).Element(selector)
	if err != nil {
		pt.result.Error = err
		return pt
	}
	pt.result.Error = element.Click(proto.InputMouseButtonLeft, 1)
	return pt
}

func (pt *PageTester) Type(selector, text string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}
	element, err := pt.page.Timeout(pt.timeout).Element(selector)
	if err != nil {
		pt.result.Error = err
		return pt
	}
	pt.result.Error = element.Input(text)
	return pt
}

func (pt *PageTester) WaitFor(selector string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}
	_, err := pt.page.Timeout(pt.timeout).Element(selector)
	pt.result.Error = err
	return pt
}

func (pt *PageTester) AssertText(selector, expected string) *PageTester {
	if pt.result.Error != nil {
		return pt
	}
	element, err := pt.page.Timeout(pt.timeout).Element(selector)
	if err != nil {
		pt.result.Error = err
		return pt
	}
	text, err := element.Text()
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

func (pt *PageTester) Result() TestResult {
	if pt.result.Error != nil {
		pt.result.Passed = false
	}
	return *pt.result
}