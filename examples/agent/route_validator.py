#!/usr/bin/env python3
"""
Reference Agent: Route Validator

Demonstrates using the M4.6 Agent-Facing Interface (MCP) to validate
and test routes. This agent uses ONLY the published interface — no
internal package imports.

Usage:
    python route_validator.py --config <route.yaml> --fixtures <fixtures.yaml>
"""

import argparse
import json
import sys
from typing import Any, Dict, Optional
from dataclasses import dataclass


# Interface types (matching internal/agent/types.go)
@dataclass
class OperationError:
    code: str
    message: str
    details: Optional[Dict[str, Any]] = None


@dataclass
class ResponseEnvelope:
    interface_version: str
    timestamp: str
    result: Optional[Dict[str, Any]] = None
    error: Optional[OperationError] = None
    deprecation_notice: Optional[str] = None


class AgentMCPClient:
    """Client for calling the Agent Interface operations"""

    def __init__(self, endpoint: str = "http://localhost:9090"):
        """
        Initialize MCP client.

        Args:
            endpoint: Base URL of the agent interface server
        """
        self.endpoint = endpoint
        self.interface_version = "1.0.0"

    def call_operation(
        self,
        operation: str,
        request: Dict[str, Any]
    ) -> ResponseEnvelope:
        """
        Call an operation via the MCP interface.

        In a real implementation, this would use:
        - stdio-based MCP transport
        - HTTP/JSON-RPC transport
        - Actual MCP SDK

        For this reference, we simulate the interface.
        """
        # In production, this would make an actual RPC call.
        # For reference purposes, we simulate the call.
        print(f"[Agent] Calling {operation}")
        print(f"[Agent] Request: {json.dumps(request, indent=2)}")

        # Simulate server response (in real implementation, would come from server)
        response = {
            "interface_version": self.interface_version,
            "timestamp": "2026-09-11T08:10:00Z",
            "result": None,
            "error": None,
        }

        # Dispatch to operation handler
        if operation == "validate_route":
            response["result"] = self._handle_validate_route(request)
        elif operation == "test_route":
            response["result"] = self._handle_test_route(request)
        elif operation == "scaffold_domain":
            response["result"] = self._handle_scaffold_domain(request)
        else:
            response["error"] = {
                "code": "NOT_IMPLEMENTED",
                "message": f"Operation '{operation}' not found",
            }

        return self._parse_response(response)

    def _handle_validate_route(self, req: Dict[str, Any]) -> Dict[str, Any]:
        """Simulate validate_route operation"""
        if "route_config_path" not in req or not req["route_config_path"]:
            return None  # Would have error in envelope

        return {
            "valid": True,
            "errors": [],
            "warnings": [],
            "route_version": "sha256:abc123def456",
        }

    def _handle_test_route(self, req: Dict[str, Any]) -> Dict[str, Any]:
        """Simulate test_route operation"""
        if "route_config_path" not in req or "fixtures_path" not in req:
            return None

        return {
            "passed": True,
            "test_results": [
                {
                    "name": "test_order_ingestion",
                    "passed": True,
                    "duration_ms": 245,
                }
            ],
            "summary": {
                "total": 1,
                "passed": 1,
                "failed": 0,
                "skipped": 0,
                "duration_ms": 245,
            },
        }

    def _handle_scaffold_domain(self, req: Dict[str, Any]) -> Dict[str, Any]:
        """Simulate scaffold_domain operation"""
        if "domain" not in req or not req["domain"]:
            return None

        domain = req["domain"]
        return {
            "success": True,
            "domain_path": f"domains/{domain}",
            "files_created": [
                f"domains/{domain}/DOMAIN.yaml",
                f"domains/{domain}/{domain}-route.yaml",
            ],
            "next_steps": [
                f"Review the generated files in domains/{domain}",
                "Run `dimctl validate` to verify the configuration",
                "Commit to git and create a pull request",
            ],
        }

    def _parse_response(self, response: Dict[str, Any]) -> ResponseEnvelope:
        """Parse MCP response into ResponseEnvelope"""
        error = None
        if response.get("error"):
            err_data = response["error"]
            error = OperationError(
                code=err_data.get("code", "UNKNOWN"),
                message=err_data.get("message", ""),
                details=err_data.get("details"),
            )

        return ResponseEnvelope(
            interface_version=response.get("interface_version", "unknown"),
            timestamp=response.get("timestamp", ""),
            result=response.get("result"),
            error=error,
            deprecation_notice=response.get("deprecation_notice"),
        )


