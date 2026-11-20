# Graph Report - dim  (2026-11-20)

## Corpus Check
- 263 files · ~419,945 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2797 nodes · 7614 edges · 94 communities detected
- Extraction: 51% EXTRACTED · 49% INFERRED · 0% AMBIGUOUS · INFERRED: 3703 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 71|Community 71]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 74|Community 74]]
- [[_COMMUNITY_Community 75|Community 75]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 77|Community 77]]
- [[_COMMUNITY_Community 78|Community 78]]
- [[_COMMUNITY_Community 79|Community 79]]
- [[_COMMUNITY_Community 80|Community 80]]
- [[_COMMUNITY_Community 81|Community 81]]
- [[_COMMUNITY_Community 82|Community 82]]
- [[_COMMUNITY_Community 83|Community 83]]
- [[_COMMUNITY_Community 84|Community 84]]
- [[_COMMUNITY_Community 85|Community 85]]
- [[_COMMUNITY_Community 86|Community 86]]
- [[_COMMUNITY_Community 87|Community 87]]
- [[_COMMUNITY_Community 88|Community 88]]
- [[_COMMUNITY_Community 89|Community 89]]
- [[_COMMUNITY_Community 90|Community 90]]
- [[_COMMUNITY_Community 91|Community 91]]
- [[_COMMUNITY_Community 92|Community 92]]
- [[_COMMUNITY_Community 93|Community 93]]
- [[_COMMUNITY_Community 94|Community 94]]
- [[_COMMUNITY_Community 95|Community 95]]
- [[_COMMUNITY_Community 96|Community 96]]
- [[_COMMUNITY_Community 97|Community 97]]
- [[_COMMUNITY_Community 98|Community 98]]
- [[_COMMUNITY_Community 99|Community 99]]
- [[_COMMUNITY_Community 100|Community 100]]
- [[_COMMUNITY_Community 101|Community 101]]
- [[_COMMUNITY_Community 102|Community 102]]
- [[_COMMUNITY_Community 103|Community 103]]
- [[_COMMUNITY_Community 104|Community 104]]
- [[_COMMUNITY_Community 105|Community 105]]
- [[_COMMUNITY_Community 106|Community 106]]
- [[_COMMUNITY_Community 107|Community 107]]
- [[_COMMUNITY_Community 108|Community 108]]
- [[_COMMUNITY_Community 109|Community 109]]
- [[_COMMUNITY_Community 110|Community 110]]
- [[_COMMUNITY_Community 111|Community 111]]
- [[_COMMUNITY_Community 112|Community 112]]
- [[_COMMUNITY_Community 113|Community 113]]
- [[_COMMUNITY_Community 114|Community 114]]
- [[_COMMUNITY_Community 115|Community 115]]
- [[_COMMUNITY_Community 116|Community 116]]
- [[_COMMUNITY_Community 117|Community 117]]
- [[_COMMUNITY_Community 118|Community 118]]
- [[_COMMUNITY_Community 119|Community 119]]
- [[_COMMUNITY_Community 120|Community 120]]
- [[_COMMUNITY_Community 121|Community 121]]
- [[_COMMUNITY_Community 122|Community 122]]
- [[_COMMUNITY_Community 123|Community 123]]
- [[_COMMUNITY_Community 124|Community 124]]
- [[_COMMUNITY_Community 125|Community 125]]

## God Nodes (most connected - your core abstractions)
1. `NewMessage()` - 246 edges
2. `NewChannel()` - 140 edges
3. `contains()` - 55 edges
4. `b()` - 47 edges
5. `NewExecutor()` - 46 edges
6. `rv()` - 46 edges
7. `NewTracingProvider()` - 41 edges
8. `LoadRouteConfig()` - 39 edges
9. `t()` - 39 edges
10. `NewContractStore()` - 37 edges

## Surprising Connections (you probably didn't know these)
- `BuildStepsFromSpec()` --calls--> `buildExecutorForRoute()`  [INFERRED]
  internal/steps/factory.go → cmd/dimctl/main.go
