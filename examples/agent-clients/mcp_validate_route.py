import asyncio, json, sys
from mcp import ClientSession
from mcp.client.streamable_http import streamable_http_client
ROUTE = "/home/gundu/portfolio/dim/domains/payments/order-payment.yaml  # edit to your absolute path"
async def main():
    async with streamable_http_client("http://127.0.0.1:9191/mcp") as s:
        async with ClientSession(s[0], s[1]) as session:
            await session.initialize()
            r = await session.call_tool("validate_route", {"route_config_path": ROUTE})
            env = json.loads(r.content[0].text)
            print("validate_route isError =", r.is_error, "| result keys =", sorted((env.get("result") or {}).keys())[:6])
            print("validate_route error =", env.get("error"))
            r2 = await session.call_tool("validate_route", {"route_config_path": "/nonexistent.yaml"})
            print("missing file isError =", r2.is_error, "|", json.loads(r2.content[0].text).get("error"))
asyncio.run(main())
