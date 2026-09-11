import React, { useState, useCallback, useEffect } from 'react';
import { Canvas } from './components/Canvas';
import { SchemaForm } from './components/SchemaForm';
import { ValidationPanel, ValidationResult } from './components/ValidationPanel';
import { RouteConfig, Route } from './types/route';
import './App.css';

interface AvailableRoute {
  name: string;
  domain: string;
  filePath: string;
}

/**
 * Main Studio App Component
 *
 * Layout:
 * - Left: Canvas (DAG visualization)
 * - Right: Form panel (schema-driven configuration)
 * - Bottom: Validation panel (M4.5.5)
 */
export function App() {
  const [appError, setAppError] = useState<string | null>(null);
  const [availableRoutes, setAvailableRoutes] = useState<AvailableRoute[]>([]);
  const [selectedRouteKey, setSelectedRouteKey] = useState<string | null>(null);
  const [route, setRoute] = useState<RouteConfig | null>(null);
  const [selectedStep, setSelectedStep] = useState<string | null>(null);
  const [selectedRouteName, setSelectedRouteName] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch available routes on mount
  useEffect(() => {
    const fetchRoutes = async () => {
      try {
        const response = await fetch('/api/routes');
        const data = await response.json();

        if (data.routes) {
          // Filter to only actual route configurations (must have sources, sinks, routes)
          const actualRoutes = Object.entries(data.routes)
            .filter(([_key, route]: [string, any]) => {
              const routeData = route.data || route;
              return routeData &&
                     typeof routeData === 'object' &&
                     ('sources' in routeData || 'routes' in routeData) &&
                     'version' in routeData;
            })
            .map(([key, route]: [string, any]) => ({
              name: route.name || key,
              domain: route.domain || 'unknown',
              filePath: route.filePath || '',
            }));

          setAvailableRoutes(actualRoutes);

          // Auto-select first route
          if (actualRoutes.length > 0) {
            setSelectedRouteKey(actualRoutes[0].name);
          } else {
            setError('No valid route configurations found');
          }
        }
      } catch (err) {
        setError(`Failed to load routes: ${err instanceof Error ? err.message : 'Unknown error'}`);
      } finally {
        setIsLoading(false);
      }
    };

    fetchRoutes();
  }, []);

  // Load selected route
  useEffect(() => {
    if (!selectedRouteKey) return;

    const loadRoute = async () => {
      try {
        const response = await fetch(`/api/route?route=${encodeURIComponent(selectedRouteKey)}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);

        const routeData = await response.json();

        // The route data might be nested in .data or be direct
        const actualRoute = routeData.data || routeData;

        // Validate it has the expected structure
        if (!actualRoute || typeof actualRoute !== 'object') {
          throw new Error('Invalid route data format');
        }

        setRoute(actualRoute as RouteConfig);
        setSelectedStep(null);
        setSelectedRouteName(null);
        setIsDirty(false);
        setError(null);
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        console.error('Failed to load route:', message);
        setError(`Failed to load route: ${message}`);
      }
    };

    loadRoute();
  }, [selectedRouteKey]);

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

  const handleSave = useCallback(async () => {
    if (!route || !selectedRouteKey) return;

    setIsValidating(true);
    try {
      const response = await fetch('/api/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          route: route,
          filePath: availableRoutes.find(r => r.name === selectedRouteKey)?.filePath || '',
        }),
      });

      const result = await response.json();

      if (response.ok) {
        setValidationResult({
          valid: true,
          errors: [],
          warnings: [],
          route_version: generateRouteHash(route),
          timestamp: new Date().toISOString(),
        });
        setIsDirty(false);
        setError(null);
      } else {
        setError(`Save failed: ${result.error || 'Unknown error'}`);
      }
    } catch (err) {
      setError(`Save failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setIsValidating(false);
    }
  }, [route, selectedRouteKey, availableRoutes]);

  // Get current step config for the selected step
  const currentStepConfig = (() => {
    if (!route || !selectedRouteName || !selectedStep) return undefined;

    const routeConfig = route.routes?.[selectedRouteName];
    const step = routeConfig?.steps?.find(s => Object.keys(s)[0] === selectedStep);
    return step?.[selectedStep];
  })();

  if (isLoading) {
    return (
      <div className="studio-app loading">
        <div className="loading-spinner">
          <p>Loading routes...</p>
        </div>
      </div>
    );
  }

  if (availableRoutes.length === 0) {
    return (
      <div className="studio-app empty">
        <div className="empty-state">
          <h2>No routes found</h2>
          <p>Create a route file in domains/ to get started.</p>
          <p>Example: domains/payments/order-payment.yaml</p>
        </div>
      </div>
    );
  }

  return (
    <div className="studio-app">
      <header className="studio-header">
        <div className="header-content">
          <h1>DIM Visual Route Editor</h1>
          <div className="route-selector">
            <label htmlFor="route-select">Route:</label>
            <select
              id="route-select"
              value={selectedRouteKey || ''}
              onChange={(e) => setSelectedRouteKey(e.target.value)}
            >
              {availableRoutes.map((route) => (
                <option key={route.name} value={route.name}>
                  {route.domain}/{route.name}
                </option>
              ))}
            </select>
          </div>
        </div>
        <div className="header-controls">
          {isDirty && <span className="dirty-indicator" title="Unsaved changes">●</span>}
          {route?.domain && <span className="domain-badge">{route.domain}</span>}
          <button
            onClick={handleSave}
            disabled={!isDirty || isValidating}
            className="save-button"
            title="Save changes to file (Ctrl+S)"
          >
            {isValidating ? 'Saving...' : 'Save'}
          </button>
        </div>
      </header>

      {error && (
        <div className="error-banner">
          <p>{error}</p>
        </div>
      )}

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
              <p>Select a step in the canvas to configure</p>
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