- `NewIdempotentStep()` --calls--> `BenchmarkStepExecution()`  [INFERRED]
  internal/steps/idempotent.go → examples/bench/throughput_test.go
- `NewTranslateStep()` --calls--> `BenchmarkMessageThroughput()`  [INFERRED]
  internal/steps/translate.go → examples/bench/throughput_test.go
- `NewTranslateStep()` --calls--> `BenchmarkStepExecution()`  [INFERRED]
  internal/steps/translate.go → examples/bench/throughput_test.go
- `NewFilterStep()` --calls--> `BenchmarkMessageThroughput()`  [INFERRED]
  internal/steps/filter.go → examples/bench/throughput_test.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (469): _0(), _a(), A0(), a1(), Aa(), ac(), Ad(), Ae() (+461 more)

### Community 1 - "Community 1"
Cohesion: 0.02
Nodes (227): AMQPSource, BenchmarkAMQPMessageConversion(), TestAMQPSinkMetrics(), TestAMQPSourceBasic(), TestAMQPSourceClose(), NewAMQPSink(), NewAMQPSinkWithConfig(), SinkConfig (+219 more)

### Community 2 - "Community 2"
Cohesion: 0.03
Nodes (165): TestAggregatorIntegration(), TestAggregatorMultipleOrdersIntegration(), TestAggregatorTimeoutCompletion(), TestAggregatorWithComplexMessages(), NewAggregateStep(), TestAggregateCountCompletion(), TestAggregateDrain(), TestAggregateMultipleCorrelationKeys() (+157 more)

### Community 3 - "Community 3"
Cohesion: 0.02
Nodes (78): AMQPSink, isVersionListResponse(), loadRoute(), TestClaimCheckCleanup(), TestClaimCheckMultipleAttachments(), TestClaimCheckOrderWithAttachmentWorkflow(), TestClaimCheckVerificationFailure(), marshalPayload() (+70 more)

### Community 4 - "Community 4"
Cohesion: 0.03
Nodes (104): namedImportResolver, contains(), findSubstringIndex(), TestScenario_AuditTrailForCrossDomain(), TestScenario_DomainIsolation(), TestScenario_ExplicitCrossDomainSharing(), TestScenario_GlobalFallback(), TestScenario_MigrationWithBackwardCompatibility() (+96 more)

### Community 5 - "Community 5"
Cohesion: 0.04
Nodes (86): TestUndefinedObligation(), CalculateBackoff(), NewDeadLetterEnvelopeWithRetryAttempt(), contains(), DeadLetterEnvelope, ErrorType, Executor, ClassifyError() (+78 more)

### Community 6 - "Community 6"
Cohesion: 0.03
Nodes (67): TestAMQPDrainTimeout(), TestAMQPHotReloadGraceful(), TestAMQPInFlightTracking(), TestAMQPNoDuplicatesOnReload(), TestAMQPReliabilityAtLeastOnce(), LineageBackend, LineageRecord, NoOpLineageBackend (+59 more)

### Community 7 - "Community 7"
Cohesion: 0.03
Nodes (85): NewManager(), parseTenanConfig(), TestManagerDefaultTenant(), TestManagerMultipleTenants(), TestManagerRegisterTenant(), TestParseTenanConfig(), TestUsageUnlimitedQuota(), TestUsageUtilization() (+77 more)

### Community 8 - "Community 8"
Cohesion: 0.03
Nodes (96): TestAuthValidationAllRoutesHaveAuth(), TestAuthValidationAuthNonePassesValidation(), TestAuthValidationAuthorizeStepPassesValidation(), TestAuthValidationEmptyRoutes(), TestAuthValidationErrorMessageContainsGuidance(), TestAuthValidationMissingAuthStrictMode(), TestAuthValidationMissingAuthWarnMode(), TestAuthValidationMultipleRoutesMixed() (+88 more)

### Community 9 - "Community 9"
Cohesion: 0.03
Nodes (90): CritiqueFinding, CritiquePattern, CritiqueRouteRequest, CritiqueRouteResponse, init(), IntentSignals, MCPOperation, MCPServer (+82 more)

