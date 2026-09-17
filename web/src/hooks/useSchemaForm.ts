import { useMemo, useCallback } from 'react';

export type FormFieldType =
  | 'text'
  | 'number'
  | 'boolean'
  | 'select'
  | 'textarea'
  | 'object'
  | 'array';

export interface JSONSchema {
  type?: string | string[];
  title?: string;
  description?: string;
  enum?: any[];
  properties?: Record<string, JSONSchema>;
  required?: string[];
  items?: JSONSchema;
  $ref?: string;
  oneOf?: JSONSchema[];
  anyOf?: JSONSchema[];
  allOf?: JSONSchema[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  default?: any;
  examples?: any[];
}

export interface FormField {
  name: string;
  type: FormFieldType;
  label: string;
  description?: string;
  required: boolean;
  placeholder?: string;
  options?: Array<{ label: string; value: any }>;
  min?: number;
  max?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  defaultValue?: any;
  nested?: FormField[];
}

/**
 * Infer form field type from JSON schema.
 * Maps JSON schema types to UI input types.
 */
function inferFieldType(schema: JSONSchema): FormFieldType {
  if (!schema) return 'text';

  // Handle oneOf/anyOf/allOf - treat as complex object
  if (schema.oneOf || schema.anyOf || schema.allOf) {
    return 'select';
  }

  // Handle enum - render as dropdown
  if (schema.enum) {
    return 'select';
  }

  const schemaType = schema.type;

  if (schemaType === 'boolean') return 'boolean';
  if (schemaType === 'integer' || schemaType === 'number') return 'number';
  if (schemaType === 'object') return 'object';
  if (schemaType === 'array') return 'array';

  // String variants
  if (schemaType === 'string') {
    // Long strings → textarea
    if (schema.maxLength && schema.maxLength > 500) return 'textarea';
    // Multi-line patterns → textarea
    if (schema.pattern && schema.pattern.includes('\n')) return 'textarea';
    return 'text';
  }

  // Default to text
  return 'text';
}

/**
 * Convert enum to select options.
 */
function enumToOptions(enumValues: any[] = []): Array<{ label: string; value: any }> {
  return enumValues.map(value => ({
    label: String(value),
    value,
  }));
}

/**
 * Parse schema properties into form fields.
 */
function schemaToFields(
  schema: JSONSchema,
  fieldName?: string
): FormField[] {
  if (!schema || !schema.properties) {
    return [];
  }

  const requiredFields = schema.required || [];
  const fields: FormField[] = [];

  Object.entries(schema.properties).forEach(([name, fieldSchema]) => {
    const isRequired = requiredFields.includes(name);
    const type = inferFieldType(fieldSchema);
    const options = fieldSchema.enum ? enumToOptions(fieldSchema.enum) : undefined;

    const field: FormField = {
      name,
      type,
      label: fieldSchema.title || formatLabel(name),
      description: fieldSchema.description,
      required: isRequired,
      placeholder: getPlaceholder(type, fieldSchema),
      options,
      min: fieldSchema.minimum,
      max: fieldSchema.maximum,
      minLength: fieldSchema.minLength,
      maxLength: fieldSchema.maxLength,
      pattern: fieldSchema.pattern,
      defaultValue: fieldSchema.default,
    };

    // For nested objects, recursively extract fields
    if (type === 'object' && fieldSchema.properties) {
      field.nested = schemaToFields(fieldSchema);
    }

    fields.push(field);
  });

  return fields;
}

/**
 * Format field name for display (snake_case → Title Case).
 */
function formatLabel(name: string): string {
  return name
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

/**
 * Get appropriate placeholder for field type.
 */
function getPlaceholder(type: FormFieldType, schema: JSONSchema): string {
  if (schema.examples && schema.examples.length > 0) {
    return `e.g., ${schema.examples[0]}`;
  }

  switch (type) {
    case 'text':
      return 'Enter text...';
    case 'number':
      return 'Enter a number...';
    case 'textarea':
      return 'Enter text...';
    case 'select':
      return 'Select an option...';
    default:
      return '';
  }
}

/**
 * Custom hook for schema-driven form generation.
 *
 * Usage:
 * ```
 * const { fields, validate } = useSchemaForm(schema, currentValue);
 * ```
 */
export function useSchemaForm(schema?: JSONSchema, currentValue?: any) {
  const fields = useMemo(() => {
    if (!schema) return [];
    return schemaToFields(schema);
  }, [schema]);

  const validate = useCallback((value: any): { valid: boolean; errors: string[] } => {
    if (!schema) return { valid: true, errors: [] };

    const errors: string[] = [];
    const required = schema.required || [];

    // Check required fields
    required.forEach(fieldName => {
      if (!value || value[fieldName] === undefined || value[fieldName] === '') {
        errors.push(`${formatLabel(fieldName)} is required`);
      }
    });

    // Check field constraints
    Object.entries(schema.properties || {}).forEach(([fieldName, fieldSchema]) => {
      const fieldValue = value?.[fieldName];
      if (fieldValue === undefined || fieldValue === null) return;

      // Type checking
      if (fieldSchema.type === 'number' || fieldSchema.type === 'integer') {
        if (typeof fieldValue !== 'number') {
          errors.push(`${formatLabel(fieldName)} must be a number`);
        }
        if (fieldSchema.minimum !== undefined && fieldValue < fieldSchema.minimum) {
          errors.push(`${formatLabel(fieldName)} must be >= ${fieldSchema.minimum}`);
        }
        if (fieldSchema.maximum !== undefined && fieldValue > fieldSchema.maximum) {
          errors.push(`${formatLabel(fieldName)} must be <= ${fieldSchema.maximum}`);
        }
      }

      if (fieldSchema.type === 'string') {
        if (typeof fieldValue !== 'string') {
          errors.push(`${formatLabel(fieldName)} must be text`);
        }
        if (fieldSchema.minLength && fieldValue.length < fieldSchema.minLength) {
          errors.push(`${formatLabel(fieldName)} must be at least ${fieldSchema.minLength} characters`);
        }
        if (fieldSchema.maxLength && fieldValue.length > fieldSchema.maxLength) {
          errors.push(`${formatLabel(fieldName)} must be at most ${fieldSchema.maxLength} characters`);
        }
        if (fieldSchema.pattern) {
          const regex = new RegExp(fieldSchema.pattern);
          if (!regex.test(fieldValue)) {
            errors.push(`${formatLabel(fieldName)} format is invalid`);
          }
        }
        if (fieldSchema.enum && !fieldSchema.enum.includes(fieldValue)) {
          errors.push(`${formatLabel(fieldName)} must be one of: ${fieldSchema.enum.join(', ')}`);
        }
      }
    });

    return {
      valid: errors.length === 0,
      errors,
    };
  }, [schema]);

  return { fields, validate };
}

// Global cache for the full route schema
let cachedRouteSchema: JSONSchema | null = null;
let schemaFetchPromise: Promise<JSONSchema | null> | null = null;

/**
 * Fetch the route schema from the backend.
 */
async function fetchRouteSchema(): Promise<JSONSchema | null> {
  // Return cached schema if available
  if (cachedRouteSchema) {
    return cachedRouteSchema;
  }

  // Return pending fetch if already in progress
  if (schemaFetchPromise) {
    return schemaFetchPromise;
  }

  // Fetch schema from backend
  schemaFetchPromise = (async () => {
    try {
      const response = await fetch('/api/schema');
      if (!response.ok) {
        console.warn('Failed to fetch schema from backend');
        return null;
      }
      cachedRouteSchema = await response.json();
      return cachedRouteSchema;
    } catch (error) {
      console.warn('Error fetching schema:', error);
      return null;
    }
  })();

  return schemaFetchPromise;
}

/**
 * Extract a step schema from the full route schema.
 * Step schemas are stored in definitions as "{stepType}_step".
 */
function extractStepSchemaFromFull(fullSchema: JSONSchema | null, stepType: string): JSONSchema {
  if (!fullSchema || !fullSchema.definitions) {
    return getDefaultStepSchema(stepType);
  }

  const definitions = fullSchema.definitions as Record<string, JSONSchema>;

  // Try the exact step name (e.g., "translate_step", "filter_step")
  let schema = definitions[`${stepType}_step`];

  // If not found and it's a known type, extract from definitions
  if (!schema && stepType === 'translate') {
    schema = definitions['translate_step'];
  } else if (!schema && stepType === 'filter') {
    schema = definitions['filter_step'];
  } else if (!schema && stepType === 'route') {
    schema = definitions['route_step'];
  } else if (!schema && stepType === 'wiretap') {
    schema = definitions['wiretap_step'];
  } else if (!schema && stepType === 'idempotent') {
    schema = definitions['idempotent_step'];
  } else if (!schema && stepType === 'authorize') {
    schema = definitions['authorize_step'];
  }

  return schema || getDefaultStepSchema(stepType);
}

/**
 * Get default/fallback schema for unknown step types.
 */
function getDefaultStepSchema(stepType: string): JSONSchema {
  return {
    type: 'object',
    title: `${stepType} Step`,
    description: `Configuration for ${stepType} step`,
    properties: {
      [stepType]: {
        type: 'object',
        title: 'Configuration',
        description: `${stepType} configuration object`,
        properties: {},
      },
    },
    required: [stepType],
  };
}

/**
 * Get schema for a specific step type.
 * Fetches from backend if available, falls back to defaults.
 */
export async function getStepSchema(stepType: string): Promise<JSONSchema> {
  const fullSchema = await fetchRouteSchema();
  return extractStepSchemaFromFull(fullSchema, stepType);
}

/**
 * Synchronous version for backward compatibility.
 * Returns default schema immediately; actual schema loads in background.
 */
export function getStepSchemaSync(stepType: string): JSONSchema {
  // Return default immediately
  return getDefaultStepSchema(stepType);
}
