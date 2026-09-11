import React, { useState, useCallback } from 'react';
import { useSchemaForm, FormField, JSONSchema, getStepSchema } from '../hooks/useSchemaForm';
import { JSONataEditor } from './JSONataEditor';
import '../styles/SchemaForm.css';

interface SchemaFormProps {
  stepType?: string;
  schema?: JSONSchema;
  value?: any;
  onChange?: (value: any) => void;
  title?: string;
}

/**
 * SchemaForm component renders auto-generated form inputs from JSON schema.
 *
 * Usage:
 * ```
 * <SchemaForm
 *   stepType="translate"
 *   value={stepConfig}
 *   onChange={(newConfig) => updateRoute(...)}
 *   title="Configure: translate"
 * />
 * ```
 */
export function SchemaForm({
  stepType,
  schema: customSchema,
  value = {},
  onChange,
  title,
}: SchemaFormProps) {
  const [errors, setErrors] = useState<string[]>([]);
  const [touched, setTouched] = useState<Set<string>>(new Set());

  // Get schema from step type or use custom schema
  const schema = customSchema || (stepType ? getStepSchema(stepType) : undefined);
  const { fields, validate } = useSchemaForm(schema, value);

  const handleChange = useCallback(
    (fieldName: string, newValue: any) => {
      const updated = { ...value, [fieldName]: newValue };
      onChange?.(updated);

      // Validate on change
      const validation = validate(updated);
      if (!validation.valid) {
        setErrors(validation.errors.filter(err => err.includes(fieldName)));
      } else {
        setErrors([]);
      }
    },
    [value, onChange, validate]
  );

  const handleBlur = useCallback(
    (fieldName: string) => {
      setTouched(prev => new Set(prev).add(fieldName));

      // Validate on blur
      const validation = validate(value);
      if (!validation.valid) {
        setErrors(validation.errors);
      }
    },
    [value, validate]
  );

  if (!schema || fields.length === 0) {
    return (
      <div className="schema-form empty">
        <p>No configuration available for this step</p>
      </div>
    );
  }

  return (
    <div className="schema-form">
      {title && <h3 className="form-title">{title}</h3>}

      <div className="form-fields">
        {fields.map(field => (
          <FormFieldInput
            key={field.name}
            field={field}
            value={value[field.name]}
            onChange={(newValue) => handleChange(field.name, newValue)}
            onBlur={() => handleBlur(field.name)}
            error={errors.find(err => err.includes(field.name))}
            touched={touched.has(field.name)}
          />
        ))}
      </div>

      {errors.length > 0 && (
        <div className="form-errors">
          <div className="error-header">⚠️ Validation Errors:</div>
          <ul>
            {errors.map((error, idx) => (
              <li key={idx}>{error}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

interface FormFieldInputProps {
  field: FormField;
  value?: any;
  onChange: (value: any) => void;
  onBlur: () => void;
  error?: string;
  touched: boolean;
}

/**
 * Renders individual form field based on type.
 */
function FormFieldInput({
  field,
  value,
  onChange,
  onBlur,
  error,
  touched,
}: FormFieldInputProps) {
  const hasError = touched && error;

  return (
    <div className={`form-field ${field.type} ${hasError ? 'has-error' : ''}`}>
      <label className={field.required ? 'required' : ''}>
        {field.label}
        {field.required && <span className="required-indicator">*</span>}
      </label>

      {field.description && <p className="field-description">{field.description}</p>}

      {renderInput(field, value, onChange, onBlur)}

      {hasError && <div className="field-error">{error}</div>}
    </div>
  );
}

/**
 * Check if a field is for JSONata expressions.
 * JSONata fields have "expr", "expression", or "condition" in their name.
 */
function isJSONataField(fieldName: string): boolean {
  const name = fieldName.toLowerCase();
  return name.includes('expr') || name.includes('expression') || name.includes('condition');
}

/**
 * Render appropriate input based on field type.
 */
function renderInput(
  field: FormField,
  value: any,
  onChange: (value: any) => void,
  onBlur: () => void
) {
  // Use JSONataEditor for expression/condition fields
  if (isJSONataField(field.name)) {
    return (
      <JSONataEditor
        value={value || ''}
        onChange={onChange}
        onBlur={onBlur}
        placeholder={field.placeholder}
        height="150px"
      />
    );
  }

  switch (field.type) {
    case 'text':
      return (
        <input
          type="text"
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          onBlur={onBlur}
          placeholder={field.placeholder}
          maxLength={field.maxLength}
        />
      );

    case 'number':
      return (
        <input
          type="number"
          value={value ?? ''}
          onChange={(e) => onChange(e.target.value ? Number(e.target.value) : '')}
          onBlur={onBlur}
          placeholder={field.placeholder}
          min={field.min}
          max={field.max}
        />
      );

    case 'textarea':
      return (
        <textarea
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          onBlur={onBlur}
          placeholder={field.placeholder}
          maxLength={field.maxLength}
          rows={4}
        />
      );

    case 'boolean':
      return (
        <input
          type="checkbox"
          checked={value || false}
          onChange={(e) => onChange(e.target.checked)}
          onBlur={onBlur}
        />
      );

    case 'select':
      return (
        <select value={value || ''} onChange={(e) => onChange(e.target.value)} onBlur={onBlur}>
          <option value="">{field.placeholder || 'Select an option...'}</option>
          {field.options?.map(opt => (
            <option key={String(opt.value)} value={String(opt.value)}>
              {opt.label}
            </option>
          ))}
        </select>
      );

    case 'object':
      return (
        <div className="nested-form">
          {field.nested?.map(nestedField => (
            <FormFieldInput
              key={nestedField.name}
              field={nestedField}
              value={value?.[nestedField.name]}
              onChange={(newValue) => onChange({ ...value, [nestedField.name]: newValue })}
              onBlur={onBlur}
              touched={true}
            />
          ))}
        </div>
      );

    case 'array':
      return (
        <input
          type="text"
          value={Array.isArray(value) ? value.join(', ') : ''}
          onChange={(e) => onChange(e.target.value.split(',').map(v => v.trim()))}
          onBlur={onBlur}
          placeholder="Comma-separated values"
        />
      );

    default:
      return <input type="text" value={value || ''} onChange={(e) => onChange(e.target.value)} />;
  }
}