### Community 10 - "Community 10"
Cohesion: 0.05
Nodes (75): ContractStore, SchemaDatasetFacet, SchemaField, GetContractViolation(), IsContractViolation(), NewContractStep(), setupContractStoreWithPaymentSchema(), setupContractStoreWithStrictContract() (+67 more)

### Community 11 - "Community 11"
Cohesion: 0.04
Nodes (55): Cluster, ClusterConfig, CoordinatedGenerationTracker, DedupStore, GenerationTracker, InMemoryDedupStore, Instance, LocalGenerationTracker (+47 more)

### Community 12 - "Community 12"
Cohesion: 0.06
Nodes (41): DefaultConfig(), FromRouteConfig(), HTTPSource, convertAttributesToClaims(), generateHTTPCorrelationID(), KafkaSink, NewKafkaSink(), NewKafkaSinkWithConfig() (+33 more)

### Community 13 - "Community 13"
Cohesion: 0.06
Nodes (45): DefaultConformanceSuite(), TestConformanceSuiteStructure(), TestConformanceTimeout(), TestDeterminismCheck(), TestErrorHandling(), TestMultipleArguments(), TestPluginConformance(), TestSummarizeResults() (+37 more)

### Community 14 - "Community 14"
Cohesion: 0.08
Nodes (29): NewAlternativeRegistry(), NewApicurioClient(), TestApicurioDefaultGroup(), TestApicurioHealth(), TestApicurioRegisterAndFetch(), runSearch(), NewApicurioClient(), NewOpenLineageClient() (+21 more)

### Community 15 - "Community 15"
Cohesion: 0.11
Nodes (31): Ga(), TestViewerIntegrationFullFlow(), TestViewerIntegrationHighLoad(), TestViewerIntegrationMultipleRoutes(), TestViewerIntegrationTimestamp(), Message, NewRingBuffer(), NewRouteViewer() (+23 more)

### Community 16 - "Community 16"
Cohesion: 0.08
Nodes (24): RouteData, StudioServer, YAMLNodeWrapper, StubPDP, TestRoundTripPreservesOrderAndComments(), TestStudioEndToEnd(), TestYAMLRoundTripIntegration(), dataToYAMLNode() (+16 more)

### Community 17 - "Community 17"
Cohesion: 0.08
Nodes (23): applyObligations(), applyRedactionObligation(), isTruthy(), redactFields(), redactInNested(), redactPath(), TestDefaultReplacement(), TestMultipleObligations() (+15 more)

### Community 18 - "Community 18"
Cohesion: 0.07
Nodes (28): AuthorizeSpec, ClaimCheckSpec, ClaimResolveSpec, ContractSpec, ContractStepSpec, ErrorPathSpec, FilterSpec, IdempotentSpec (+20 more)

### Community 19 - "Community 19"
Cohesion: 0.1
Nodes (14): DatabaseSink, SinkConfig, Dataset, Job, OpenLineageEmitter, OpenLineageEvent, Run, SchemaDatasetFacet (+6 more)

### Community 20 - "Community 20"
Cohesion: 0.12
Nodes (19): ImpactQueryRequest, ImpactQueryResponse, BuildImpactIndex(), containsDynamicReferences(), ExtractAllReferences(), TestAddReference(), TestContainsDynamicReferences(), buildImpactSummary() (+11 more)

### Community 21 - "Community 21"
Cohesion: 0.08
Nodes (22): CapabilitiesRequest, CapabilitiesResponse, Capability, LineageRecord, LineageRequest, LineageResponse, OperationErr, OperationError (+14 more)

### Community 22 - "Community 22"
Cohesion: 0.18
Nodes (20): JWTValidator, CreateTestToken(), ExtractBearerToken(), ExtractPrincipalFromHeader(), NewJWTValidator(), parseRSAPublicKey(), TestCreateTestToken(), TestExtractBearerToken() (+12 more)

