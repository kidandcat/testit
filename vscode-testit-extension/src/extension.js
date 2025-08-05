const vscode = require('vscode');

const TESTIT_COMMANDS = [
  // Navigation
  { label: 'navigate', kind: vscode.CompletionItemKind.Function, detail: 'Navigate to URL', insertText: 'navigate ${1:url}' },
  
  // Interaction
  { label: 'click', kind: vscode.CompletionItemKind.Function, detail: 'Click element', insertText: 'click ${1:selector}' },
  { label: 'type', kind: vscode.CompletionItemKind.Function, detail: 'Type text', insertText: 'type ${1:selector} ${2:text}' },
  { label: 'select', kind: vscode.CompletionItemKind.Function, detail: 'Select option', insertText: 'select ${1:selector} ${2:value}' },
  { label: 'check', kind: vscode.CompletionItemKind.Function, detail: 'Check checkbox', insertText: 'check ${1:selector}' },
  { label: 'uncheck', kind: vscode.CompletionItemKind.Function, detail: 'Uncheck checkbox', insertText: 'uncheck ${1:selector}' },
  { label: 'hover', kind: vscode.CompletionItemKind.Function, detail: 'Hover over element', insertText: 'hover ${1:selector}' },
  
  // Wait commands
  { label: 'wait_for', kind: vscode.CompletionItemKind.Function, detail: 'Wait for element', insertText: 'wait_for ${1:selector}' },
  { label: 'wait_for_text', kind: vscode.CompletionItemKind.Function, detail: 'Wait for text', insertText: 'wait_for_text ${1:selector} ${2:text}' },
  { label: 'wait_for_url', kind: vscode.CompletionItemKind.Function, detail: 'Wait for URL', insertText: 'wait_for_url ${1:url_pattern}' },
  
  // Assertions
  { label: 'assert_text', kind: vscode.CompletionItemKind.Function, detail: 'Assert exact text', insertText: 'assert_text ${1:selector} ${2:expected_text}' },
  { label: 'assert_text_contains', kind: vscode.CompletionItemKind.Function, detail: 'Assert contains text', insertText: 'assert_text_contains ${1:selector} ${2:partial_text}' },
  { label: 'assert_text_visible', kind: vscode.CompletionItemKind.Function, detail: 'Assert text visible', insertText: 'assert_text_visible ${1:text}' },
  { label: 'assert_element_exists', kind: vscode.CompletionItemKind.Function, detail: 'Assert element exists', insertText: 'assert_element_exists ${1:selector}' },
  { label: 'assert_element_not_exists', kind: vscode.CompletionItemKind.Function, detail: 'Assert element not exists', insertText: 'assert_element_not_exists ${1:selector}' },
  { label: 'assert_url', kind: vscode.CompletionItemKind.Function, detail: 'Assert URL matches', insertText: 'assert_url ${1:expected_url}' },
  { label: 'assert_title', kind: vscode.CompletionItemKind.Function, detail: 'Assert page title', insertText: 'assert_title ${1:expected_title}' },
  { label: 'assert_attribute', kind: vscode.CompletionItemKind.Function, detail: 'Assert attribute value', insertText: 'assert_attribute ${1:selector} ${2:attribute} ${3:expected_value}' },
  
  // Visual testing
  { label: 'screenshot', kind: vscode.CompletionItemKind.Function, detail: 'Take screenshot', insertText: 'screenshot${1: ${2:filename.png}}' },
  { label: 'snapshot', kind: vscode.CompletionItemKind.Function, detail: 'Take HTML snapshot', insertText: 'snapshot${1: ${2:filename.html}}' },
  
  // Test declaration
  { label: 'test', kind: vscode.CompletionItemKind.Keyword, detail: 'Define a test', insertText: 'test "${1:Test name}"\n  ${2:}' }
];

const CSS_SELECTORS = [
  { label: '#id', kind: vscode.CompletionItemKind.Value, detail: 'ID selector', insertText: '#${1:id}' },
  { label: '.class', kind: vscode.CompletionItemKind.Value, detail: 'Class selector', insertText: '.${1:class}' },
  { label: 'tag', kind: vscode.CompletionItemKind.Value, detail: 'Tag selector', insertText: '${1:div}' },
  { label: '[attribute]', kind: vscode.CompletionItemKind.Value, detail: 'Attribute selector', insertText: '[${1:attribute}="${2:value}"]' },
  { label: 'button[type="submit"]', kind: vscode.CompletionItemKind.Value, detail: 'Submit button', insertText: 'button[type="submit"]' },
  { label: 'input[type="text"]', kind: vscode.CompletionItemKind.Value, detail: 'Text input', insertText: 'input[type="text"]' },
  { label: 'input[type="email"]', kind: vscode.CompletionItemKind.Value, detail: 'Email input', insertText: 'input[type="email"]' },
  { label: 'input[type="password"]', kind: vscode.CompletionItemKind.Value, detail: 'Password input', insertText: 'input[type="password"]' }
];

