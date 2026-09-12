import { describe, it, expect } from 'vitest';
import { validateJSONata, extractVariablesFromExpression, getJSONataCompletions } from '../JSONataEditor';

describe('JSONataEditor Helpers', () => {
  describe('validateJSONata', () => {
    it('accepts valid expressions', () => {
      expect(validateJSONata('$payload.id')).toBe(true);
      expect(validateJSONata('$payload.amount * 1.1')).toBe(true);
      expect(validateJSONata('$payload.status = "active"')).toBe(true);
      expect(validateJSONata('$sum($payload.items.price)')).toBe(true);
    });

    it('accepts empty expression', () => {
      expect(validateJSONata('')).toBe(true);
      expect(validateJSONata('   ')).toBe(true);
    });

    it('rejects unbalanced parentheses', () => {
      expect(validateJSONata('$sum($payload.items.price')).toBe(false);
      expect(validateJSONata('$payload.id)')).toBe(false);
    });

    it('rejects unbalanced brackets', () => {
      expect(validateJSONata('$payload[0')).toBe(false);
      expect(validateJSONata('$payload.items]')).toBe(false);
    });

    it('rejects unbalanced braces', () => {
      expect(validateJSONata('{ "key": value')).toBe(false);
      expect(validateJSONata('}')).toBe(false);
    });

    it('rejects unbalanced quotes', () => {
      expect(validateJSONata('$payload.name = "test')).toBe(false);
      expect(validateJSONata("$payload.id = 'test")).toBe(false);
    });

    it('rejects incomplete expressions', () => {
      expect(validateJSONata('$payload.id.')).toBe(false);
      expect(validateJSONata('$payload(')).toBe(false);
    });
  });

  describe('extractVariablesFromExpression', () => {
    it('extracts simple variables', () => {
      const vars = extractVariablesFromExpression('$payload.id');
      expect(vars).toContain('$payload');
    });

    it('extracts multiple variables', () => {
      const vars = extractVariablesFromExpression('$payload.id & $context.user');
      expect(vars).toContain('$payload');
      expect(vars).toContain('$context');
    });

    it('handles nested property access', () => {
      const vars = extractVariablesFromExpression('$payload.user.profile.name');
      expect(vars).toContain('$payload');
      expect(vars).not.toContain('$user');
    });

    it('returns empty for no variables', () => {
      const vars = extractVariablesFromExpression('"constant string"');
      expect(vars).toHaveLength(0);
    });

    it('handles complex expressions', () => {
      const vars = extractVariablesFromExpression(
        '$sum($payload.items.price) + $context.tax * $payload.subtotal'
      );
      expect(vars).toContain('$payload');
      expect(vars).toContain('$context');
    });
  });

  describe('getJSONataCompletions', () => {
    it('suggests payload fields after $payload.', () => {
      const completions = getJSONataCompletions('$payload.', 9);
      expect(completions.length).toBeGreaterThan(0);
      expect(completions.some(c => c.label === 'id')).toBe(true);
    });

    it('suggests functions after $', () => {
      const completions = getJSONataCompletions('$', 1);
      expect(completions.length).toBeGreaterThan(0);
      expect(completions.some(c => c.label.includes('sum'))).toBe(true);
    });

    it('returns empty when not in completion context', () => {
      const completions = getJSONataCompletions('$payload.id', 10);
      expect(completions).toHaveLength(0);
    });

    it('completion items have label and detail', () => {
      const completions = getJSONataCompletions('$payload.', 9);
      completions.forEach(c => {
        expect(c.label).toBeTruthy();
        expect(c.detail).toBeTruthy();
      });
    });

    it('includes common JSONata functions', () => {
      const completions = getJSONataCompletions('$', 1);
      const funcLabels = completions.map(c => c.label);

      expect(funcLabels.some(l => l.includes('sum'))).toBe(true);
      expect(funcLabels.some(l => l.includes('count'))).toBe(true);
      expect(funcLabels.some(l => l.includes('map'))).toBe(true);
    });
  });
});
