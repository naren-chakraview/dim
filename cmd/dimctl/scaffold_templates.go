package main

// Passthrough template - direct source to sink
var passthroughTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: {{.SourceType}}

routes:
  passthrough:
    from: input
    auth: none
    error_path:
      target: error
    steps:
      - fragment: governance-baseline
      - filter:
          expr: "true"

sinks:
  output:
    type: {{.SinkType}}
    path: ./output/messages.jsonl
  error:
    type: file
    path: ./output/error.jsonl
`

// Transform template - source with transform step to sink
var transformTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: {{.SourceType}}

routes:
  with-transform:
    from: input
    auth: none
    error_path:
      target: error
    steps:
      - fragment: governance-baseline
      - translate:
          expr: '{"id": body.id, "amount": body.amount}'

sinks:
  output:
    type: {{.SinkType}}
    path: ./output/messages.jsonl
  error:
    type: file
    path: ./output/error.jsonl
`

// Contract-enforced template - source to contract-enforced sink
var contractTemplate = `version: 1
imports:
  - ../governance/fragments.yaml

sources:
  input:
    type: {{.SourceType}}

routes:
  with-contract:
    from: input
    auth: none
    error_path:
      target: error
    contracts:
      output-contract:
        enforce: true
        schema:
          type: object
          properties:
            id:
              type: string
            data:
              type: object
          required: [id, data]
    steps:
      - fragment: governance-baseline
      - filter:
          expr: "true"
      - contract:
          id: output-contract
          strict: false

sinks:
  output:
    type: {{.SinkType}}
    path: ./output/messages.jsonl
  error:
    type: file
    path: ./output/error.jsonl
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
imports:
  - ../governance/fragments.yaml

# Error path template — uncomment and customize as needed
# This configuration handles messages that fail validation or processing
#
# routes:
#   error-handler:
#     steps:
#       - fragment: governance-baseline
#       - log:
#           level: error
#           message: 'Failed to process message'
#
# sinks:
#   dlq:
#     type: file
#     path: ./dlq/failed-messages.jsonl
`
