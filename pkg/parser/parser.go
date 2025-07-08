package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kidandcat/fasttest/pkg/fasttest"
)

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) ParseFile(filename string) ([]fasttest.Test, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	return p.parse(scanner)
}

func (p *Parser) ParseString(content string) ([]fasttest.Test, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	return p.parse(scanner)
}

func (p *Parser) parse(scanner *bufio.Scanner) ([]fasttest.Test, error) {
	var tests []fasttest.Test
	var currentTest *fasttest.Test
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "test ") {
			if currentTest != nil {
				tests = append(tests, *currentTest)
			}

			testNamePart := strings.TrimPrefix(line, "test ")
			testName := strings.Trim(testNamePart, `"'`)
			currentTest = &fasttest.Test{
				Name: testName,
			}
		} else if currentTest != nil {
			step, err := p.parseLine(line, lineNum)
			if err != nil {
				return nil, err
			}
			if step != nil {
				currentTest.Steps = append(currentTest.Steps, *step)
			}
		}
	}

	if currentTest != nil {
		tests = append(tests, *currentTest)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tests, nil
}

func (p *Parser) parseLine(line string, lineNum int) (*fasttest.Step, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil, nil
	}

	action := parts[0]

	switch action {
	case "navigate":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: navigate requires a URL", lineNum)
		}
		return &fasttest.Step{
			Action: "navigate",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "click":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: click requires a selector", lineNum)
		}
		return &fasttest.Step{
			Action: "click",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "type":
		if len(parts) < 3 {
			return nil, fmt.Errorf("line %d: type requires a selector and value", lineNum)
		}
		selector := strings.Trim(parts[1], `"'`)
		value := strings.Trim(strings.Join(parts[2:], " "), `"'`)
		return &fasttest.Step{
			Action: "type",
			Target: selector,
			Value:  value,
		}, nil

	case "wait_for":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: wait_for requires a selector", lineNum)
		}
		return &fasttest.Step{
			Action: "wait_for",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "assert_text":
		if len(parts) < 3 {
			return nil, fmt.Errorf("line %d: assert_text requires a selector and expected text", lineNum)
		}
		selector := strings.Trim(parts[1], `"'`)
		expectedText := strings.Trim(strings.Join(parts[2:], " "), `"'`)
		return &fasttest.Step{
			Action: "assert_text",
			Target: selector,
			Value:  expectedText,
		}, nil

	case "assert_element_exists":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: assert_element_exists requires a selector", lineNum)
		}
		return &fasttest.Step{
			Action: "assert_element_exists",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "assert_element_not_exists":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: assert_element_not_exists requires a selector", lineNum)
		}
		return &fasttest.Step{
			Action: "assert_element_not_exists",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "assert_text_contains":
		if len(parts) < 3 {
			return nil, fmt.Errorf("line %d: assert_text_contains requires a selector and text", lineNum)
		}
		selector := strings.Trim(parts[1], `"'`)
		text := strings.Trim(strings.Join(parts[2:], " "), `"'`)
		return &fasttest.Step{
			Action: "assert_text_contains",
			Target: selector,
			Value:  text,
		}, nil

	case "assert_url":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: assert_url requires a URL pattern", lineNum)
		}
		return &fasttest.Step{
			Action: "assert_url",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "assert_title":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: assert_title requires expected title", lineNum)
		}
		return &fasttest.Step{
			Action: "assert_title",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "assert_text_visible":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: assert_text_visible requires text to search for", lineNum)
		}
		return &fasttest.Step{
			Action: "assert_text_visible",
			Value:  strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "assert_attribute":
		if len(parts) < 4 {
			return nil, fmt.Errorf("line %d: assert_attribute requires selector, attribute name, and expected value", lineNum)
		}
		selector := strings.Trim(parts[1], `"'`)
		attribute := strings.Trim(parts[2], `"'`)
		value := strings.Trim(strings.Join(parts[3:], " "), `"'`)
		return &fasttest.Step{
			Action: "assert_attribute",
			Target: selector + "|" + attribute,
			Value:  value,
		}, nil

	case "screenshot":
		filename := ""
		if len(parts) >= 2 {
			filename = strings.Trim(strings.Join(parts[1:], " "), `"'`)
		}
		return &fasttest.Step{
			Action: "screenshot",
			Target: filename,
		}, nil

	case "snapshot":
		filename := ""
		if len(parts) >= 2 {
			filename = strings.Trim(strings.Join(parts[1:], " "), `"'`)
		}
		return &fasttest.Step{
			Action: "snapshot",
			Target: filename,
		}, nil

	case "wait_for_text":
		if len(parts) < 3 {
			return nil, fmt.Errorf("line %d: wait_for_text requires a selector and text", lineNum)
		}
		selector := strings.Trim(parts[1], `"'`)
		text := strings.Trim(strings.Join(parts[2:], " "), `"'`)
		return &fasttest.Step{
			Action: "wait_for_text",
			Target: selector,
			Value:  text,
		}, nil

	case "wait_for_url":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: wait_for_url requires a URL pattern", lineNum)
		}
		return &fasttest.Step{
			Action: "wait_for_url",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "select":
		if len(parts) < 3 {
			return nil, fmt.Errorf("line %d: select requires a selector and value", lineNum)
		}
		selector := strings.Trim(parts[1], `"'`)
		value := strings.Trim(strings.Join(parts[2:], " "), `"'`)
		return &fasttest.Step{
			Action: "select",
			Target: selector,
			Value:  value,
		}, nil

	case "check":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: check requires a selector", lineNum)
		}
		return &fasttest.Step{
			Action: "check",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "uncheck":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: uncheck requires a selector", lineNum)
		}
		return &fasttest.Step{
			Action: "uncheck",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	case "hover":
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: hover requires a selector", lineNum)
		}
		return &fasttest.Step{
			Action: "hover",
			Target: strings.Trim(strings.Join(parts[1:], " "), `"'`),
		}, nil

	default:
		return nil, fmt.Errorf("line %d: unknown action: %s", lineNum, action)
	}
}
