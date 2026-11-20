# Agent clients

Official MCP and A2A Python SDK clients, run against `dimctl agent serve`.

    dimctl agent serve --addr 127.0.0.1:9191     # from the repository root
    pip install -r requirements.txt
    python mcp_and_a2a_check.py http://127.0.0.1:9191
    python mcp_validate_route.py                 # edit ROUTE to an absolute path first

`mcp_and_a2a_check.py` connects over MCP (initialize, list tools, call `get_capabilities`) and over A2A (fetch the agent card, send a `SendMessage` for `get_capabilities`).
