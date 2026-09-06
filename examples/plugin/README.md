# dim Function Plugin Examples

This directory contains reference implementations of dim function plugins, demonstrating both native (Go RPC) and WASM (sandboxed) runtimes.

## Overview

These examples show how to build third-party function plugins using only the public `pkg/sdk` interface — no need to read dim's internal code or use internal packages.

- **Native Go Plugin**: Subprocess-based via RPC, full computational power
- **WASM Rust Plugin**: Sandboxed WebAssembly module, portable and safe

Both implement the same logical contract defined in [design/M3.3.1_PLUGIN_SDK_SPEC.md](../../design/M3.3.1_PLUGIN_SDK_SPEC.md).

---

## Native Go Plugin

**Location**: `native-go/main.go`

### What It Implements

Three example functions:
1. `validate_email` — Email format validation
2. `enrich_customer` — Fetch customer enrichment data (simulated)
3. `transform_phone` — Format phone numbers

### Building

```bash
cd native-go
go build -o validate-email .
```

**Output**: `./validate-email` (executable)

### Using in a Route

```yaml
routes:
  order-processing:
    steps:
      - type: translate
        spec:
          expression: |
            {
              "email_valid": $validate_email(message.email),
              "customer": $enrich_customer(message.customer_id),
              "phone": $transform_phone(message.phone)
            }
          functions:
            - name: validate_email
              type: plugin
              runtime: native
              ref: /path/to/validate-email
            - name: enrich_customer
              type: plugin
              runtime: native
              ref: /path/to/validate-email
            - name: transform_phone
              type: plugin
              runtime: native
              ref: /path/to/validate-email
```

### Testing

Run conformance tests:
```bash
dimctl plugin conformance --native ./validate-email
```

### Key Points

- Uses only `github.com/naren-chakraview/dim/pkg/sdk` (public)
- No internal imports (`internal/expr`, `internal/engine`, etc.)
- Process isolation via subprocess
- JSON-RPC communication
- Implements `sdk.Function` interface
- Error handling via `error` return

---

## WASM Rust Plugin

**Location**: `wasm-rust/`

### What It Implements

Three example functions:
1. `validate_credit_card` — Credit card validation using Luhn algorithm
2. `validate_email` — Email format validation
3. `transform_phone` — Format phone numbers

### Building

Prerequisites:
```bash
rustup target add wasm32-unknown-unknown
```

Build:
```bash
cd wasm-rust
cargo build --target wasm32-unknown-unknown --release
```

**Output**: `target/wasm32-unknown-unknown/release/dim_wasm_plugin_example.wasm`

### Using in a Route

```yaml
routes:
  order-processing:
    steps:
      - type: filter
        spec:
          expression: $validate_credit_card(message.card).valid
          functions:
            - name: validate_credit_card
              type: plugin
              runtime: wasm
              ref: /path/to/plugin.wasm
```

### Testing

Run conformance tests:
```bash
dimctl plugin conformance --wasm ./target/wasm32-unknown-unknown/release/dim_wasm_plugin_example.wasm
```

### Key Points

- Uses standard Rust crates (`serde`, `serde_json`)
- Implements dim WASM ABI (linear memory I/O)
- Exports named functions with signature `(i32, i32) -> u64`
- Memory management via `malloc`/`free`
- JSON marshaling for I/O
- Deterministic computation (no floats, randomness, or side effects)
- SDK version export via `get_sdk_version()`
- Safe sandboxed execution

---

## WASM ABI Explained

### Function Signature

```rust
#[no_mangle]
pub extern "C" fn my_function(args_ptr: i32, args_len: i32) -> u64 {
    // Read arguments from memory at args_ptr (args_len bytes)
    // ... compute ...
    // Write result to memory via malloc
    // Return (result_ptr << 32) | result_len
}
```

### I/O Protocol

1. **Input**: Host writes JSON-serialized arguments to WASM linear memory
   - `args_ptr`: Pointer to JSON bytes
   - `args_len`: Byte count

2. **Processing**: Function reads from memory, processes

3. **Output**: Function allocates memory, writes JSON result
   - Result format: `{"result": <value>}` or `{"error": "<message>"}`

4. **Return**: `(result_ptr << 32) | result_len`

### Example

