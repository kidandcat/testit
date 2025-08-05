# TestIt VS Code Extension

Syntax highlighting and autocomplete support for the TestIt browser automation framework.

## Features

- **Syntax Highlighting**: Full syntax highlighting for `.test` and `.testit` files
- **Autocomplete**: IntelliSense support for all TestIt commands
- **Snippets**: Code snippets for common test patterns
- **Hover Documentation**: Hover over commands to see documentation
- **CSS Selector Suggestions**: Get autocomplete for CSS selectors within commands

## Installation

1. Open VS Code
2. Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
3. Type "Install from VSIX"
4. Select the `.vsix` file

Or install from source:
```bash
cd vscode-testit-extension
npm install
npm run package
```

## Usage

1. Create a file with `.test` or `.testit` extension
2. Start typing TestIt commands and use autocomplete
3. Use snippets like `test-login` or `test-form` for templates

## Commands

All TestIt commands are supported with autocomplete:

- Navigation: `navigate`
- Interaction: `click`, `type`, `select`, `check`, `uncheck`, `hover`
- Wait: `wait_for`, `wait_for_text`, `wait_for_url`
- Assertions: `assert_text`, `assert_text_contains`, `assert_text_visible`, etc.
- Visual: `screenshot`, `snapshot`

## Example

```testit
# Login test
test "User can login"
  navigate https://example.com/login
  type #username user@example.com
  type #password password123
  click button[type='submit']
  wait_for_url https://example.com/dashboard
  assert_text_visible Welcome
  screenshot login_success.png
```

## Development

To contribute or modify:

1. Clone the repository
2. Open in VS Code
3. Press `F5` to launch a new VS Code window with the extension loaded
4. Make changes and reload the window to test