function activate(context) {
  // Register completion provider for TestIt commands
  const commandProvider = vscode.languages.registerCompletionItemProvider(
    'testit',
    {
      provideCompletionItems(document, position) {
        const linePrefix = document.lineAt(position).text.substr(0, position.character);
        
        // Don't provide completions in comments
        if (linePrefix.includes('#')) {
          return undefined;
        }
        
        // Check if we're at the beginning of a command line (after indentation)
        const trimmedPrefix = linePrefix.trim();
        
        // If we're after a command and a space, provide selector suggestions
        const words = trimmedPrefix.split(/\s+/);
        if (words.length > 1) {
          const command = words[0];
          const selectorCommands = ['click', 'type', 'select', 'check', 'uncheck', 'hover', 
                                   'wait_for', 'wait_for_text', 'assert_text', 'assert_text_contains',
                                   'assert_element_exists', 'assert_element_not_exists', 'assert_attribute'];
          
          if (selectorCommands.includes(command) && words.length === 2) {
            return CSS_SELECTORS.map(item => {
              const completion = new vscode.CompletionItem(item.label, item.kind);
              completion.detail = item.detail;
              completion.insertText = new vscode.SnippetString(item.insertText);
              return completion;
            });
          }
        }
        
        // Provide command completions
        return TESTIT_COMMANDS.map(item => {
          const completion = new vscode.CompletionItem(item.label, item.kind);
          completion.detail = item.detail;
          completion.insertText = new vscode.SnippetString(item.insertText);
          return completion;
        });
      }
    }
  );
  
  // Register hover provider for command documentation
  const hoverProvider = vscode.languages.registerHoverProvider('testit', {
    provideHover(document, position) {
      const range = document.getWordRangeAtPosition(position);
      const word = document.getText(range);
      
      const commandDocs = {
        'navigate': 'Navigate to the specified URL\n\nUsage: `navigate url`\n\nExample: `navigate https://example.com`',
        'click': 'Click on an element matching the CSS selector\n\nUsage: `click selector`\n\nExample: `click button[type="submit"]`',
        'type': 'Type text into an input field\n\nUsage: `type selector text`\n\nExample: `type #username admin@example.com`',
        'select': 'Select an option from a dropdown\n\nUsage: `select selector value`\n\nExample: `select #country US`',
        'check': 'Check a checkbox\n\nUsage: `check selector`\n\nExample: `check #agree-terms`',
        'uncheck': 'Uncheck a checkbox\n\nUsage: `uncheck selector`\n\nExample: `uncheck #newsletter`',
        'hover': 'Hover over an element\n\nUsage: `hover selector`\n\nExample: `hover .dropdown-menu`',
        'wait_for': 'Wait for an element to be visible\n\nUsage: `wait_for selector`\n\nExample: `wait_for #loading-complete`',
        'wait_for_text': 'Wait for specific text in an element\n\nUsage: `wait_for_text selector text`\n\nExample: `wait_for_text .status Success`',
        'wait_for_url': 'Wait for URL to match pattern\n\nUsage: `wait_for_url url_pattern`\n\nExample: `wait_for_url https://example.com/dashboard`',
        'assert_text': 'Assert exact text match in element\n\nUsage: `assert_text selector expected_text`\n\nExample: `assert_text h1 Welcome Home`',
        'assert_text_contains': 'Assert element contains text\n\nUsage: `assert_text_contains selector partial_text`\n\nExample: `assert_text_contains .message successfully`',
        'assert_text_visible': 'Assert text is visible on page\n\nUsage: `assert_text_visible text`\n\nExample: `assert_text_visible Login successful`',
        'assert_element_exists': 'Assert element exists in DOM\n\nUsage: `assert_element_exists selector`\n\nExample: `assert_element_exists #user-profile`',
        'assert_element_not_exists': 'Assert element does not exist\n\nUsage: `assert_element_not_exists selector`\n\nExample: `assert_element_not_exists .error-message`',
        'assert_url': 'Assert current URL matches\n\nUsage: `assert_url expected_url`\n\nExample: `assert_url https://example.com/success`',
        'assert_title': 'Assert page title matches\n\nUsage: `assert_title expected_title`\n\nExample: `assert_title My Application - Home`',
        'assert_attribute': 'Assert element attribute value\n\nUsage: `assert_attribute selector attribute expected_value`\n\nExample: `assert_attribute input#email type email`',
        'screenshot': 'Take a screenshot\n\nUsage: `screenshot [filename]`\n\nExamples:\n- `screenshot` (auto-generated filename)\n- `screenshot login_page.png`',
        'snapshot': 'Take an HTML snapshot\n\nUsage: `snapshot [filename]`\n\nExamples:\n- `snapshot` (auto-generated filename)\n- `snapshot page_state.html`',
        'test': 'Define a test\n\nUsage: `test "Test name"`\n\nExample:\n```\ntest "User can login"\n  navigate https://example.com/login\n  type #username admin\n  click button[type="submit"]\n```'
      };
      
      if (commandDocs[word]) {
        return new vscode.Hover(new vscode.MarkdownString(commandDocs[word]));
      }
    }
  });
  
  context.subscriptions.push(commandProvider, hoverProvider);
}

function deactivate() {}

module.exports = {
  activate,
  deactivate
};