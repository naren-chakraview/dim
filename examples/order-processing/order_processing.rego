# OPA Policy for Order Processing Authorization (PBAC, M1.2, R19)
# Demonstrates attribute-based access control using order attributes
# and user attributes extracted from JWT.

package order_processing

# Default deny — any access not explicitly allowed is denied
default allow = false

# Allow access if user is a seller and order belongs to their account
# Attributes:
#   - subject_id: User ID from JWT
#   - role: User's role (seller, admin, customer_service)
#   - department: User's department (sales, support, ops)
#   - order_customer: Customer ID from the order being processed
allow {
	input.role == "seller"
	input.order_customer == input.subject_id
}

# Allow access if user is admin
allow {
	input.role == "admin"
}

# Allow access if user is in customer_service department reviewing own customer's order
allow {
	input.role == "customer_service"
	input.department == "support"
	startswith(input.order_customer, "CUST_")
}

# Obligations: log all access attempts (audit trail)
obligations[obligation] {
	obligation := {
		"type": "log_audit",
		"action": "order_access_attempt",
		"user": input.subject_id,
		"order_customer": input.order_customer,
		"result": allow ? "allowed" : "denied"
	}
}

# Obligations: redact sensitive fields if user is not admin
# Non-admins can see order total and customer name, but not payment method
obligations[obligation] {
	not allow
	obligation := {
		"type": "redact_fields",
		"fields": ["payment_method", "shipping_address"]
	}
}