```rust
// Input: {"email": "alice@example.com"}
// Memory layout: [JSON bytes...]
//                ^args_ptr

unsafe {
    let args_bytes = std::slice::from_raw_parts(args_ptr as *const u8, args_len as usize);
    let args: Value = serde_json::from_slice(args_bytes)?;
    let email = args["email"].as_str()?;
    
    // Validate...
    let result = json!({"result": is_valid});
    
    // Allocate and write result
    let bytes = result.to_string().into_bytes();
    let ptr = malloc(bytes.len() as u32);
    std::ptr::copy_nonoverlapping(bytes.as_ptr(), ptr as *mut u8, bytes.len());
    
    // Return (ptr, len) packed as u64
    ((ptr as u64) << 32) | (bytes.len() as u64)
}
```

---

## Testing Your Plugin

### Run Conformance Tests

Both runtimes support conformance validation:

```bash
# Native plugin
dimctl plugin conformance --native /path/to/plugin

# WASM plugin
dimctl plugin conformance --wasm /path/to/module.wasm
```

**Tests verify**:
- ✅ Plugin loads without errors
- ✅ Handshake succeeds (native) / exports present (WASM)
- ✅ Simple function calls work
- ✅ Error handling works
- ✅ Timeout enforcement works
- ✅ Concurrent calls handled
- ✅ Output is JSON-serializable

### Manual Testing

**Native plugin** (Go):
```bash
./validate-email &
# Plugin starts; test via dimctl
```

**WASM plugin** (Rust):
```bash
# Use dimctl to call the plugin
dimctl plugin call --wasm ./plugin.wasm validate_email '{"email":"alice@example.com"}'
```

---

## SDK Versioning

Both examples target **SDK v1.0**, which is independent from the dim engine version.

- A plugin built against v1.0 works with engine v1.x
- Breaking changes only at major boundaries (v1 → v2)

Check plugin SDK version:
```bash
# In Go
import "github.com/naren-chakraview/dim/pkg/sdk"
fmt.Println(sdk.Version()) // "1.0"

# In Rust (WASM)
#[no_mangle]
pub extern "C" fn get_sdk_version() -> u32 {
    1 // Major version 1
}
```

---

## Common Patterns

### Error Handling

**Native (Go)**:
```go
func (p *MyPlugin) Call(args ...interface{}) (interface{}, error) {
    if len(args) == 0 {
        return nil, fmt.Errorf("argument required")
    }
    // ...
    return result, nil
}
```

**WASM (Rust)**:
```rust
let error = json!({"error": "argument required"});
let (ptr, len) = write_string(&error.to_string());
return ((ptr as u64) << 32) | (len as u64);
```

### Timeout Handling

The host enforces timeouts via `context.Context` (native) or background watchdog (WASM).

Plugins should:
- Complete quickly (default timeout: 5s)
- Avoid infinite loops
- Free resources on cancellation

### Determinism

For lineage and replay correctness, plugins must be deterministic:
- Same inputs → same outputs
- No floating point (or round consistently)
- No randomness
- No timestamps/date-based logic
- No external state

---

## Deployment

### Native Plugin

1. Build: `go build -o my-plugin`
2. Deploy: Copy binary to a path accessible to dimd
3. Configure: Reference in route config (`ref: /path/to/my-plugin`)
4. Test: Run conformance tests before production

### WASM Plugin

1. Build: `cargo build --target wasm32-unknown-unknown --release`
2. Deploy: Copy `.wasm` file to a path accessible to dimd
3. Configure: Reference in route config (`ref: /path/to/plugin.wasm`)
4. Test: Run conformance tests before production

---

## Troubleshooting

### Native Plugin Won't Load

- Check magic cookie verification: `DIM_PLUGIN=dim-function-plugin-v1`
- Verify binary path is absolute or resolvable
- Check file permissions (executable)
- Run conformance test: `dimctl plugin conformance --native <path>`

### WASM Plugin Won't Load

- Verify target: `wasm32-unknown-unknown` (not `wasm32-wasi`)
- Check SDK version export: `get_sdk_version() -> u32`
- Run conformance test: `dimctl plugin conformance --wasm <path>`

### Plugin Calls Timeout

- Reduce computation time (optimize algorithms)
- Increase configured timeout (if appropriate)
- Check for infinite loops or blocking I/O

### JSON Serialization Errors

- Ensure all outputs are JSON-serializable
- Native: Use `json.Marshal` on Go types
- WASM: Use `serde_json::to_string()` for all results

---

## Next Steps

- Read [design/M3.3.1_PLUGIN_SDK_SPEC.md](../../design/M3.3.1_PLUGIN_SDK_SPEC.md) for the formal contract
- Read [pkg/sdk](../../pkg/sdk) for the public Go interface
- Build your own plugin using these examples as a template
- Submit conformance tests before deploying to production

Happy plugin building! 🚀