### Community 23 - "Community 23"
Cohesion: 0.16
Nodes (13): AgentMCPClient, main(), OperationError, Parse MCP response into ResponseEnvelope, Agent for validating and testing routes, Validate a route configuration.          Returns True if valid, False otherwise., Test a route with fixtures.          Returns True if all tests pass, False other, Scaffold a new domain.          Returns True if successful. (+5 more)

### Community 24 - "Community 24"
Cohesion: 0.16
Nodes (8): NewLogBasedCDCSource(), TestDebeziumEventParsing(), TestDebeziumOperationMapping(), TestMaxwellEventParsing(), DebeziumEvent, LogBasedCDCConfig, LogBasedCDCSource, Source

### Community 25 - "Community 25"
Cohesion: 0.31
Nodes (7): NewOPAAdapter(), TestOPAAdapterAllow(), TestOPAAdapterDeny(), TestOPAAdapterMissingResult(), TestOPAAdapterObligations(), TestOPAAdapterRequestTranslation(), OPAAdapter

### Community 26 - "Community 26"
Cohesion: 0.23
Nodes (5): extractStepSchemaFromFull(), fetchRouteSchema(), getDefaultStepSchema(), getStepSchema(), getStepSchemaSync()

### Community 27 - "Community 27"
Cohesion: 0.27
Nodes (2): DatabaseSource, SourceConfig

### Community 28 - "Community 28"
Cohesion: 0.31
Nodes (2): S3Source, SourceConfig

### Community 29 - "Community 29"
Cohesion: 0.33
Nodes (5): ApicurioArtifact, ApicurioResponse, CatalogResult, OpenLineageDataset, SearchResults

### Community 30 - "Community 30"
Cohesion: 0.4
Nodes (4): CallResult, Function, FunctionDef, FunctionType

### Community 31 - "Community 31"
Cohesion: 0.5
Nodes (3): Result, Sink, Source

### Community 32 - "Community 32"
Cohesion: 0.5
Nodes (3): Placeholder, ScaffoldFromIntentRequest, ScaffoldFromIntentResponse

### Community 34 - "Community 34"
Cohesion: 0.67
Nodes (2): PluginLoader, Registry

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (1): Principal

### Community 41 - "Community 41"
Cohesion: 1.0
Nodes (2): M2.3.1 Polling Query Source, Database Adapters Design

### Community 42 - "Community 42"
Cohesion: 1.0
Nodes (2): M2.1 Aggregator Step, Phase 2 Track B (M2.1-M2.7)

### Community 43 - "Community 43"
Cohesion: 1.0
Nodes (2): Authorization & Redaction Pattern, M2.5 Obligations Example

### Community 70 - "Community 70"
Cohesion: 1.0
Nodes (1): Client for calling the Agent Interface operations

### Community 71 - "Community 71"
Cohesion: 1.0
Nodes (1): Initialize MCP client.          Args:             endpoint: Base URL of the agen

### Community 72 - "Community 72"
Cohesion: 1.0
Nodes (1): Call an operation via the MCP interface.          In a real implementation, this

### Community 73 - "Community 73"
Cohesion: 1.0
Nodes (1): Simulate validate_route operation

### Community 74 - "Community 74"
Cohesion: 1.0
Nodes (1): Simulate test_route operation

### Community 75 - "Community 75"
Cohesion: 1.0
Nodes (1): Simulate scaffold_domain operation

### Community 76 - "Community 76"
Cohesion: 1.0
Nodes (1): Parse MCP response into ResponseEnvelope

### Community 77 - "Community 77"
Cohesion: 1.0
Nodes (1): Agent for validating and testing routes

### Community 78 - "Community 78"
Cohesion: 1.0
Nodes (1): Validate a route configuration.          Returns True if valid, False otherwise.

### Community 79 - "Community 79"
Cohesion: 1.0
Nodes (1): Test a route with fixtures.          Returns True if all tests pass, False other

### Community 80 - "Community 80"
Cohesion: 1.0
Nodes (1): Scaffold a new domain.          Returns True if successful.

### Community 81 - "Community 81"
Cohesion: 1.0
Nodes (1): Fragment Parameterization

### Community 82 - "Community 82"
Cohesion: 1.0
Nodes (1): Parameter Syntax ${PARAM:name}

