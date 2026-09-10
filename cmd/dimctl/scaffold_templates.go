package main

// Passthrough template - direct source to sink
var passthroughTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: http

routes:
  passthrough:
    from: input
    steps: []

sinks:
  output:
    type: file
    path: ./output/messages.jsonl
`

// Transform template - source with transform step to sink
var transformTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: http

routes:
  with-transform:
    from: input
    steps:
      - translate:
          expr: '{"id": body.id, "amount": body.amount}'

sinks:
  output:
    type: file
    path: ./output/messages.jsonl
`

// Contract-enforced template - source to contract-enforced sink
var contractTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: http

contracts:
  data-contract:
    schema: |
      {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "amount": { "type": "number" }
        },
        "required": ["id", "amount"]
      }

routes:
  with-contract:
    from: input
    steps: []

sinks:
  output:
    type: file
    path: ./output/messages.jsonl
    enforce: true
    contract: data-contract
`

// Domain metadata template
var domainMetadataTemplate = `# Domain metadata for {{.DomainName}}
domain:
  name: {{.DomainName}}
  owner: user@example.com
  slack_channel: "#{{.DomainName}}-team"
  description: {{.DomainName}} data processing and integration
  owner_team: {{.DomainName}}-platform
`

// Error path template
var errorPathTemplate = `version: 1

# Error path template — uncomment and customize as needed
# This configuration handles messages that fail validation or processing
#
# error_path:
#   steps:
#     - log:
#         level: error
#         message: 'Failed to process message'
#   sinks:
#     dlq:
#       type: file
#       path: ./dlq/failed-messages.jsonl
`
