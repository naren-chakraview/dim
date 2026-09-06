// Reference implementation: WASM function plugin in Rust
//
// This is a complete, working example of a dim function plugin using WebAssembly.
// It demonstrates:
// - Implementing the dim WASM ABI (linear memory I/O)
// - JSON marshaling/unmarshaling in WASM
// - Memory allocation via malloc/free
// - SDK version export
//
// Building:
//   rustup target add wasm32-unknown-unknown
//   cargo build --target wasm32-unknown-unknown --release
//
// Output:
//   target/wasm32-unknown-unknown/release/dim_wasm_plugin_example.wasm
//
// Using in a route:
//   routes:
//     order-processing:
//       steps:
//         - type: translate
//           spec:
//             expression: $validate_credit_card(message.card)
//             functions:
//               - name: validate_credit_card
//                 type: plugin
//                 runtime: wasm
//                 ref: /path/to/plugin.wasm
//
// Testing:
//   dimctl plugin conformance --wasm /path/to/plugin.wasm

use serde_json::{json, Value};

// Simple allocator for WASM linear memory
// In production, use a proper allocator like wee_alloc
static mut HEAP: [u8; 65536] = [0; 65536];
static mut HEAP_OFFSET: u32 = 0;

#[no_mangle]
pub extern "C" fn malloc(size: u32) -> u32 {
    unsafe {
        let offset = HEAP_OFFSET;
        HEAP_OFFSET += size;
        if HEAP_OFFSET as usize > HEAP.len() {
            panic!("Out of memory");
        }
        offset
    }
}

#[no_mangle]
pub extern "C" fn free(_ptr: u32, _size: u32) {
    // No-op; in production, implement proper memory management
}

// Helper: Read a string from WASM linear memory
unsafe fn read_string(ptr: u32, len: u32) -> String {
    let bytes = std::slice::from_raw_parts(ptr as *const u8, len as usize);
    String::from_utf8_lossy(bytes).to_string()
}

// Helper: Write a string to WASM linear memory
unsafe fn write_string(s: &str) -> (u32, u32) {
    let bytes = s.as_bytes();
    let ptr = malloc(bytes.len() as u32);
    std::ptr::copy_nonoverlapping(bytes.as_ptr(), (ptr as *mut u8), bytes.len());
    (ptr, bytes.len() as u32)
}

/// Validate a credit card number using Luhn algorithm
///
/// Input JSON: {"card_number": "4532015112830366"}
/// Output JSON: {"valid": true, "card_type": "visa", "last_four": "0366"}
#[no_mangle]
pub extern "C" fn validate_credit_card(args_ptr: u32, args_len: u32) -> u64 {
    unsafe {
        // Parse input JSON
        let args_str = read_string(args_ptr, args_len);
        let args: Value = match serde_json::from_str(&args_str) {
            Ok(v) => v,
            Err(e) => {
                let error = json!({"error": format!("Invalid JSON: {}", e)});
                let (ptr, len) = write_string(&error.to_string());
                return ((ptr as u64) << 32) | (len as u64);
            }
        };

        // Extract card number
        let card_number = match args.get("card_number").and_then(|v| v.as_str()) {
            Some(cn) => cn,
            None => {
                let error = json!({"error": "card_number field required"});
                let (ptr, len) = write_string(&error.to_string());
                return ((ptr as u64) << 32) | (len as u64);
            }
        };

        // Validate using Luhn algorithm
        let valid = luhn_check(card_number);

        // Determine card type
        let card_type = if card_number.starts_with("4") {
            "visa"
        } else if card_number.starts_with("5") {
            "mastercard"
        } else if card_number.starts_with("3") {
            "amex"
        } else {
            "unknown"
        };

        // Get last 4 digits
        let last_four = if card_number.len() >= 4 {
            &card_number[card_number.len() - 4..]
        } else {
            "****"
        };

        let result = json!({
            "result": {
                "valid": valid,
                "card_type": card_type,
                "last_four": last_four
            }
        });

        let (ptr, len) = write_string(&result.to_string());
        ((ptr as u64) << 32) | (len as u64)
    }
}

