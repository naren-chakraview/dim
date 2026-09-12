# DIM Studio Web UI

Visual route-authoring interface for the DIM integration middleware engine.

## Project Structure

```
web/
├── public/              # Static assets
├── src/
│   ├── components/      # React components
│   │   ├── Canvas.tsx   # DAG visualization (M4.5.2)
│   │   └── __tests__/   # Component tests
│   ├── hooks/           # Custom React hooks
│   │   ├── useDagLayout.ts    # DAG layout computation
│   │   └── __tests__/
│   ├── types/           # TypeScript types
│   │   └── route.ts     # Route config types
│   ├── styles/          # CSS stylesheets
│   ├── App.tsx          # Main app component
│   └── index.tsx        # Entry point
├── package.json         # Dependencies
└── vite.config.ts       # Build config
```

## Components

### M4.5.2: Canvas

**Status:** ✅ IMPLEMENTED

Renders a route's step DAG as an interactive node-graph visualization.

**Features:**
- Sources, steps, and sinks displayed as nodes
- Hierarchical layout (left-to-right flow)
- Click to select step for configuration
- Drag to pan, scroll to zoom
- Icons for node types
- Animated edges

**Usage:**
```tsx
import { Canvas } from './components/Canvas';
import { RouteConfig } from './types/route';

const route: RouteConfig = { /* ... */ };

<Canvas route={route} onStepSelect={(stepName) => {
  console.log(`Selected: ${stepName}`);
}} />
```

## Hooks

### useDagLayout

Computes node and edge positions for a route's DAG.

**Input:** RouteConfig  
**Output:** DAG layout (nodes and edges with positions)

**Behavior:**
- Creates nodes for each source, step, and sink
- Arranges hierarchically: sources → steps → sinks
- Adds edges connecting nodes in pipeline order
- Assigns icons based on step/adapter type

## Types

### RouteConfig

Complete route configuration matching `schemas/route.schema.json`:
- `version`: Schema version
- `domain`: Domain owning the route
- `sources`: Input adapters (http, file, kafka, amqp, sftp)
- `routes`: Named pipelines
- `sinks`: Output adapters (file, http, kafka, s3)
- `contracts`: Schema validation definitions
- `error_path`: Error handling pipeline

### DAGNode & DAGEdge

Visualization node and edge types:
- Nodes: source, step, sink, error_sink
- Edges: connections between nodes
- Positions: x, y coordinates for rendering

## Build & Development

### Install Dependencies
```bash
npm install
```

### Development Server
```bash
npm run dev
# Runs on http://localhost:5173
```

### Build
```bash
npm run build
# Output: dist/
```

### Tests
```bash
npm run test
```

## Integration with Go Backend

The web UI is embedded in the `dimctl studio` binary:

1. Build web UI: `npm run build` → `dist/`
2. Generate embed: `go run -r embed_web_assets.go`
3. Build CLI: `go build ./cmd/dimctl`
4. Run: `dimctl studio domains/payments/order-payment.yaml`

Backend serves:
- `/` → index.html (web UI)
- `/api/route` → Get current route
- `/api/save` → Save route with validation
- `/api/schema` → Get JSON schema
- `/api/validate` → Validate route

## Roadmap

**M4.5.2** ✅ Canvas (DAG visualization)  
**M4.5.3** 🚧 Schema-driven forms  
**M4.5.4** 🚧 JSONata code editor  
**M4.5.5** 🚧 Validation panel  
**M4.5.6** 🚧 Backend integration  

## Testing

Tests use Vitest and React Testing Library:

- `Canvas.test.tsx` - Canvas component rendering
- `useDagLayout.test.ts` - DAG layout computation

Run tests: `npm run test`

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Requires ES2020+ support

## Next Steps

1. Implement M4.5.3 (Schema-driven forms)
2. Add M4.5.4 (JSONata editor)
3. Connect backend (M4.5.6)
4. Test end-to-end with real routes