### Community 83 - "Community 83"
Cohesion: 1.0
Nodes (1): Fragment Defaults and Overrides

### Community 84 - "Community 84"
Cohesion: 1.0
Nodes (1): Composition Semantics - Late Binding

### Community 85 - "Community 85"
Cohesion: 1.0
Nodes (1): M2.3.2 Upsert/Insert Sink

### Community 86 - "Community 86"
Cohesion: 1.0
Nodes (1): M2.3.3 CDC Spike Evaluation

### Community 87 - "Community 87"
Cohesion: 1.0
Nodes (1): M2.3.4 CDC Implementation

### Community 88 - "Community 88"
Cohesion: 1.0
Nodes (1): M2.3.4a Trigger-Based CDC Source

### Community 89 - "Community 89"
Cohesion: 1.0
Nodes (1): M2.3.4b Log-Based CDC via Kafka

### Community 90 - "Community 90"
Cohesion: 1.0
Nodes (1): Watermark-Based Incremental Pull Pattern

### Community 91 - "Community 91"
Cohesion: 1.0
Nodes (1): Static Contract Conformance Checking

### Community 92 - "Community 92"
Cohesion: 1.0
Nodes (1): Analyzable Transform Subset

### Community 93 - "Community 93"
Cohesion: 1.0
Nodes (1): Runtime Enforcement enforce:true

### Community 94 - "Community 94"
Cohesion: 1.0
Nodes (1): Purge-Log Auto-Export on Expiry Warning

### Community 95 - "Community 95"
Cohesion: 1.0
Nodes (1): PurgeEvent with Retention Tracking

### Community 96 - "Community 96"
Cohesion: 1.0
Nodes (1): Expiry Warning Lead Days Mechanism

### Community 97 - "Community 97"
Cohesion: 1.0
Nodes (1): Secrets vs Parameters Distinction

### Community 98 - "Community 98"
Cohesion: 1.0
Nodes (1): Log-Based CDC Architecture Decision

### Community 99 - "Community 99"
Cohesion: 1.0
Nodes (1): Trigger-Based CDC Architecture Decision

### Community 100 - "Community 100"
Cohesion: 1.0
Nodes (1): M2.2 Splitter Step

### Community 101 - "Community 101"
Cohesion: 1.0
Nodes (1): M2.3 Database Adapters (JDBC, CDC)

### Community 102 - "Community 102"
Cohesion: 1.0
Nodes (1): M2.5 Authorization Obligations & Redaction

### Community 103 - "Community 103"
Cohesion: 1.0
Nodes (1): M2.6 Purge-Log Auto-Export to S3

### Community 104 - "Community 104"
Cohesion: 1.0
Nodes (1): M2.7 Static Contract Conformance Checking

### Community 105 - "Community 105"
Cohesion: 1.0
Nodes (1): Phase 0 Production Ready (v0.5.0)

### Community 106 - "Community 106"
Cohesion: 1.0
Nodes (1): Phase 1 Track A (R1-R21)

### Community 107 - "Community 107"
Cohesion: 1.0
Nodes (1): Phase 1 Track B (M1.2-M1.8)

### Community 108 - "Community 108"
Cohesion: 1.0
Nodes (1): dim — Object Knowledge Framework

### Community 109 - "Community 109"
Cohesion: 1.0
Nodes (1): Architectural Decision: Language — Go

### Community 110 - "Community 110"
Cohesion: 1.0
Nodes (1): Architectural Decision: Hot Reload with Generation Takeover

### Community 111 - "Community 111"
Cohesion: 1.0
Nodes (1): Architectural Decision: Per-Instance Embedded Lineage Store

### Community 112 - "Community 112"
Cohesion: 1.0
Nodes (1): Architectural Decision: Structural Authorization Declaration

### Community 113 - "Community 113"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Channels and Backpressure

### Community 114 - "Community 114"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Executor Worker Pool

### Community 115 - "Community 115"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Hot Reload State Machine

### Community 116 - "Community 116"
Cohesion: 1.0
Nodes (1): Obligation Type: Redaction (redact_fields)