/// Validate an email address using simple regex
///
/// Input JSON: {"email": "alice@example.com"}
/// Output JSON: {"result": true, "reason": "Valid email format"}
#[no_mangle]
pub extern "C" fn validate_email(args_ptr: u32, args_len: u32) -> u64 {
    unsafe {
        // Parse input JSON
        let args_str = read_string(args_ptr, args_len);
        let args: Value = match serde_json::from_str(&args_str) {
            Ok(v) => v,
            Err(e) => {
                let error = json!({"error": format!("Invalid JSON: {}", e)});
                let (ptr, len) = write_string(&error.to_string());
                return ((ptr as u64) << 32) | (len as u64);
            }
        };

        // Extract email
        let email = match args.get("email").and_then(|v| v.as_str()) {
            Some(e) => e,
            None => {
                let error = json!({"error": "email field required"});
                let (ptr, len) = write_string(&error.to_string());
                return ((ptr as u64) << 32) | (len as u64);
            }
        };

        // Simple email validation
        let valid = email.contains("@") && email.len() > 5;

        let result = json!({
            "result": valid,
            "reason": if valid { "Valid email format" } else { "Invalid email format" }
        });

        let (ptr, len) = write_string(&result.to_string());
        ((ptr as u64) << 32) | (len as u64)
    }
}

/// Transform a phone number to standard format
///
/// Input JSON: {"phone": "5551234567"}
/// Output JSON: {"result": "(555) 123-4567"}
#[no_mangle]
pub extern "C" fn transform_phone(args_ptr: u32, args_len: u32) -> u64 {
    unsafe {
        // Parse input JSON
        let args_str = read_string(args_ptr, args_len);
        let args: Value = match serde_json::from_str(&args_str) {
            Ok(v) => v,
            Err(e) => {
                let error = json!({"error": format!("Invalid JSON: {}", e)});
                let (ptr, len) = write_string(&error.to_string());
                return ((ptr as u64) << 32) | (len as u64);
            }
        };

        // Extract phone number
        let phone = match args.get("phone").and_then(|v| v.as_str()) {
            Some(p) => p,
            None => {
                let error = json!({"error": "phone field required"});
                let (ptr, len) = write_string(&error.to_string());
                return ((ptr as u64) << 32) | (len as u64);
            }
        };

        // Extract digits only
        let digits: String = phone.chars().filter(|c| c.is_ascii_digit()).collect();

        // Format as (XXX) XXX-XXXX if 10 digits
        let formatted = if digits.len() == 10 {
            format!(
                "({}) {}-{}",
                &digits[0..3],
                &digits[3..6],
                &digits[6..10]
            )
        } else {
            format!("digits only: {}", digits)
        };

        let result = json!({"result": formatted});

        let (ptr, len) = write_string(&result.to_string());
        ((ptr as u64) << 32) | (len as u64)
    }
}

/// Return the SDK version this plugin targets
#[no_mangle]
pub extern "C" fn get_sdk_version() -> u32 {
    1 // SDK version 1.0
}

// Luhn algorithm for credit card validation
fn luhn_check(card_number: &str) -> bool {
    let digits: Vec<u32> = card_number
        .chars()
        .filter_map(|c| c.to_digit(10))
        .collect();

    if digits.is_empty() {
        return false;
    }

    let mut sum = 0;
    let mut is_second = false;

    for &digit in digits.iter().rev() {
        let mut d = digit;
        if is_second {
            d *= 2;
            if d > 9 {
                d -= 9;
            }
        }
        sum += d;
        is_second = !is_second;
    }

    sum % 10 == 0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_luhn_valid() {
        assert!(luhn_check("4532015112830366")); // Valid Visa
    }

    #[test]
    fn test_luhn_invalid() {
        assert!(!luhn_check("1234567890123456")); // Invalid
    }
}
