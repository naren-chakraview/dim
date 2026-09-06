// Reference implementation: Native Go function plugin
//
// This is a working example of a dim function plugin using only pkg/sdk.
// It demonstrates:
// - Implementing the sdk.Function interface
// - Marshaling/unmarshaling JSON arguments
// - Error handling
//
// Building:
//   go build -o validate-email ./examples/plugin/native-go
//
// Using in a route:
//   routes:
//     order-processing:
//       steps:
//         - type: translate
//           spec:
//             expression: $validate_email(message.email)
//             functions:
//               - name: validate_email
//                 type: plugin
//                 runtime: native
//                 ref: /path/to/validate-email
//
// Testing:
//   dimctl plugin conformance --native /path/to/validate-email
//
// Note: This reference implementation shows the sdk.Function interface.
// Production deployment uses hashicorp/go-plugin for the RPC layer,
// which this example omits for clarity (see M3.3.1_PLUGIN_SDK_SPEC.md).

package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/naren-chakraview/dim/pkg/sdk"
)

// ValidateEmailPlugin implements the sdk.Function interface
// It provides email validation for routing decisions
type ValidateEmailPlugin struct{}

// Call validates an email address
// Input: email address (string)
// Output: {"valid": bool, "reason": string}
func (p *ValidateEmailPlugin) Call(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("email address required")
	}

	// Extract email from first argument
	var email string
	switch v := args[0].(type) {
	case string:
		email = v
	case map[string]interface{}:
		// Handle {"email": "..."}
		if e, ok := v["email"].(string); ok {
			email = e
		} else {
			return nil, fmt.Errorf("email field not found in map")
		}
	default:
		return nil, fmt.Errorf("expected string or map, got %T", args[0])
	}

	email = strings.TrimSpace(email)

	// Validate email format (simple regex)
	// Production: use a proper email validation library
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	valid := emailRegex.MatchString(email)

	result := map[string]interface{}{
		"valid": valid,
		"email": email,
	}

	if !valid {
		result["reason"] = "Invalid email format"
	} else {
		result["reason"] = "Valid email format"
	}

	return result, nil
}

// FakeDataPlugin demonstrates a plugin with multiple functions
// In a real scenario, the host would route to different functions by name
type FakeDataPlugin struct {
	functions map[string]sdk.Function
}

// NewFakeDataPlugin creates a plugin with multiple functions
func NewFakeDataPlugin() *FakeDataPlugin {
	return &FakeDataPlugin{
		functions: map[string]sdk.Function{
			"validate_email":   &ValidateEmailPlugin{},
			"enrich_customer":  &EnrichCustomerPlugin{},
			"transform_phone":  &TransformPhonePlugin{},
		},
	}
}

// Call routes to the appropriate function
// The function name is passed as metadata, not as args
// For now, we'll just validate emails (the plugin RPC layer handles routing)
func (p *FakeDataPlugin) Call(args ...interface{}) (interface{}, error) {
	// In the actual plugin protocol, the RPC layer determines which function to call
	// For this reference implementation, we default to email validation
	return p.functions["validate_email"].Call(args...)
}

// EnrichCustomerPlugin demonstrates fetching external data (e.g., from an API)
type EnrichCustomerPlugin struct{}

func (p *EnrichCustomerPlugin) Call(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("customer_id required")
	}

	customerID := fmt.Sprintf("%v", args[0])

	// In production, this would call an external API
	// For now, return mock data
	result := map[string]interface{}{
		"customer_id": customerID,
		"name":        "Alice Smith",
		"tier":        "premium",
		"balance":     5000.00,
		"region":      "US-EAST-1",
	}

	return result, nil
}

// TransformPhonePlugin demonstrates data transformation
type TransformPhonePlugin struct{}

func (p *TransformPhonePlugin) Call(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("phone number required")
	}

	phone := fmt.Sprintf("%v", args[0])

	// Remove non-digits
	var digitsOnly strings.Builder
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digitsOnly.WriteRune(ch)
		}
	}

	digits := digitsOnly.String()

	// Format as (XXX) XXX-XXXX if 10 digits (US format)
	if len(digits) == 10 {
		formatted := fmt.Sprintf("(%s) %s-%s", digits[:3], digits[3:6], digits[6:])
		return map[string]interface{}{
			"original":  phone,
			"formatted": formatted,
			"digits":    digits,
		}, nil
	}

	return map[string]interface{}{
		"original": phone,
		"digits":   digits,
		"warning":  "Not a standard 10-digit US number",
	}, nil
}

func main() {
	// This is a reference implementation demonstrating the sdk.Function interface.
	//
	// In production, plugins:
	// 1. Check the magic cookie (DIM_PLUGIN env var)
	// 2. Serve via hashicorp/go-plugin (RPC protocol)
	// 3. Implement the sdk.Function interface for each function
	//
	// See design/M3.3.1_PLUGIN_SDK_SPEC.md §2.2 for the full native plugin protocol.

	// Example: Test the plugins locally
	fmt.Println("dim Native Go Plugin Reference Implementation")
	fmt.Println(strings.Repeat("=", 50))

	// Test email validation
	emailPlugin := &ValidateEmailPlugin{}
	result, _ := emailPlugin.Call("alice@example.com")
	fmt.Printf("validate_email(\"alice@example.com\") = %v\n", result)

	// Test customer enrichment
	customerPlugin := &EnrichCustomerPlugin{}
	result, _ = customerPlugin.Call("customer-123")
	fmt.Printf("enrich_customer(\"customer-123\") = %v\n", result)

	// Test phone transformation
	phonePlugin := &TransformPhonePlugin{}
	result, _ = phonePlugin.Call("5551234567")
	fmt.Printf("transform_phone(\"5551234567\") = %v\n", result)

	fmt.Println("\nFor production use, implement the RPC handshake and serve via go-plugin.")
	fmt.Println("See examples/plugin/native-go/main.go and design/M3.3.1_PLUGIN_SDK_SPEC.md")
}
