import React, { useState, useCallback } from 'react';
import { RouteConfig } from '../types/route';
import '../styles/ValidationPanel.css';

export interface ValidationError {
  code: string;
  message: string;
  path?: string;
  severity: 'error' | 'warning';
}

export interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
  warnings: ValidationError[];
  route_version?: string;
  timestamp?: string;
}

interface ValidationPanelProps {
  route?: RouteConfig;
  isValidating?: boolean;
  result?: ValidationResult;
  onValidate?: (route: RouteConfig) => Promise<ValidationResult>;
  autoValidate?: boolean;
}

/**
 * ValidationPanel component displays route validation results.
 *
 * Features:
 * - Show validation errors and warnings
 * - "Ready to save" indicator when valid
 * - Error categorization
 * - Clickable errors (highlight fields)
 * - Auto-validate on changes (optional)
 * - Route version hash display
 *
 * Usage:
 * ```
 * <ValidationPanel
 *   route={route}
 *   result={validationResult}
 *   onValidate={validateRoute}
 *   autoValidate={true}
 * />
 * ```
 */
export function ValidationPanel({
  route,
  isValidating = false,
  result,
  onValidate,
  autoValidate = false,
}: ValidationPanelProps) {
  const [lastValidatedRoute, setLastValidatedRoute] = useState<string>('');

  // Auto-validate when route changes
  React.useEffect(() => {
    if (!autoValidate || !route || !onValidate) return;

    const routeStr = JSON.stringify(route);
    if (routeStr === lastValidatedRoute) return;

    const validateRoute = async () => {
      try {
        await onValidate(route);
        setLastValidatedRoute(routeStr);
      } catch (err) {
        console.error('Validation error:', err);
      }
    };

    // Debounce validation to avoid too-frequent calls
    const timer = setTimeout(validateRoute, 500);
    return () => clearTimeout(timer);
  }, [route, autoValidate, onValidate, lastValidatedRoute]);

  if (!result) {
    return (
      <div className="validation-panel empty">
        <div className="validation-placeholder">
          <p>No validation results yet</p>
          <small>Edit a route and save to validate</small>
        </div>
      </div>
    );
  }

  const errorCount = result.errors.length;
  const warningCount = result.warnings.length;
  const allIssues = [...result.errors, ...result.warnings];

  return (
    <div className={`validation-panel ${result.valid ? 'valid' : 'invalid'}`}>
      {/* Header */}
      <div className="validation-header">
        <div className="status-indicator">
          {isValidating ? (
            <>
              <span className="spinner">⟳</span>
              <span className="status-text">Validating...</span>
            </>
          ) : result.valid ? (
            <>
              <span className="status-icon">✓</span>
              <span className="status-text">Valid and ready to save</span>
            </>
          ) : (
            <>
              <span className="status-icon">⚠️</span>
              <span className="status-text">
                {errorCount} error{errorCount !== 1 ? 's' : ''}, {warningCount} warning
                {warningCount !== 1 ? 's' : ''}
              </span>
            </>
          )}
        </div>

        {result.route_version && (
          <div className="route-version">
            <code>{result.route_version.substring(0, 12)}...</code>
          </div>
        )}
      </div>

      {/* Issues List */}
      {allIssues.length > 0 && (
        <div className="issues-container">
          {result.errors.length > 0 && (
            <div className="issue-group errors">
              <h4 className="issue-header">
                <span className="icon">✕</span>
                Errors ({errorCount})
              </h4>
              <ul className="issue-list">
                {result.errors.map((error, idx) => (
                  <IssueItem key={`error-${idx}`} issue={error} />
                ))}
              </ul>
            </div>
          )}

          {result.warnings.length > 0 && (
            <div className="issue-group warnings">
              <h4 className="issue-header">
                <span className="icon">!</span>
                Warnings ({warningCount})
              </h4>
              <ul className="issue-list">
                {result.warnings.map((warning, idx) => (
                  <IssueItem key={`warning-${idx}`} issue={warning} />
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* Success Message */}
      {result.valid && allIssues.length === 0 && (
        <div className="validation-success">
          <div className="success-icon">✓</div>
          <div className="success-text">
            <p>Route configuration is valid</p>
            <small>All checks passed • Safe to save</small>
          </div>
        </div>
      )}

      {/* Timestamp */}
      {result.timestamp && (
        <div className="validation-timestamp">
          <small>Validated at {formatTime(result.timestamp)}</small>
        </div>
      )}
    </div>
  );
}

/**
 * Individual validation issue display.
 */
function IssueItem({ issue }: { issue: ValidationError }) {
  const [expanded, setExpanded] = React.useState(false);

  return (
    <li
      className={`issue-item ${issue.severity}`}
      onClick={() => setExpanded(!expanded)}
    >
      <div className="issue-header-row">
        <span className="issue-code">{issue.code}</span>
        {issue.path && <span className="issue-path">{issue.path}</span>}
        <span className="expand-icon">{expanded ? '▼' : '▶'}</span>
      </div>
      <div className="issue-message">{issue.message}</div>
      {expanded && (
        <div className="issue-details">
          <details open>
            <summary>How to fix</summary>
            <p>{getFixSuggestion(issue.code)}</p>
          </details>
        </div>
      )}
    </li>
  );
}

/**
 * Get helpful fix suggestion based on error code.
 */
function getFixSuggestion(code: string): string {
  const suggestions: Record<string, string> = {
    REQUIRED_FIELD_MISSING: 'Add the missing required field to your route configuration.',
    INVALID_SCHEMA: 'Check that your YAML syntax is correct and matches the route schema.',
    VALIDATION_FAILED: 'Review the error details and adjust your configuration accordingly.',
    AUTH_VALIDATION_ERROR:
      'Ensure all auth declarations are valid and properly configured for your route.',
    CROSS_DOMAIN_SECRET:
      'Use explicit cross-domain syntax for shared secrets: ${SECRET:domain/name}',
    UNRESOLVED_SECRET: 'Check that the secret is registered in the correct domain.',
    DUPLICATE_STEP:
      'Each step name must be unique within a route. Rename duplicate steps.',
    INVALID_ADAPTER_TYPE: 'Use a supported adapter type: http, kafka, file, s3, amqp, sftp.',
    MISSING_IMPORT: 'Add the missing governance fragment import to your route.',
    MANDATORY_FRAGMENT_MISSING: 'Ensure all mandatory governance fragments are imported.',
  };

  return suggestions[code] || 'Review the error message and check the documentation.';
}

/**
 * Format ISO timestamp for display.
 */
function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    const seconds = date.getSeconds().toString().padStart(2, '0');
    return `${hours}:${minutes}:${seconds}`;
  } catch {
    return timestamp;
  }
}
