#!/usr/bin/env python3
"""
Reference Agent: Route Validator

Demonstrates using the M4.6 Agent-Facing Interface to validate and test
routes via HTTP. This agent uses ONLY the published interface — no
internal package imports. Requires 'dimctl agent serve' to be running.

Usage:
    # Start the agent server in another terminal:
    #   dimctl agent serve --addr localhost:9090

    # Then run this agent:
    python route_validator.py --config <route.yaml> --fixtures <fixtures.yaml>
    python route_validator.py --scaffold --config <domain-name>
"""

import argparse
import json
import sys
import requests
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

        Makes a real HTTP request to the dimctl agent serve server.
        """
        try:
            # Make HTTP request to agent server
            url = f"{self.endpoint}/call"
            payload = {
                "operation": operation,
                "request": request,
            }

            response = requests.post(url, json=payload, timeout=30)
            response.raise_for_status()

            # Parse response
            return self._parse_response(response.json())

        except requests.exceptions.ConnectionError:
            return ResponseEnvelope(
                interface_version=self.interface_version,
                timestamp="",
                error=OperationError(
                    code="CONNECTION_FAILED",
                    message=f"Could not connect to agent server at {self.endpoint}. "
                           f"Make sure 'dimctl agent serve' is running.",
                )
            )
        except requests.exceptions.Timeout:
            return ResponseEnvelope(
                interface_version=self.interface_version,
                timestamp="",
                error=OperationError(
                    code="TIMEOUT",
                    message=f"Request to agent server timed out",
                )
            )
        except Exception as e:
            return ResponseEnvelope(
                interface_version=self.interface_version,
                timestamp="",
                error=OperationError(
                    code="REQUEST_FAILED",
                    message=f"Failed to call operation: {str(e)}",
                )
            )

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
    parser.add_argument(
        "--server",
        help="Agent server address (default: http://localhost:9090)",
        default="http://localhost:9090",
    )

    args = parser.parse_args()

    # Create MCP client and agent
    mcp_client = AgentMCPClient(endpoint=args.server)
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