class RouteValidator:
    """Agent for validating and testing routes"""

    def __init__(self, mcp_client: AgentMCPClient):
        self.client = mcp_client

    def validate_route(self, config_path: str, strict: bool = False) -> bool:
        """
        Validate a route configuration.

        Returns True if valid, False otherwise.
        """
        print(f"\n📋 Validating route: {config_path}")

        response = self.client.call_operation(
            "validate_route",
            {
                "route_config_path": config_path,
                "strict_mode": strict,
            }
        )

        if response.error:
            print(f"❌ Validation failed: {response.error.message}")
            if response.error.details:
                print(f"   Details: {json.dumps(response.error.details, indent=2)}")
            return False

        result = response.result
        if not result.get("valid"):
            print(f"❌ Validation failed with {len(result.get('errors', []))} error(s)")
            for err in result.get("errors", []):
                print(f"   - {err.get('message')} at {err.get('path')}")
            return False

        print("✅ Validation passed")
        if result.get("warnings"):
            print(f"   ⚠️  {len(result['warnings'])} warning(s)")
            for warning in result["warnings"]:
                print(f"      - {warning}")

        if result.get("route_version"):
            print(f"   Route version: {result['route_version']}")

        return True

    def test_route(self, config_path: str, fixtures_path: str) -> bool:
        """
        Test a route with fixtures.

        Returns True if all tests pass, False otherwise.
        """
        print(f"\n🧪 Testing route: {config_path} with {fixtures_path}")

        response = self.client.call_operation(
            "test_route",
            {
                "route_config_path": config_path,
                "fixtures_path": fixtures_path,
            }
        )

        if response.error:
            print(f"❌ Tests failed: {response.error.message}")
            return False

        result = response.result
        summary = result.get("summary", {})

        if not result.get("passed"):
            print(f"❌ Tests failed: {summary.get('failed', 0)} failure(s)")
            for test in result.get("test_results", []):
                if not test.get("passed"):
                    print(f"   - {test.get('name')}: {test.get('error', 'Unknown error')}")
            return False

        print("✅ All tests passed")
        print(
            f"   {summary.get('passed')}/{summary.get('total')} tests passed "
            f"({summary.get('duration_ms')}ms)"
        )

        return True

    def scaffold_domain(
        self,
        domain: str,
        source_type: str = "http",
        sink_type: str = "file",
    ) -> bool:
        """
        Scaffold a new domain.

        Returns True if successful.
        """
        print(f"\n🏗️  Scaffolding domain: {domain}")

        response = self.client.call_operation(
            "scaffold_domain",
            {
                "domain": domain,
                "source_type": source_type,
                "sink_type": sink_type,
            }
        )

        if response.error:
            print(f"❌ Scaffolding failed: {response.error.message}")
            return False

        result = response.result
        if not result.get("success"):
            print("❌ Scaffolding failed")
            return False

        print("✅ Scaffolding complete")
        print(f"   Domain path: {result.get('domain_path')}")
        print(f"   Files created:")
        for file in result.get("files_created", []):
            print(f"      - {file}")

        print(f"   Next steps:")
        for step in result.get("next_steps", []):
            print(f"      - {step}")

        return True


def main():
    parser = argparse.ArgumentParser(
        description="Route Validator Agent — uses published MCP interface to validate routes"
    )
    parser.add_argument(
        "--config",
        help="Path to route configuration file",
        default="domains/payments/order-payment.yaml",
    )
    parser.add_argument(
        "--fixtures",
        help="Path to test fixtures",
        default=None,
    )
    parser.add_argument(
        "--scaffold",
        help="Scaffold a new domain instead of validating",
        action="store_true",
    )
    parser.add_argument(
        "--strict",
        help="Strict validation mode",
        action="store_true",
    )

    args = parser.parse_args()

    # Create MCP client and agent
    mcp_client = AgentMCPClient()
    agent = RouteValidator(mcp_client)

    print("=" * 60)
    print("Route Validator Agent (M4.6 Reference Implementation)")
    print("=" * 60)
    print(f"Interface Version: {mcp_client.interface_version}")
    print()

    success = True

    # Scaffold mode
    if args.scaffold:
        domain_name = args.config.split("/")[1] if "/" in args.config else "new-domain"
        success = agent.scaffold_domain(domain_name)
    else:
        # Validate mode
        if not agent.validate_route(args.config, args.strict):
            success = False

        # Test mode (if fixtures provided)
        if args.fixtures and success:
            if not agent.test_route(args.config, args.fixtures):
                success = False

    print()
    print("=" * 60)
    if success:
        print("✅ Operation completed successfully")
        return 0
    else:
        print("❌ Operation failed")
        return 1


if __name__ == "__main__":
    sys.exit(main())
