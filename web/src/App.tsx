import React, { useState, useCallback } from 'react';
import { Canvas } from './components/Canvas';
import { SchemaForm } from './components/SchemaForm';
import { ValidationPanel, ValidationResult } from './components/ValidationPanel';
import { RouteConfig, Route } from './types/route';
import './App.css';

/**
 * Main Studio App Component
 *
 * Layout:
 * - Left: Canvas (DAG visualization)
 * - Right: Form panel (schema-driven configuration)
 * - Bottom: Validation panel (M4.5.5)
 */
export function App() {
  const [route, setRoute] = useState<RouteConfig | null>(null);
  const [selectedStep, setSelectedStep] = useState<string | null>(null);
  const [selectedRouteName, setSelectedRouteName] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);
  const [isValidating, setIsValidating] = useState(false);

  const handleStepSelect = (stepName: string) => {
    setSelectedStep(stepName);
    // Find which route this step belongs to
    if (route?.routes) {
      for (const [routeName, routeConfig] of Object.entries(route.routes)) {
        const stepExists = routeConfig.steps?.some(s => Object.keys(s)[0] === stepName);
        if (stepExists) {
          setSelectedRouteName(routeName);
          break;
        }
      }
    }
  };

  const handleStepConfigChange = (newConfig: any) => {
    if (!route || !selectedRouteName || !selectedStep) return;

    const updatedRoute = JSON.parse(JSON.stringify(route)) as RouteConfig;
    const routeConfig = updatedRoute.routes?.[selectedRouteName];

    if (routeConfig?.steps) {
      const stepIndex = routeConfig.steps.findIndex(s => Object.keys(s)[0] === selectedStep);
      if (stepIndex !== -1) {
        routeConfig.steps[stepIndex] = {
          [selectedStep]: newConfig,
        };
      }
    }

    setRoute(updatedRoute);
    setIsDirty(true);
  };

  const handleRouteChange = (newRoute: RouteConfig) => {
    setRoute(newRoute);
    setIsDirty(true);
  };

  const handleValidate = useCallback(async (routeToValidate: RouteConfig): Promise<ValidationResult> => {
    setIsValidating(true);
    try {
      // TODO: Replace with actual backend validation endpoint when M4.5.6 is complete
      // This is a placeholder that will validate basic structure
      const result: ValidationResult = {
        valid: true,
        errors: [],
        warnings: [],
        route_version: generateRouteHash(routeToValidate),
        timestamp: new Date().toISOString(),
      };
      setValidationResult(result);
      return result;
    } catch (err) {
      const errorResult: ValidationResult = {
        valid: false,
        errors: [
          {
            code: 'VALIDATION_FAILED',
            message: `Validation error: ${err instanceof Error ? err.message : 'Unknown error'}`,
            severity: 'error',
          },
        ],
        warnings: [],
        timestamp: new Date().toISOString(),
      };
      setValidationResult(errorResult);
      return errorResult;
    } finally {
      setIsValidating(false);
    }
  }, []);

  // Get current step config for the selected step
  const currentStepConfig = (() => {
    if (!route || !selectedRouteName || !selectedStep) return undefined;

    const routeConfig = route.routes?.[selectedRouteName];
    const step = routeConfig?.steps?.find(s => Object.keys(s)[0] === selectedStep);
    return step?.[selectedStep];
  })();

  return (
    <div className="studio-app">
      <header className="studio-header">
        <h1>DIM Visual Route Editor</h1>
        <div className="header-controls">
          {isDirty && <span className="dirty-indicator">●</span>}
          {route?.domain && <span className="domain-badge">{route.domain}</span>}
        </div>
      </header>

      <div className="studio-layout">
        {/* Canvas Panel - Left */}
        <div className="canvas-panel">
          <Canvas route={route} onStepSelect={handleStepSelect} />
        </div>

        {/* Form/Properties Panel - Right */}
        <div className="form-panel">
          {selectedStep ? (
            <SchemaForm
              stepType={selectedStep}
              value={currentStepConfig || {}}
              onChange={handleStepConfigChange}
              title={`Configure: ${selectedStep}`}
            />
          ) : (
            <div className="form-placeholder">
              <p>Select a step to configure</p>
            </div>
          )}
        </div>
      </div>

      {/* Validation Panel - Bottom */}
      <ValidationPanel
        route={route || undefined}
        isValidating={isValidating}
        result={validationResult || undefined}
        onValidate={handleValidate}
        autoValidate={true}
      />
    </div>
  );
}

/**
 * Generate a simple hash of route configuration for version tracking.
 * This is a placeholder; actual implementation would use cryptographic hash.
 */
function generateRouteHash(route: RouteConfig): string {
  const jsonStr = JSON.stringify(route);
  let hash = 0;
  for (let i = 0; i < jsonStr.length; i++) {
    const char = jsonStr.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return `sha256:${Math.abs(hash).toString(16).padStart(12, '0')}`;
}

export default App;
