import React, { useState } from 'react';
import { Canvas } from './components/Canvas';
import { RouteConfig } from './types/route';
import './App.css';

/**
 * Main Studio App Component
 *
 * Layout:
 * - Left: Canvas (DAG visualization)
 * - Right: Form panel (will be added in M4.5.3)
 * - Bottom: Validation panel (will be added in M4.5.5)
 */
export function App() {
  const [route, setRoute] = useState<RouteConfig | null>(null);
  const [selectedStep, setSelectedStep] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);

  const handleStepSelect = (stepName: string) => {
    setSelectedStep(stepName);
  };

  const handleRouteChange = (newRoute: RouteConfig) => {
    setRoute(newRoute);
    setIsDirty(true);
  };

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
            <div className="form-content">
              <h3>Configure: {selectedStep}</h3>
              <p>Schema-driven form will render here (M4.5.3)</p>
            </div>
          ) : (
            <div className="form-placeholder">
              <p>Select a step to configure</p>
            </div>
          )}
        </div>
      </div>

      {/* Validation Panel - Bottom */}
      <div className="validation-panel">
        <p>✓ Ready to save (validation will update here)</p>
      </div>
    </div>
  );
}

export default App;
