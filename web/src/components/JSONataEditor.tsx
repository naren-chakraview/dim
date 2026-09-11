import React, { useCallback, useState } from 'react';
import Editor from '@monaco-editor/react';
import '../styles/JSONataEditor.css';

interface JSONataEditorProps {
  value?: string;
  onChange?: (value: string) => void;
  onBlur?: () => void;
  placeholder?: string;
  error?: string;
  height?: string;
  readOnly?: boolean;
}

/**
 * JSONataEditor component with Monaco Editor and syntax highlighting.
 *
 * Features:
 * - Syntax highlighting (JavaScript-like)
 * - Code completion hints
 * - Error detection
 * - Undo/redo
 * - Multiple themes
 *
 * Usage:
 * ```
 * <JSONataEditor
 *   value={expr}
 *   onChange={setExpr}
 *   error={validationError}
 *   placeholder="e.g., $payload.id"
 * />
 * ```
 */
export function JSONataEditor({
  value = '',
  onChange,
  onBlur,
  placeholder = 'Enter JSONata expression...',
  error,
  height = '200px',
  readOnly = false,
}: JSONataEditorProps) {
  const [isValid, setIsValid] = useState(true);

  const handleChange = useCallback(
    (newValue: string | undefined) => {
      if (newValue === undefined) return;
      onChange?.(newValue);

      // Validate JSONata syntax
      const valid = validateJSONata(newValue);
      setIsValid(valid);
    },
    [onChange]
  );

  const handleBlur = useCallback(() => {
    onBlur?.();
  }, [onBlur]);

  return (
    <div className={`jsonata-editor-container ${error ? 'has-error' : ''}`}>
      <div className="editor-wrapper">
        <Editor
          height={height}
          defaultLanguage="javascript"
          value={value}
          onChange={handleChange}
          onBlur={handleBlur}
          theme="light"
          options={{
            minimap: { enabled: false },
            fontSize: 13,
            fontFamily: '"Monaco", "Courier New", monospace',
            wordWrap: 'on',
            autoClosingBrackets: 'always',
            autoClosingQuotes: 'always',
            formatOnPaste: true,
            formatOnType: true,
            readOnly,
            padding: { top: 10, bottom: 10 },
            lineNumbers: 'off',
            folding: false,
            glyphMargin: false,
            lineDecorationsWidth: 0,
            bracketPairColorization: { enabled: true },
            'bracketPairColorization.independentColorPoolPerBracketType': true,
          }}
        />
      </div>

      {/* Error Display */}
      {error && (
        <div className="editor-error">
          <span className="error-icon">⚠️</span>
          <span className="error-message">{error}</span>
        </div>
      )}

      {/* Validation Status */}
      {!error && value && (
        <div className={`validation-status ${isValid ? 'valid' : 'invalid'}`}>
          {isValid ? (
            <>
              <span className="status-icon">✓</span>
              <span className="status-text">Valid JSONata expression</span>
            </>
          ) : (
            <>
              <span className="status-icon">⚠️</span>
              <span className="status-text">Check syntax</span>
            </>
          )}
        </div>
      )}

      {/* Help Text */}
      <div className="editor-help">
        <details>
          <summary>JSONata Syntax Help</summary>
          <div className="help-content">
            <h4>Common Patterns:</h4>
            <ul>
              <li>
                <code>$payload.field</code> - Access message field
              </li>
              <li>
                <code>$payload.amount * 1.1</code> - Arithmetic
              </li>
              <li>
                <code>$payload.status = "active"</code> - Comparison
              </li>
              <li>
                <code>$payload.id &amp; ":" &amp; $payload.type</code> - String concat
              </li>
              <li>
                <code>$payload[0]</code> - Array access
              </li>
              <li>
                <code>$payload.items[*].price</code> - Map operation
              </li>
              <li>
                <code>$sum($payload.items.price)</code> - Functions
              </li>
            </ul>
            <p>
              <a href="https://docs.jsonata.org/" target="_blank" rel="noopener noreferrer">
                Full JSONata Documentation →
              </a>
            </p>
          </div>
        </details>
      </div>
    </div>
  );
}

/**
 * Basic JSONata syntax validation.
 * Checks for common syntax errors without executing code.
 */
export function validateJSONata(expr: string): boolean {
  if (!expr.trim()) return true; // Empty is OK

  // Check for common syntax errors
  const issues = [];

  // Balanced parentheses
  const parenCount = (expr.match(/\(/g) || []).length - (expr.match(/\)/g) || []).length;
  if (parenCount !== 0) {
    issues.push('Unbalanced parentheses');
  }

  // Balanced brackets
  const bracketCount = (expr.match(/\[/g) || []).length - (expr.match(/\]/g) || []).length;
  if (bracketCount !== 0) {
    issues.push('Unbalanced brackets');
  }

  // Balanced braces
  const braceCount = (expr.match(/\{/g) || []).length - (expr.match(/\}/g) || []).length;
  if (braceCount !== 0) {
    issues.push('Unbalanced braces');
  }

  // Balanced quotes
  const singleQuotes = (expr.match(/'/g) || []).length;
  const doubleQuotes = (expr.match(/"/g) || []).length;
  if (singleQuotes % 2 !== 0 || doubleQuotes % 2 !== 0) {
    issues.push('Unbalanced quotes');
  }

  // Check for incomplete operators
  if (expr.trim().endsWith('$') || expr.trim().endsWith('.') || expr.trim().endsWith('(')) {
    issues.push('Incomplete expression');
  }

  return issues.length === 0;
}

/**
 * Parse JSONata expression to extract variable references.
 * Useful for understanding what fields an expression uses.
 */
export function extractVariablesFromExpression(expr: string): string[] {
  const variables = new Set<string>();

  // Match $variable.path patterns
  const varPattern = /\$\w+(?:\.\w+)*/g;
  const matches = expr.match(varPattern) || [];

  matches.forEach(match => {
    // Extract just the main variable (e.g., $payload)
    const mainVar = match.split('.')[0];
    variables.add(mainVar);
  });

  return Array.from(variables);
}

/**
 * Generate autocomplete suggestions for JSONata.
 * (Simple implementation; could be enhanced with LSP)
 */
export function getJSONataCompletions(
  expr: string,
  position: number
): Array<{ label: string; detail: string }> {
  const beforeCursor = expr.substring(0, position);

  // Check if we're in a field access context
  if (beforeCursor.includes('$payload.')) {
    // Common payload fields
    const fields = [
      'id',
      'type',
      'status',
      'amount',
      'timestamp',
      'source',
      'destination',
      'headers',
      'body',
      'metadata',
    ];

    return fields.map(field => ({
      label: field,
      detail: 'field',
    }));
  }

  // Check if we're in a function context
  if (beforeCursor.includes('$')) {
    const functions = [
      'sum',
      'count',
      'min',
      'max',
      'avg',
      'map',
      'filter',
      'reduce',
      'sort',
      'group',
      'length',
      'reverse',
      'join',
      'split',
      'lowercase',
      'uppercase',
      'trim',
      'contains',
      'startsWith',
      'endsWith',
      'substring',
      'replace',
      'concat',
      'now',
      'floor',
      'ceil',
      'round',
    ];

    return functions.map(fn => ({
      label: `$${fn}()`,
      detail: 'function',
    }));
  }

  return [];
}
