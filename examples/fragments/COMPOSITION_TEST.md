# Fragment Composition Demonstration

This directory demonstrates the fragment composition feature of the dim middleware.

## Fragment Files

1. **base.yaml** - Base route with HTTP source and file sink
   - HTTP source listening on `/ingest`
   - File sink writing to `/tmp/output.jsonl`
   - Single translate step (required by schema)

2. **auth.yaml** - Authorization layer that imports base.yaml
   - Imports base.yaml to inherit HTTP source and file sink
   - Adds an authorize step requiring the 'admin' role

3. **retry.yaml** - Transformation layer that imports base.yaml
   - Imports base.yaml to inherit HTTP source and file sink
   - Adds a filter step for active messages
   - Adds a translate step with enhanced transformation

4. **composed.yaml** - Full composition that imports retry.yaml
   - Imports retry.yaml (which itself imports base.yaml)
   - Adds RBAC authorization at the route level
   - Final route has all steps from all layers

## How Composition Works

When you load a route with `$import` directives:

1. **Fragment Resolution**: The loader reads the importing file and detects the `$import: path/to/fragment.yaml` directive
2. **Recursive Loading**: The referenced fragment is loaded, and if it also has imports, those are resolved recursively
3. **Cycle Detection**: Circular imports are detected and rejected with a clear error message
4. **Configuration Merging**: 
   - Sources/Sinks are merged by key (child definitions override parent)
   - Steps are concatenated (parent steps first, then child steps)
   - Other fields are overridden (child value wins)
4. **Validation**: The fully-resolved configuration is validated against the JSON Schema

## Merge Semantics

### Sources and Sinks
Child definitions override parent definitions for the same key. If a source/sink is defined in both parent and child with the same name, the child definition is used.

### Steps
Steps are concatenated in order:
1. Parent steps are added first
2. Child steps are added after

This allows fragments to layer processing steps on top of a base route.

### Other Fields
Fields like `version`, `auth`, `error_path`, etc. use override semantics where the child value completely replaces the parent value.

## Example Validation

All fragments validate successfully:

```bash
$ go run ./cmd/midctl validate ./examples/fragments/base.yaml
OK: valid route configuration
  Version: 1
  Sources: 1
  Sinks: 1
  Routes: 1

$ go run ./cmd/midctl validate ./examples/fragments/auth.yaml
OK: valid route configuration
  Version: 1
  Sources: 1
  Sinks: 1
  Routes: 1

$ go run ./cmd/midctl validate ./examples/fragments/retry.yaml
OK: valid route configuration
  Version: 1
  Sources: 1
  Sinks: 1
  Routes: 1

$ go run ./cmd/midctl validate ./examples/fragments/composed.yaml
OK: valid route configuration
  Version: 1
  Sources: 1
  Sinks: 1
  Routes: 1
```

## Testing Fragment Composition

Run the fragment tests to verify composition logic:

```bash
$ go test ./internal/config -v -run Fragment
=== RUN   TestLoadSingleFragment
--- PASS: TestLoadSingleFragment (0.00s)
=== RUN   TestResolveImportSingleLevel
--- PASS: TestResolveImportSingleLevel (0.00s)
=== RUN   TestResolveImportRecursive
--- PASS: TestResolveImportRecursive (0.00s)
=== RUN   TestCycleDetection
--- PASS: TestCycleDetection (0.00s)
```

## Benefits of Fragment Composition

1. **Reusability**: Define base infrastructure (HTTP source, file sink) once in base.yaml
2. **Layering**: Add authorization, retry logic, or transformations in separate fragments
3. **Composition**: Combine multiple fragments to build complex routes
4. **Maintenance**: Update base infrastructure in one place, all composed routes inherit the changes
5. **Testing**: Each fragment can be validated independently

## Example Use Cases

### Use Case 1: Base Route with Authorization
```yaml
# Compose auth.yaml to add RBAC to the base route
$import: base.yaml
routes:
  process:
    auth: none
    steps:
      - authorize:
          mode: rbac
          require_roles: [admin]
```

### Use Case 2: Base Route with Transformation
```yaml
# Compose retry.yaml to add filtering and transformation
$import: base.yaml
routes:
  process:
    auth: none
    steps:
      - filter:
          expr: 'body.status = "active"'
      - translate:
          expr: '{ id: body.id, message: body.message }'
```

### Use Case 3: Full Stack with Multiple Layers
```yaml
# Compose auth.yaml + transformation in one route
$import: auth.yaml
routes:
  process:
    auth: none
    steps:
      - filter:
          expr: 'body.status = "active"'
      - translate:
          expr: '{ id: body.id, message: body.message }'
```

This gives you a route that:
- Uses HTTP source from base.yaml
- Uses file sink from base.yaml
- Enforces admin role requirement from auth.yaml
- Filters active messages and transforms them with JSONata

## Relative Path Resolution

Import paths are resolved relative to the location of the importing file. For example:

```yaml
# File: /project/routes/auth.yaml
$import: ../base/common.yaml  # Resolves to /project/base/common.yaml
```

This allows you to organize fragments in a directory structure without worrying about absolute paths.