### Community 117 - "Community 117"
Cohesion: 1.0
Nodes (1): Obligation Feature: Field-Path Resolution

### Community 118 - "Community 118"
Cohesion: 1.0
Nodes (1): Obligation Feature: Lineage Tracking (Obligation Facets)

### Community 119 - "Community 119"
Cohesion: 1.0
Nodes (1): Community: Authorization & Obligations

### Community 120 - "Community 120"
Cohesion: 1.0
Nodes (1): Community: Aggregator Step

### Community 121 - "Community 121"
Cohesion: 1.0
Nodes (1): Community: Database Adapters

### Community 122 - "Community 122"
Cohesion: 1.0
Nodes (1): CDC Pattern (Log-Based & Trigger-Based)

### Community 123 - "Community 123"
Cohesion: 1.0
Nodes (1): Graph Structure Analysis & Insights

### Community 124 - "Community 124"
Cohesion: 1.0
Nodes (1): Analysis: False Positive Isolated Nodes

### Community 125 - "Community 125"
Cohesion: 1.0
Nodes (1): Phase 2 Completion & Exit Criteria

## Knowledge Gaps
- **231 isolated node(s):** `OperationError`, `OperationMetadata`, `ConformanceTest`, `ConformanceResult`, `ConformanceSuite` (+226 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 27`** (10 nodes): `DatabaseSource`, `.Close()`, `.GetLastPollTime()`, `.GetWatermark()`, `.HealthCheck()`, `.poll()`, `.SetWatermark()`, `.Start()`, `SourceConfig`, `database_source.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 28`** (9 nodes): `s3_source.go`, `S3Source`, `.Checkpoint()`, `.Close()`, `.HealthCheck()`, `.poll()`, `.Start()`, `NewS3Source()`, `SourceConfig`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (3 nodes): `registry.go`, `PluginLoader`, `Registry`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (2 nodes): `Principal`, `principal.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (2 nodes): `M2.3.1 Polling Query Source`, `Database Adapters Design`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (2 nodes): `M2.1 Aggregator Step`, `Phase 2 Track B (M2.1-M2.7)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (2 nodes): `Authorization & Redaction Pattern`, `M2.5 Obligations Example`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (1 nodes): `Client for calling the Agent Interface operations`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 71`** (1 nodes): `Initialize MCP client.          Args:             endpoint: Base URL of the agen`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 72`** (1 nodes): `Call an operation via the MCP interface.          In a real implementation, this`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (1 nodes): `Simulate validate_route operation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 74`** (1 nodes): `Simulate test_route operation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 75`** (1 nodes): `Simulate scaffold_domain operation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 76`** (1 nodes): `Parse MCP response into ResponseEnvelope`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (1 nodes): `Agent for validating and testing routes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (1 nodes): `Validate a route configuration.          Returns True if valid, False otherwise.`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 79`** (1 nodes): `Test a route with fixtures.          Returns True if all tests pass, False other`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 80`** (1 nodes): `Scaffold a new domain.          Returns True if successful.`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 81`** (1 nodes): `Fragment Parameterization`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 82`** (1 nodes): `Parameter Syntax ${PARAM:name}`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (1 nodes): `Fragment Defaults and Overrides`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (1 nodes): `Composition Semantics - Late Binding`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 85`** (1 nodes): `M2.3.2 Upsert/Insert Sink`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 86`** (1 nodes): `M2.3.3 CDC Spike Evaluation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 87`** (1 nodes): `M2.3.4 CDC Implementation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 88`** (1 nodes): `M2.3.4a Trigger-Based CDC Source`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 89`** (1 nodes): `M2.3.4b Log-Based CDC via Kafka`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 90`** (1 nodes): `Watermark-Based Incremental Pull Pattern`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 91`** (1 nodes): `Static Contract Conformance Checking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 92`** (1 nodes): `Analyzable Transform Subset`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 93`** (1 nodes): `Runtime Enforcement enforce:true`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 94`** (1 nodes): `Purge-Log Auto-Export on Expiry Warning`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 95`** (1 nodes): `PurgeEvent with Retention Tracking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 96`** (1 nodes): `Expiry Warning Lead Days Mechanism`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 97`** (1 nodes): `Secrets vs Parameters Distinction`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 98`** (1 nodes): `Log-Based CDC Architecture Decision`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 99`** (1 nodes): `Trigger-Based CDC Architecture Decision`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 100`** (1 nodes): `M2.2 Splitter Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 101`** (1 nodes): `M2.3 Database Adapters (JDBC, CDC)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 102`** (1 nodes): `M2.5 Authorization Obligations & Redaction`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 103`** (1 nodes): `M2.6 Purge-Log Auto-Export to S3`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 104`** (1 nodes): `M2.7 Static Contract Conformance Checking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 105`** (1 nodes): `Phase 0 Production Ready (v0.5.0)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 106`** (1 nodes): `Phase 1 Track A (R1-R21)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 107`** (1 nodes): `Phase 1 Track B (M1.2-M1.8)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 108`** (1 nodes): `dim — Object Knowledge Framework`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 109`** (1 nodes): `Architectural Decision: Language — Go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 110`** (1 nodes): `Architectural Decision: Hot Reload with Generation Takeover`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 111`** (1 nodes): `Architectural Decision: Per-Instance Embedded Lineage Store`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 112`** (1 nodes): `Architectural Decision: Structural Authorization Declaration`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 113`** (1 nodes): `Concurrency Pattern: Channels and Backpressure`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 114`** (1 nodes): `Concurrency Pattern: Executor Worker Pool`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 115`** (1 nodes): `Concurrency Pattern: Hot Reload State Machine`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 116`** (1 nodes): `Obligation Type: Redaction (redact_fields)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 117`** (1 nodes): `Obligation Feature: Field-Path Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 118`** (1 nodes): `Obligation Feature: Lineage Tracking (Obligation Facets)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 119`** (1 nodes): `Community: Authorization & Obligations`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 120`** (1 nodes): `Community: Aggregator Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 121`** (1 nodes): `Community: Database Adapters`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 122`** (1 nodes): `CDC Pattern (Log-Based & Trigger-Based)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 123`** (1 nodes): `Graph Structure Analysis & Insights`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 124`** (1 nodes): `Analysis: False Positive Isolated Nodes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 125`** (1 nodes): `Phase 2 Completion & Exit Criteria`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewMessage()` connect `Community 1` to `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 10`, `Community 17`, `Community 24`, `Community 27`, `Community 28`?**
  _High betweenness centrality (0.088) - this node is a cross-community bridge._
- **Why does `LoadRouteConfig()` connect `Community 8` to `Community 1`, `Community 3`, `Community 6`, `Community 9`, `Community 12`, `Community 13`, `Community 16`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 5` to `Community 0`, `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 8`, `Community 9`, `Community 11`, `Community 12`, `Community 16`?**
  _High betweenness centrality (0.034) - this node is a cross-community bridge._
- **Are the 244 inferred relationships involving `NewMessage()` (e.g. with `TestRecordLineage()` and `TestConcurrentWrites()`) actually correct?**
  _`NewMessage()` has 244 INFERRED edges - model-reasoned connections that need verification._
- **Are the 139 inferred relationships involving `NewChannel()` (e.g. with `TestWiretapMessageCopiedToTapSink()` and `TestWiretapOriginalContinuesDownstream()`) actually correct?**
  _`NewChannel()` has 139 INFERRED edges - model-reasoned connections that need verification._
- **Are the 54 inferred relationships involving `contains()` (e.g. with `TestExportInvalidFormat()` and `TestAuthorizeStepNewAuthorizeStepInvalidMode()`) actually correct?**
  _`contains()` has 54 INFERRED edges - model-reasoned connections that need verification._
- **Are the 44 inferred relationships involving `NewExecutor()` (e.g. with `TestWiretapIntegrationWithExecutor()` and `TestFileSinkIntegrationEndToEnd()`) actually correct?**
  _`NewExecutor()` has 44 INFERRED edges - model-reasoned connections that need verification._