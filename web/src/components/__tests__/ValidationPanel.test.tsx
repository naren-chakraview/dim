import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { render, screen } from '@testing-library/react';
import { ValidationPanel, ValidationResult, ValidationError } from '../ValidationPanel';

describe('ValidationPanel Component', () => {
  const mockValidResult: ValidationResult = {
    valid: true,
    errors: [],
    warnings: [],
    route_version: 'sha256:abc123def456',
    timestamp: new Date().toISOString(),
  };

  const mockInvalidResult: ValidationResult = {
    valid: false,
    errors: [
      {
        code: 'REQUIRED_FIELD_MISSING',
        message: 'Field "from" is required',
        path: 'routes.order-payment',
        severity: 'error',
      },
      {
        code: 'INVALID_SCHEMA',
        message: 'Schema validation failed',
        path: 'routes.order-payment.steps[0]',
        severity: 'error',
      },
    ],
    warnings: [
      {
        code: 'MISSING_IMPORT',
        message: 'Governance fragment not imported',
        severity: 'warning',
      },
    ],
    route_version: 'sha256:xyz789uvw012',
    timestamp: new Date().toISOString(),
  };

  it('renders empty state when no result', () => {
    render(<ValidationPanel />);
    expect(screen.getByText('No validation results yet')).toBeTruthy();
  });

  it('shows valid status for valid routes', () => {
    render(<ValidationPanel result={mockValidResult} />);
    expect(screen.getByText('Valid and ready to save')).toBeTruthy();
    expect(screen.getByText(/Route configuration is valid/)).toBeTruthy();
  });

  it('shows error count for invalid routes', () => {
    render(<ValidationPanel result={mockInvalidResult} />);
    expect(screen.getByText(/2 errors, 1 warning/)).toBeTruthy();
  });

  it('displays error group with error count', () => {
    render(<ValidationPanel result={mockInvalidResult} />);
    expect(screen.getByText(/Errors \(2\)/)).toBeTruthy();
  });

  it('displays warning group with warning count', () => {
    render(<ValidationPanel result={mockInvalidResult} />);
    expect(screen.getByText(/Warnings \(1\)/)).toBeTruthy();
  });

  it('shows individual error details', () => {
    render(<ValidationPanel result={mockInvalidResult} />);
    expect(screen.getByText('Field "from" is required')).toBeTruthy();
    expect(screen.getByText('Schema validation failed')).toBeTruthy();
  });

  it('displays error paths', () => {
    render(<ValidationPanel result={mockInvalidResult} />);
    expect(screen.getByText('routes.order-payment')).toBeTruthy();
    expect(screen.getByText('routes.order-payment.steps[0]')).toBeTruthy();
  });

  it('shows loading state while validating', () => {
    render(<ValidationPanel result={mockValidResult} isValidating={true} />);
    expect(screen.getByText('Validating...')).toBeTruthy();
  });

  it('displays route version hash', () => {
    render(<ValidationPanel result={mockValidResult} />);
    expect(screen.getByText(/abc123def456/)).toBeTruthy();
  });

  it('shows timestamp', () => {
    render(<ValidationPanel result={mockValidResult} />);
    expect(screen.getByText(/Validated at/)).toBeTruthy();
  });

  it('handles single error correctly', () => {
    const singleError: ValidationResult = {
      valid: false,
      errors: [
        {
          code: 'REQUIRED_FIELD_MISSING',
          message: 'Field "domain" is required',
          severity: 'error',
        },
      ],
      warnings: [],
    };

    render(<ValidationPanel result={singleError} />);
    expect(screen.getByText(/1 error, 0 warnings/)).toBeTruthy();
  });

  it('handles single warning correctly', () => {
    const singleWarning: ValidationResult = {
      valid: true,
      errors: [],
      warnings: [
        {
          code: 'MISSING_IMPORT',
          message: 'Optional fragment not imported',
          severity: 'warning',
        },
      ],
    };

    render(<ValidationPanel result={singleWarning} />);
    expect(screen.getByText(/Warnings \(1\)/)).toBeTruthy();
  });

  it('shows success message when valid with no warnings', () => {
    render(<ValidationPanel result={mockValidResult} />);
    expect(screen.getByText(/All checks passed/)).toBeTruthy();
  });

  it('applies correct styling for valid panel', () => {
    const { container } = render(<ValidationPanel result={mockValidResult} />);
    const panel = container.querySelector('.validation-panel');
    expect(panel?.classList.contains('valid')).toBe(true);
  });

  it('applies correct styling for invalid panel', () => {
    const { container } = render(<ValidationPanel result={mockInvalidResult} />);
    const panel = container.querySelector('.validation-panel');
    expect(panel?.classList.contains('invalid')).toBe(true);
  });

  it('calls onValidate when route changes and autoValidate is true', async () => {
    const onValidate = vi.fn().mockResolvedValue(mockValidResult);
    const route = {
      version: 1,
      sources: { input: { type: 'http' } },
      routes: { test: { from: 'input', steps: [] } },
      sinks: { output: { type: 'file', path: './out.jsonl' } },
    };

    render(<ValidationPanel route={route} onValidate={onValidate} autoValidate={true} />);

    // Wait for debounce
    await new Promise(resolve => setTimeout(resolve, 600));

    expect(onValidate).toHaveBeenCalledWith(route);
  });

  it('debounces validation calls', async () => {
    const onValidate = vi.fn().mockResolvedValue(mockValidResult);
    const route = {
      version: 1,
      sources: { input: { type: 'http' } },
      routes: { test: { from: 'input', steps: [] } },
      sinks: { output: { type: 'file', path: './out.jsonl' } },
    };

    const { rerender } = render(
      <ValidationPanel route={route} onValidate={onValidate} autoValidate={true} />
    );

    // Change route multiple times quickly
    rerender(
      <ValidationPanel
        route={{ ...route, domain: 'payments' }}
        onValidate={onValidate}
        autoValidate={true}
      />
    );

    // Only one validation call should be made
    await new Promise(resolve => setTimeout(resolve, 600));
    expect(onValidate).toHaveBeenCalledTimes(1);
  });
});
