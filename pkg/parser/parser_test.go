package parser

import (
	"testing"

	"github.com/kidandcat/fasttest/pkg/fasttest"
)

func TestParseString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []fasttest.Test
		wantErr bool
	}{
		{
			name: "simple test",
			input: `test "Login test"
  navigate "https://example.com"
  click "#button"
  assert_text ".result" "Success"`,
			want: []fasttest.Test{
				{
					Name: "Login test",
					Steps: []fasttest.Step{
						{Action: "navigate", Target: "https://example.com"},
						{Action: "click", Target: "#button"},
						{Action: "assert_text", Target: ".result", Value: "Success"},
					},
				},
			},
		},
		{
			name: "multiple tests",
			input: `test "First test"
  navigate "https://example.com"

test "Second test"
  click "#button"`,
			want: []fasttest.Test{
				{
					Name: "First test",
					Steps: []fasttest.Step{
						{Action: "navigate", Target: "https://example.com"},
					},
				},
				{
					Name: "Second test",
					Steps: []fasttest.Step{
						{Action: "click", Target: "#button"},
					},
				},
			},
		},
		{
			name: "test with comments and empty lines",
			input: `# This is a comment
test "Test with comments"
  # Navigate to page
  navigate "https://example.com"
  
  # Click button
  click "#button"`,
			want: []fasttest.Test{
				{
					Name: "Test with comments",
					Steps: []fasttest.Step{
						{Action: "navigate", Target: "https://example.com"},
						{Action: "click", Target: "#button"},
					},
				},
			},
		},
		{
			name: "screenshot commands",
			input: `test "Screenshot test"
  navigate "https://example.com"
  screenshot
  screenshot "custom.png"`,
			want: []fasttest.Test{
				{
					Name: "Screenshot test",
					Steps: []fasttest.Step{
						{Action: "navigate", Target: "https://example.com"},
						{Action: "screenshot", Target: ""},
						{Action: "screenshot", Target: "custom.png"},
					},
				},
			},
		},
		{
			name: "type command with spaces",
			input: `test "Type test"
  type "#input" "Hello World"`,
			want: []fasttest.Test{
				{
					Name: "Type test",
					Steps: []fasttest.Step{
						{Action: "type", Target: "#input", Value: "Hello World"},
					},
				},
			},
		},
		{
			name: "assert_attribute command",
			input: `test "Attribute test"
  assert_attribute "#link" "href" "https://example.com"`,
			want: []fasttest.Test{
				{
					Name: "Attribute test",
					Steps: []fasttest.Step{
						{Action: "assert_attribute", Target: "#link|href", Value: "https://example.com"},
					},
				},
			},
		},
		{
			name: "wait commands",
			input: `test "Wait test"
  wait_for ".element"
  wait_for_text ".message" "Loading complete"
  wait_for_url "/dashboard"`,
			want: []fasttest.Test{
				{
					Name: "Wait test",
					Steps: []fasttest.Step{
						{Action: "wait_for", Target: ".element"},
						{Action: "wait_for_text", Target: ".message", Value: "Loading complete"},
						{Action: "wait_for_url", Target: "/dashboard"},
					},
				},
			},
		},
		{
			name: "form interaction commands",
			input: `test "Form test"
  select "#country" "USA"
  check "#terms"
  uncheck "#newsletter"
  hover "#submit"`,
			want: []fasttest.Test{
				{
					Name: "Form test",
					Steps: []fasttest.Step{
						{Action: "select", Target: "#country", Value: "USA"},
						{Action: "check", Target: "#terms"},
						{Action: "uncheck", Target: "#newsletter"},
						{Action: "hover", Target: "#submit"},
					},
				},
			},
		},
		{
			name: "quoted selectors",
			input: `test "Quoted test"
  click ".btn[data-action='save']"
  type 'input[name="email"]' "test@example.com"`,
			want: []fasttest.Test{
				{
					Name: "Quoted test",
					Steps: []fasttest.Step{
						{Action: "click", Target: ".btn[data-action='save']"},
						{Action: "type", Target: "input[name=\"email\"]", Value: "test@example.com"},
					},
				},
			},
		},
		{
			name: "invalid command",
			input: `test "Invalid"
  invalid_command "arg"`,
			wantErr: true,
		},
		{
			name: "missing required argument",
			input: `test "Invalid"
  click`,
			wantErr: true,
		},
	}

	parser := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseString() got %d tests, want %d", len(got), len(tt.want))
					return
				}
				for i := range got {
					if got[i].Name != tt.want[i].Name {
						t.Errorf("Test[%d].Name = %v, want %v", i, got[i].Name, tt.want[i].Name)
					}
					if len(got[i].Steps) != len(tt.want[i].Steps) {
						t.Errorf("Test[%d] got %d steps, want %d", i, len(got[i].Steps), len(tt.want[i].Steps))
						continue
					}
					for j := range got[i].Steps {
						if got[i].Steps[j] != tt.want[i].Steps[j] {
							t.Errorf("Test[%d].Steps[%d] = %v, want %v", i, j, got[i].Steps[j], tt.want[i].Steps[j])
						}
					}
				}
			}
		})
	}
}
