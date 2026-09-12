import { describe, it, expect } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useSchemaForm, JSONSchema, getStepSchema } from '../useSchemaForm';

describe('useSchemaForm Hook', () => {
  const mockSchema: JSONSchema = {
    type: 'object',
    properties: {
      name: {
        type: 'string',
        title: 'Name',
        description: 'Full name',
        minLength: 2,
        maxLength: 100,
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
        enum: ['active', 'inactive'],
      },
      enabled: {
        type: 'boolean',
        title: 'Enabled',
      },
      description: {
        type: 'string',
        title: 'Description',
        maxLength: 500,
      },
    },
    required: ['name', 'status'],
  };

  it('generates form fields from schema', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    expect(result.current.fields).toHaveLength(5);
  });

  it('marks required fields', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    const nameField = result.current.fields.find(f => f.name === 'name');
    const amountField = result.current.fields.find(f => f.name === 'amount');

    expect(nameField?.required).toBe(true);
    expect(amountField?.required).toBe(false);
  });

  it('infers correct field types', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));

    expect(result.current.fields[0].type).toBe('text'); // name
    expect(result.current.fields[1].type).toBe('number'); // amount
    expect(result.current.fields[2].type).toBe('select'); // status (enum)
    expect(result.current.fields[3].type).toBe('boolean'); // enabled
    expect(result.current.fields[4].type).toBe('textarea'); // description (long string)
  });

  it('converts enum to select options', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    const statusField = result.current.fields.find(f => f.name === 'status');

    expect(statusField?.options).toEqual([
      { label: 'active', value: 'active' },
      { label: 'inactive', value: 'inactive' },
    ]);
  });

  it('includes field constraints (min/max)', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    const amountField = result.current.fields.find(f => f.name === 'amount');

    expect(amountField?.min).toBe(0);
    expect(amountField?.max).toBe(1000);
  });

  it('validates required fields', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    const validation = result.current.validate({});

    expect(validation.valid).toBe(false);
    expect(validation.errors).toContain('Name is required');
    expect(validation.errors).toContain('Status is required');
  });

  it('validates string length', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    const validation = result.current.validate({
      name: 'a', // Too short (min 2)
      status: 'active',
    });

    expect(validation.valid).toBe(false);
    expect(validation.errors.some(e => e.includes('Name'))).toBe(true);
  });

  it('validates number range', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    const validation = result.current.validate({
      name: 'John',
      status: 'active',
      amount: 2000, // Too high (max 1000)
    });

    expect(validation.valid).toBe(false);
    expect(validation.errors.some(e => e.includes('Amount'))).toBe(true);
  });

  it('validates enum values', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    const validation = result.current.validate({
      name: 'John',
      status: 'invalid', // Not in enum
    });

    expect(validation.valid).toBe(false);
    expect(validation.errors.some(e => e.includes('Status'))).toBe(true);
  });

  it('passes validation with valid data', () => {
    const { result } = renderHook(() => useSchemaForm(mockSchema));
    const validation = result.current.validate({
      name: 'John Doe',
      status: 'active',
      amount: 500,
      enabled: true,
      description: 'Test',
    });

    expect(validation.valid).toBe(true);
    expect(validation.errors).toHaveLength(0);
  });

  it('returns empty fields for undefined schema', () => {
    const { result } = renderHook(() => useSchemaForm(undefined));
    expect(result.current.fields).toHaveLength(0);
  });

  it('handles missing properties', () => {
    const minimalSchema: JSONSchema = {
      type: 'object',
    };
    const { result } = renderHook(() => useSchemaForm(minimalSchema));
    expect(result.current.fields).toHaveLength(0);
  });
});

describe('getStepSchema', () => {
  it('returns translate step schema', () => {
    const schema = getStepSchema('translate');
    expect(schema.properties?.expr).toBeTruthy();
    expect(schema.required).toContain('expr');
  });

  it('returns log step schema', () => {
    const schema = getStepSchema('log');
    expect(schema.properties?.level).toBeTruthy();
    expect(schema.properties?.level.enum).toContain('debug');
  });

  it('returns filter step schema', () => {
    const schema = getStepSchema('filter');
    expect(schema.properties?.condition).toBeTruthy();
  });

  it('returns delay step schema', () => {
    const schema = getStepSchema('delay');
    expect(schema.properties?.duration).toBeTruthy();
  });

  it('returns default schema for unknown step type', () => {
    const schema = getStepSchema('unknown');
    expect(schema.type).toBe('object');
    expect(schema.properties?.config).toBeTruthy();
  });
});
