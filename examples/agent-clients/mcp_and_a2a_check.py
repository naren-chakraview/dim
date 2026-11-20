import asyncio, json, sys
URL = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:9191"

async def mcp_check():
    from mcp import ClientSession
    from mcp.client.streamable_http import streamable_http_client
    async with streamable_http_client(URL + "/mcp") as streams:
        read, write = streams[0], streams[1]
        async with ClientSession(read, write) as session:
            init = await session.initialize()
            print("MCP initialize: server", init.server_info.name, init.protocol_version)
            tools = await session.list_tools()
            names = sorted(t.name for t in tools.tools)
            print("MCP tools:", len(names), names)
            schema = next(t for t in tools.tools if t.name == "validate_route").input_schema
            print("MCP validate_route required:", schema.get("required"))
            result = await session.call_tool("get_capabilities", {"filter_type": "adapter"})
            env = json.loads(result.content[0].text)
            print("MCP get_capabilities: adapters =", len(env["result"]["capabilities"]), "isError =", result.is_error)

async def a2a_check():
    import httpx
    from a2a.client import ClientFactory, ClientConfig
    from a2a.types import Message, Part, Role
    from google.protobuf import json_format
    from google.protobuf.struct_pb2 import Value
    async with httpx.AsyncClient() as http:
        from a2a.client.card_resolver import A2ACardResolver
        card = await A2ACardResolver(http, URL).get_agent_card()
        print("A2A card:", card.name, "skills =", len(card.skills), "url =", card.supported_interfaces[0].url)
        client = ClientFactory(ClientConfig(httpx_client=http, streaming=False, supported_protocol_bindings=["JSONRPC"])).create(card)
        msg = Message(message_id="m-1", role=Role.ROLE_USER,
                      parts=[Part(data=json_format.ParseDict({"operation": "get_capabilities", "request": {"filter_type": "adapter"}}, Value()))])
        from a2a.types import SendMessageRequest, SendMessageConfiguration
        request = SendMessageRequest(message=msg, configuration=SendMessageConfiguration())
        async for event in client.send_message(request):
            task = event[0] if isinstance(event, tuple) else event
            if hasattr(task, "task") and task.HasField("task"):
                task = task.task
            if hasattr(task, "status"):
                from a2a.types import TaskState
                print("A2A task state:", TaskState.Name(task.status.state), "artifacts =", len(task.artifacts))
                data = json_format.MessageToDict(task.artifacts[0].parts[0].data)
                print("A2A result adapters =", len(data["result"]["capabilities"]))
                break
            print("A2A event:", type(event).__name__)

async def main():
    await mcp_check()
    await a2a_check()

asyncio.run(main())
