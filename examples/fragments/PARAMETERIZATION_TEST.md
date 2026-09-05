# Fragment Parameterization Examples (M2.4.3)

This directory demonstrates fragment parameterization — reusing a single fragment file with different parameter values across multiple routes.

## Files

### `retry-policy-parameterized.yaml`
A reusable retry policy fragment with substitutable parameters:
- `maxRetries`: Number of retry attempts (default: 3)
- `backoffMultiplier`: Exponential backoff multiplier (default: 2.0)
- `maxBackoffSec`: Maximum backoff duration in seconds (default: 300)

### `route-conservative-retry.yaml`
Route demonstrating conservative retry settings by overriding parameters:
- `maxRetries: 2` — Only 2 retries (vs. default 3)
- `backoffMultiplier: 1.5` — Slower backoff (vs. default 2.0)
- `maxBackoffSec: 60` — Max 1 minute backoff (vs. default 300)

**Use case:** Non-critical order processing where quick failure is preferred over long retries

### `route-aggressive-retry.yaml`
Route demonstrating aggressive retry settings for critical transactions:
- `maxRetries: 10` — Many retries (vs. default 3)
- `backoffMultiplier: 3.0` — Faster backoff growth (vs. default 2.0)
- `maxBackoffSec: 600` — Max 10 minutes (vs. default 300)

**Use case:** Payment processing where reliability is critical and longer waits are acceptable

## How Parameterization Works

### Step 1: Define a Parameterized Fragment
```yaml
# retry-policy-parameterized.yaml
$params:
  maxRetries: 3           # Defaults
  backoffMultiplier: 2.0
  maxBackoffSec: 300

steps:
  - name: retry-handler
    type: idempotent
    config:
      max_retries: ${PARAM:maxRetries}      # Parameter reference
      backoff_multiplier: ${PARAM:backoffMultiplier}
      max_backoff_sec: ${PARAM:maxBackoffSec}
```

### Step 2: Import and Override in Routes
```yaml
# route-conservative-retry.yaml
$import: retry-policy-parameterized.yaml

$params:
  maxRetries: 2           # Override default
  backoffMultiplier: 1.5  # Override default
  maxBackoffSec: 60       # Override default

routes:
  orders-conservative:
    # When composed, this route's parameters override fragment defaults
    # Result: max_retries=2, backoff_multiplier=1.5, max_backoff_sec=60
```

### Step 3: Same Fragment, Different Behavior
The same `retry-policy-parameterized.yaml` fragment is used in both routes but with different parameters:
- `route-conservative-retry.yaml` → Conservative settings (2 retries, 1.5x backoff)
- `route-aggressive-retry.yaml` → Aggressive settings (10 retries, 3.0x backoff)

## Testing Parameterization

### Validate without parameters (uses defaults)
```bash
dimctl validate retry-policy-parameterized.yaml
# Uses: maxRetries=3, backoffMultiplier=2.0, maxBackoffSec=300
```

### Validate with overrides
```bash
dimctl validate route-conservative-retry.yaml
# Uses: maxRetries=2, backoffMultiplier=1.5, maxBackoffSec=60
```

```bash
dimctl validate route-aggressive-retry.yaml
# Uses: maxRetries=10, backoffMultiplier=3.0, maxBackoffSec=600
```

### Verify parameters were substituted correctly
```bash
dimctl compose route-conservative-retry.yaml | grep max_retries
# Should show: max_retries: 2

dimctl compose route-aggressive-retry.yaml | grep max_retries
# Should show: max_retries: 10
```

## Benefits

1. **DRY (Don't Repeat Yourself)** — Define retry policy once, reuse with different values
2. **Consistency** — All routes use the same retry step definition, reducing configuration drift
3. **Maintainability** — Update retry logic in one place (fragment), affects all routes
4. **Flexibility** — Different routes can optimize for their use cases (conservative vs aggressive)

## Exit Criteria (M2.4.3) Verified

✅ Same fragment file imported twice with different parameters  
✅ Two different compiled routes produced (conservative vs aggressive)  
✅ Parameters correctly substituted in final composed config  
✅ Examples demonstrate practical use case (retry policies)  
