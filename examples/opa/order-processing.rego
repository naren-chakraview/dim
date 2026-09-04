package dim.authorize

# Order Processing Authorization Policy
# Defines who can perform which actions on orders
# Used by M1.2.3 (OPA reference implementation)

# Default: deny all access
default allow = false

# Sellers can process their own orders
allow = true {
	input.principal.roles[_] = "seller"
	input.action = "process_order"
}

# Admins can do anything
allow = true {
	input.principal.roles[_] = "admin"
}

# Viewers can read orders but not modify
allow = true {
	input.principal.roles[_] = "viewer"
	input.action = "read_order"
}

# Compliance officers can audit orders
allow = true {
	input.principal.roles[_] = "compliance"
	input.action = "audit_order"
}

# Reason for denial (used in DecisionResponse)
reason = "insufficient roles for this action" {
	not allow
}

reason = "access allowed" {
	allow
}

# Obligations: PII data requires redaction for non-admins
obligations[obligation] {
	allow
	input.principal.roles[_] != "admin"
	input.resource.classification = "pii"
	obligation := {
		"type": "redact_fields",
		"parameters": {
			"fields": ["ssn", "credit_card", "phone", "email"]
		}
	}
}

# Obligations: Rate limiting for non-premium sellers
obligations[obligation] {
	allow
	input.principal.roles[_] = "seller"
	input.principal.attributes.tier != "premium"
	obligation := {
		"type": "rate_limit",
		"parameters": {
			"requests_per_minute": 100
		}
	}
}

# Obligations: Audit logging for admin actions
obligations[obligation] {
	allow
	input.principal.roles[_] = "admin"
	obligation := {
		"type": "audit_log",
		"parameters": {
			"level": "WARNING",
			"tags": ["admin_action", "potential_sensitive_operation"]
		}
	}
}

# Obligations: Data classification tagging for compliance
obligations[obligation] {
	allow
	input.resource.classification = "pci"
	obligation := {
		"type": "data_classification_tag",
		"parameters": {
			"level": "pci",
			"handling": "pci-dss-compliant"
		}
	}
}
