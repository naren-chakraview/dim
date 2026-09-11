import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { SchemaForm } from '../SchemaForm';
import { JSONSchema } from '../../hooks/useSchemaForm';

describe('SchemaForm Component', () => {
  const mockSchema: JSONSchema = {
    type: 'object',
    title: 'Test Form',
    properties: {
      name: {
        type: 'string',
        title: 'Name',
        description: 'Full name',
      },
      amount: {
        type: 'number',
        title: 'Amount',
        minimum: 0,
        maximum: 1000,
      },
      status: {
        type: 'string',
        title: 'Status',
        enum: ['active', 'inactive', 'pending'],
      },
      enabled: {
        type: 'boolean',
        title: 'Enabled',
      },
      notes: {
        type: 'string',
        title: 'Notes',
        maxLength: 500,
      },
    },
    required: ['name', 'status'],
  };

  it('renders form title', () => {
    render(<SchemaForm schema={mockSchema} title="Configure Step" />);
    expect(screen.getByText('Configure Step')).toBeTruthy();
  });

  it('renders form fields from schema', () => {
    render(<SchemaForm schema={mockSchema} />);
    expect(screen.getByText('Name')).toBeTruthy();
    expect(screen.getByText('Amount')).toBeTruthy();
    expect(screen.getByText('Status')).toBeTruthy();
    expect(screen.getByText('Enabled')).toBeTruthy();
  });

  it('marks required fields', () => {
    render(<SchemaForm schema={mockSchema} />);
    const nameLabel = screen.getByText('Name').closest('.form-field');
    const amountLabel = screen.getByText('Amount').closest('.form-field');

    expect(nameLabel?.textContent).toContain('Name*');
    expect(amountLabel?.textContent).not.toContain('*');
  });

  it('renders correct input types', () => {
    const { container } = render(<SchemaForm schema={mockSchema} />);

    // Text input for name
    const nameInput = container.querySelector('input[type="text"]');
    expect(nameInput).toBeTruthy();

    // Number input for amount
    const amountInput = container.querySelector('input[type="number"]');
    expect(amountInput).toBeTruthy();

    // Select for status
    const statusSelect = container.querySelector('select');
    expect(statusSelect).toBeTruthy();

    // Checkbox for enabled
    const enabledCheckbox = container.querySelector('input[type="checkbox"]');
    expect(enabledCheckbox).toBeTruthy();

    // Textarea for notes
    const notesTextarea = container.querySelector('textarea');
    expect(notesTextarea).toBeTruthy();
  });

  it('calls onChange when value changes', () => {
    const onChange = vi.fn();
    const { container } = render(
      <SchemaForm schema={mockSchema} value={{}} onChange={onChange} />
    );

    const nameInput = container.querySelector('input[type="text"]') as HTMLInputElement;
    fireEvent.change(nameInput, { target: { value: 'John Doe' } });

    expect(onChange).toHaveBeenCalledWith({ name: 'John Doe' });
  });

  it('handles select options from enum', () => {
    const { container } = render(<SchemaForm schema={mockSchema} />);
    const select = container.querySelector('select') as HTMLSelectElement;

    expect(select.options[0].textContent).toContain('Select');
    expect(select.options[1].value).toBe('active');
    expect(select.options[2].value).toBe('inactive');
    expect(select.options[3].value).toBe('pending');
  });

  it('renders step type forms', () => {
    render(<SchemaForm stepType="translate" />);
    expect(screen.getByText(/Expression/i)).toBeTruthy();
  });

  it('shows empty state for no schema', () => {
    render(<SchemaForm />);
    expect(screen.getByText('No configuration available')).toBeTruthy();
  });

  it('populates initial values', () => {
    const { container } = render(
      <SchemaForm
        schema={mockSchema}
        value={{
          name: 'Jane Doe',
          amount: 500,
          status: 'active',
          enabled: true,
        }}
      />
    );

    const nameInput = container.querySelector('input[type="text"]') as HTMLInputElement;
    const amountInput = container.querySelector('input[type="number"]') as HTMLInputElement;
    const statusSelect = container.querySelector('select') as HTMLSelectElement;
    const enabledCheckbox = container.querySelector('input[type="checkbox"]') as HTMLInputElement;

    expect(nameInput.value).toBe('Jane Doe');
    expect(amountInput.value).toBe('500');
    expect(statusSelect.value).toBe('active');
    expect(enabledCheckbox.checked).toBe(true);
  });

  it('validates on blur', () => {
    const { container } = render(
      <SchemaForm
        schema={mockSchema}
        value={{}}
        onChange={() => {}}
      />
    );

    const nameInput = container.querySelector('input[type="text"]') as HTMLInputElement;
    fireEvent.blur(nameInput);

    // Should show validation error since name is required
    expect(screen.queryByText(/Validation Errors/i)).toBeTruthy();
  });
});
