# Graph Report - dim  (2026-11-20)

## Corpus Check
- 260 files · ~416,665 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2781 nodes · 7580 edges · 94 communities detected
- Extraction: 51% EXTRACTED · 49% INFERRED · 0% AMBIGUOUS · INFERRED: 3689 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 69|Community 69]]
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

## God Nodes (most connected - your core abstractions)
1. `NewMessage()` - 246 edges
2. `NewChannel()` - 140 edges
3. `contains()` - 54 edges
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
Nodes (471): _0(), _a(), A0(), a1(), Aa(), ac(), Ad(), Ae() (+463 more)

### Community 1 - "Community 1"
Cohesion: 0.02
Nodes (233): BenchmarkAMQPMessageConversion(), TestAMQPSinkMetrics(), TestAMQPSourceBasic(), TestAMQPSourceClose(), NewAMQPSink(), NewAMQPSinkWithConfig(), SinkConfig, TestApicurioQueryContracts() (+225 more)

### Community 2 - "Community 2"
Cohesion: 0.02
Nodes (189): TestAggregatorIntegration(), TestAggregatorMultipleOrdersIntegration(), TestAggregatorTimeoutCompletion(), TestAggregatorWithComplexMessages(), NewAggregateStep(), TestAggregateCountCompletion(), TestAggregateDrain(), TestAggregateMultipleCorrelationKeys() (+181 more)

### Community 3 - "Community 3"
Cohesion: 0.03
Nodes (109): namedImportResolver, contains(), findSubstringIndex(), TestScenario_AuditTrailForCrossDomain(), TestScenario_DomainIsolation(), TestScenario_ExplicitCrossDomainSharing(), TestScenario_GlobalFallback(), TestScenario_MigrationWithBackwardCompatibility() (+101 more)

### Community 4 - "Community 4"
Cohesion: 0.03
Nodes (98): CalculateBackoff(), DefaultConfig(), FromRouteConfig(), NewManager(), parseTenanConfig(), TestManagerDefaultTenant(), TestManagerMultipleTenants(), TestManagerRegisterTenant() (+90 more)

### Community 5 - "Community 5"
Cohesion: 0.03
Nodes (67): TestAMQPDrainTimeout(), TestAMQPHotReloadGraceful(), TestAMQPInFlightTracking(), TestAMQPNoDuplicatesOnReload(), TestAMQPReliabilityAtLeastOnce(), LineageBackend, LineageRecord, NoOpLineageBackend (+59 more)

### Community 6 - "Community 6"
Cohesion: 0.03
Nodes (53): NewAlternativeRegistry(), AMQPSink, isVersionListResponse(), NewApicurioClient(), TestApicurioDefaultGroup(), TestApicurioHealth(), TestApicurioRegisterAndFetch(), loadRoute() (+45 more)

### Community 7 - "Community 7"
Cohesion: 0.03
Nodes (94): CritiqueFinding, CritiquePattern, CritiqueRouteRequest, CritiqueRouteResponse, ImpactQueryRequest, ImpactQueryResponse, init(), IntentSignals (+86 more)

### Community 8 - "Community 8"
Cohesion: 0.04
Nodes (84): AuthValidationMode, FragmentResolver, RouteVersion, sortedMap, extractParams(), hasImport(), NewFragmentResolver(), TestConfigMergingOverride() (+76 more)

### Community 9 - "Community 9"
Cohesion: 0.04
Nodes (55): Cluster, ClusterConfig, CoordinatedGenerationTracker, DedupStore, GenerationTracker, InMemoryDedupStore, Instance, LocalGenerationTracker (+47 more)

### Community 10 - "Community 10"
Cohesion: 0.05
Nodes (57): TestClaimCheckCleanup(), TestClaimCheckMultipleAttachments(), TestClaimCheckOrderWithAttachmentWorkflow(), TestClaimCheckVerificationFailure(), marshalPayload(), NewClaimCheckStep(), ResolveClaimCheck(), TestClaimCheckCorrelationIDPreserved() (+49 more)

### Community 11 - "Community 11"
Cohesion: 0.07
Nodes (57): ContractStore, SchemaDatasetFacet, SchemaField, GetContractViolation(), IsContractViolation(), NewContractStep(), setupContractStoreWithPaymentSchema(), setupContractStoreWithStrictContract() (+49 more)

### Community 12 - "Community 12"
Cohesion: 0.08
Nodes (46): Principal, Span, SpanEvent, SpanStreamer, stepError, TracingProvider, TestIntegration_FullTraceTailWorkflow(), TestIntegration_MultipleRoutesFiltering() (+38 more)

### Community 13 - "Community 13"
Cohesion: 0.05
Nodes (35): DatabaseSink, SinkConfig, FileSink, HTTPSink, HTTPSinkConfig, OBOConfig, NewHTTPSink(), TestHTTPSinkBadJSON() (+27 more)

### Community 14 - "Community 14"
Cohesion: 0.05
Nodes (40): TestAuthValidationAllRoutesHaveAuth(), TestAuthValidationAuthNonePassesValidation(), TestAuthValidationAuthorizeStepPassesValidation(), TestAuthValidationEmptyRoutes(), TestAuthValidationErrorMessageContainsGuidance(), TestAuthValidationMissingAuthStrictMode(), TestAuthValidationMissingAuthWarnMode(), TestAuthValidationMultipleRoutesMixed() (+32 more)

### Community 15 - "Community 15"
Cohesion: 0.06
Nodes (46): DefaultConformanceSuite(), TestConformanceSuiteStructure(), TestConformanceTimeout(), TestDeterminismCheck(), TestErrorHandling(), TestMultipleArguments(), TestPluginConformance(), TestSummarizeResults() (+38 more)

### Community 16 - "Community 16"
Cohesion: 0.12
Nodes (30): TestViewerIntegrationFullFlow(), TestViewerIntegrationHighLoad(), TestViewerIntegrationMultipleRoutes(), TestViewerIntegrationTimestamp(), Message, NewRingBuffer(), NewRouteViewer(), NewViewerServer() (+22 more)

### Community 17 - "Community 17"
Cohesion: 0.16
Nodes (23): getPercentile(), NewMetricsCollector(), BenchmarkMetricsCollector_GetMetrics(), BenchmarkMetricsCollector_RecordMessageSuccess(), TestMetricsCollector_Disable(), TestMetricsCollector_GetMetrics_PrometheusFormat(), TestMetricsCollector_LatencyAverage(), TestMetricsCollector_LatencyPercentiles() (+15 more)

### Community 18 - "Community 18"
Cohesion: 0.07
Nodes (28): AuthorizeSpec, ClaimCheckSpec, ClaimResolveSpec, ContractSpec, ContractStepSpec, ErrorPathSpec, FilterSpec, IdempotentSpec (+20 more)

### Community 19 - "Community 19"
Cohesion: 0.12
Nodes (23): JWTValidator, HTTPSource, convertAttributesToClaims(), generateHTTPCorrelationID(), CreateTestToken(), ExtractBearerToken(), ExtractPrincipalFromHeader(), NewJWTValidator() (+15 more)

### Community 20 - "Community 20"
Cohesion: 0.08
Nodes (22): CapabilitiesRequest, CapabilitiesResponse, Capability, LineageRecord, LineageRequest, LineageResponse, OperationErr, OperationError (+14 more)

### Community 21 - "Community 21"
Cohesion: 0.16
Nodes (13): AgentMCPClient, main(), OperationError, Parse MCP response into ResponseEnvelope, Agent for validating and testing routes, Validate a route configuration.          Returns True if valid, False otherwise., Test a route with fixtures.          Returns True if all tests pass, False other, Scaffold a new domain.          Returns True if successful. (+5 more)

### Community 22 - "Community 22"
Cohesion: 0.15
Nodes (12): isTruthy(), redactFields(), redactInNested(), redactPath(), AuthorizeStep, PDPContext, PDPDecisionRequest, PDPDecisionResponse (+4 more)

### Community 23 - "Community 23"
Cohesion: 0.31
Nodes (7): NewOPAAdapter(), TestOPAAdapterAllow(), TestOPAAdapterDeny(), TestOPAAdapterMissingResult(), TestOPAAdapterObligations(), TestOPAAdapterRequestTranslation(), OPAAdapter

### Community 24 - "Community 24"
Cohesion: 0.29
Nodes (7): luhn_check(), malloc(), read_string(), transform_phone(), validate_credit_card(), validate_email(), write_string()

### Community 25 - "Community 25"
Cohesion: 0.23
Nodes (5): extractStepSchemaFromFull(), fetchRouteSchema(), getDefaultStepSchema(), getStepSchema(), getStepSchemaSync()

### Community 26 - "Community 26"
Cohesion: 0.24
Nodes (4): KafkaSink, NewKafkaSink(), NewKafkaSinkWithConfig(), SinkConfig

### Community 27 - "Community 27"
Cohesion: 0.25
Nodes (4): AMQPSource, NewAMQPSource(), NewAMQPSourceWithConfig(), SourceConfig

### Community 28 - "Community 28"
Cohesion: 0.48
Nodes (5): extractAdapterTypeEnum(), extractSchema(), extractStepType(), generateManifestFromSchema(), getDescription()

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

### Community 40 - "Community 40"
Cohesion: 1.0
Nodes (2): M2.3.1 Polling Query Source, Database Adapters Design

### Community 41 - "Community 41"
Cohesion: 1.0
Nodes (2): M2.1 Aggregator Step, Phase 2 Track B (M2.1-M2.7)

### Community 42 - "Community 42"
Cohesion: 1.0
Nodes (2): Authorization & Redaction Pattern, M2.5 Obligations Example

### Community 68 - "Community 68"
Cohesion: 1.0
Nodes (1): Client for calling the Agent Interface operations

### Community 69 - "Community 69"
Cohesion: 1.0
Nodes (1): Initialize MCP client.          Args:             endpoint: Base URL of the agen

### Community 70 - "Community 70"
Cohesion: 1.0
Nodes (1): Call an operation via the MCP interface.          In a real implementation, this

### Community 71 - "Community 71"
Cohesion: 1.0
Nodes (1): Simulate validate_route operation

### Community 72 - "Community 72"
Cohesion: 1.0
Nodes (1): Simulate test_route operation

### Community 73 - "Community 73"
Cohesion: 1.0
Nodes (1): Simulate scaffold_domain operation

### Community 74 - "Community 74"
Cohesion: 1.0
Nodes (1): Parse MCP response into ResponseEnvelope

### Community 75 - "Community 75"
Cohesion: 1.0
Nodes (1): Agent for validating and testing routes

### Community 76 - "Community 76"
Cohesion: 1.0
Nodes (1): Validate a route configuration.          Returns True if valid, False otherwise.

### Community 77 - "Community 77"
Cohesion: 1.0
Nodes (1): Test a route with fixtures.          Returns True if all tests pass, False other

### Community 78 - "Community 78"
Cohesion: 1.0
Nodes (1): Scaffold a new domain.          Returns True if successful.

### Community 79 - "Community 79"
Cohesion: 1.0
Nodes (1): Fragment Parameterization

### Community 80 - "Community 80"
Cohesion: 1.0
Nodes (1): Parameter Syntax ${PARAM:name}

### Community 81 - "Community 81"
Cohesion: 1.0
Nodes (1): Fragment Defaults and Overrides

### Community 82 - "Community 82"
Cohesion: 1.0
Nodes (1): Composition Semantics - Late Binding

### Community 83 - "Community 83"
Cohesion: 1.0
Nodes (1): M2.3.2 Upsert/Insert Sink

### Community 84 - "Community 84"
Cohesion: 1.0
Nodes (1): M2.3.3 CDC Spike Evaluation

### Community 85 - "Community 85"
Cohesion: 1.0
Nodes (1): M2.3.4 CDC Implementation

### Community 86 - "Community 86"
Cohesion: 1.0
Nodes (1): M2.3.4a Trigger-Based CDC Source

### Community 87 - "Community 87"
Cohesion: 1.0
Nodes (1): M2.3.4b Log-Based CDC via Kafka

### Community 88 - "Community 88"
Cohesion: 1.0
Nodes (1): Watermark-Based Incremental Pull Pattern

### Community 89 - "Community 89"
Cohesion: 1.0
Nodes (1): Static Contract Conformance Checking

### Community 90 - "Community 90"
Cohesion: 1.0
Nodes (1): Analyzable Transform Subset

### Community 91 - "Community 91"
Cohesion: 1.0
Nodes (1): Runtime Enforcement enforce:true

### Community 92 - "Community 92"
Cohesion: 1.0
Nodes (1): Purge-Log Auto-Export on Expiry Warning

### Community 93 - "Community 93"
Cohesion: 1.0
Nodes (1): PurgeEvent with Retention Tracking

### Community 94 - "Community 94"
Cohesion: 1.0
Nodes (1): Expiry Warning Lead Days Mechanism

### Community 95 - "Community 95"
Cohesion: 1.0
Nodes (1): Secrets vs Parameters Distinction

### Community 96 - "Community 96"
Cohesion: 1.0
Nodes (1): Log-Based CDC Architecture Decision

### Community 97 - "Community 97"
Cohesion: 1.0
Nodes (1): Trigger-Based CDC Architecture Decision

### Community 98 - "Community 98"
Cohesion: 1.0
Nodes (1): M2.2 Splitter Step

### Community 99 - "Community 99"
Cohesion: 1.0
Nodes (1): M2.3 Database Adapters (JDBC, CDC)

### Community 100 - "Community 100"
Cohesion: 1.0
Nodes (1): M2.5 Authorization Obligations & Redaction

### Community 101 - "Community 101"
Cohesion: 1.0
Nodes (1): M2.6 Purge-Log Auto-Export to S3

### Community 102 - "Community 102"
Cohesion: 1.0
Nodes (1): M2.7 Static Contract Conformance Checking

### Community 103 - "Community 103"
Cohesion: 1.0
Nodes (1): Phase 0 Production Ready (v0.5.0)

### Community 104 - "Community 104"
Cohesion: 1.0
Nodes (1): Phase 1 Track A (R1-R21)

### Community 105 - "Community 105"
Cohesion: 1.0
Nodes (1): Phase 1 Track B (M1.2-M1.8)

### Community 106 - "Community 106"
Cohesion: 1.0
Nodes (1): dim — Object Knowledge Framework

### Community 107 - "Community 107"
Cohesion: 1.0
Nodes (1): Architectural Decision: Language — Go

### Community 108 - "Community 108"
Cohesion: 1.0
Nodes (1): Architectural Decision: Hot Reload with Generation Takeover

### Community 109 - "Community 109"
Cohesion: 1.0
Nodes (1): Architectural Decision: Per-Instance Embedded Lineage Store

### Community 110 - "Community 110"
Cohesion: 1.0
Nodes (1): Architectural Decision: Structural Authorization Declaration

### Community 111 - "Community 111"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Channels and Backpressure

### Community 112 - "Community 112"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Executor Worker Pool

### Community 113 - "Community 113"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Hot Reload State Machine

### Community 114 - "Community 114"
Cohesion: 1.0
Nodes (1): Obligation Type: Redaction (redact_fields)

### Community 115 - "Community 115"
Cohesion: 1.0
Nodes (1): Obligation Feature: Field-Path Resolution

### Community 116 - "Community 116"
Cohesion: 1.0
Nodes (1): Obligation Feature: Lineage Tracking (Obligation Facets)

### Community 117 - "Community 117"
Cohesion: 1.0
Nodes (1): Community: Authorization & Obligations

### Community 118 - "Community 118"
Cohesion: 1.0
Nodes (1): Community: Aggregator Step

### Community 119 - "Community 119"
Cohesion: 1.0
Nodes (1): Community: Database Adapters

### Community 120 - "Community 120"
Cohesion: 1.0
Nodes (1): CDC Pattern (Log-Based & Trigger-Based)

### Community 121 - "Community 121"
Cohesion: 1.0
Nodes (1): Graph Structure Analysis & Insights

### Community 122 - "Community 122"
Cohesion: 1.0
Nodes (1): Analysis: False Positive Isolated Nodes

### Community 123 - "Community 123"
Cohesion: 1.0
Nodes (1): Phase 2 Completion & Exit Criteria

## Knowledge Gaps
- **231 isolated node(s):** `OperationError`, `OperationMetadata`, `ConformanceTest`, `ConformanceResult`, `ConformanceSuite` (+226 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 34`** (3 nodes): `registry.go`, `PluginLoader`, `Registry`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (2 nodes): `Principal`, `principal.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (2 nodes): `M2.3.1 Polling Query Source`, `Database Adapters Design`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (2 nodes): `M2.1 Aggregator Step`, `Phase 2 Track B (M2.1-M2.7)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (2 nodes): `Authorization & Redaction Pattern`, `M2.5 Obligations Example`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 68`** (1 nodes): `Client for calling the Agent Interface operations`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 69`** (1 nodes): `Initialize MCP client.          Args:             endpoint: Base URL of the agen`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (1 nodes): `Call an operation via the MCP interface.          In a real implementation, this`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 71`** (1 nodes): `Simulate validate_route operation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 72`** (1 nodes): `Simulate test_route operation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (1 nodes): `Simulate scaffold_domain operation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 74`** (1 nodes): `Parse MCP response into ResponseEnvelope`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 75`** (1 nodes): `Agent for validating and testing routes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 76`** (1 nodes): `Validate a route configuration.          Returns True if valid, False otherwise.`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (1 nodes): `Test a route with fixtures.          Returns True if all tests pass, False other`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (1 nodes): `Scaffold a new domain.          Returns True if successful.`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 79`** (1 nodes): `Fragment Parameterization`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 80`** (1 nodes): `Parameter Syntax ${PARAM:name}`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 81`** (1 nodes): `Fragment Defaults and Overrides`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 82`** (1 nodes): `Composition Semantics - Late Binding`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (1 nodes): `M2.3.2 Upsert/Insert Sink`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (1 nodes): `M2.3.3 CDC Spike Evaluation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 85`** (1 nodes): `M2.3.4 CDC Implementation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 86`** (1 nodes): `M2.3.4a Trigger-Based CDC Source`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 87`** (1 nodes): `M2.3.4b Log-Based CDC via Kafka`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 88`** (1 nodes): `Watermark-Based Incremental Pull Pattern`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 89`** (1 nodes): `Static Contract Conformance Checking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 90`** (1 nodes): `Analyzable Transform Subset`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 91`** (1 nodes): `Runtime Enforcement enforce:true`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 92`** (1 nodes): `Purge-Log Auto-Export on Expiry Warning`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 93`** (1 nodes): `PurgeEvent with Retention Tracking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 94`** (1 nodes): `Expiry Warning Lead Days Mechanism`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 95`** (1 nodes): `Secrets vs Parameters Distinction`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 96`** (1 nodes): `Log-Based CDC Architecture Decision`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 97`** (1 nodes): `Trigger-Based CDC Architecture Decision`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 98`** (1 nodes): `M2.2 Splitter Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 99`** (1 nodes): `M2.3 Database Adapters (JDBC, CDC)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 100`** (1 nodes): `M2.5 Authorization Obligations & Redaction`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 101`** (1 nodes): `M2.6 Purge-Log Auto-Export to S3`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 102`** (1 nodes): `M2.7 Static Contract Conformance Checking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 103`** (1 nodes): `Phase 0 Production Ready (v0.5.0)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 104`** (1 nodes): `Phase 1 Track A (R1-R21)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 105`** (1 nodes): `Phase 1 Track B (M1.2-M1.8)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 106`** (1 nodes): `dim — Object Knowledge Framework`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 107`** (1 nodes): `Architectural Decision: Language — Go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 108`** (1 nodes): `Architectural Decision: Hot Reload with Generation Takeover`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 109`** (1 nodes): `Architectural Decision: Per-Instance Embedded Lineage Store`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 110`** (1 nodes): `Architectural Decision: Structural Authorization Declaration`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 111`** (1 nodes): `Concurrency Pattern: Channels and Backpressure`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 112`** (1 nodes): `Concurrency Pattern: Executor Worker Pool`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 113`** (1 nodes): `Concurrency Pattern: Hot Reload State Machine`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 114`** (1 nodes): `Obligation Type: Redaction (redact_fields)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 115`** (1 nodes): `Obligation Feature: Field-Path Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 116`** (1 nodes): `Obligation Feature: Lineage Tracking (Obligation Facets)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 117`** (1 nodes): `Community: Authorization & Obligations`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 118`** (1 nodes): `Community: Aggregator Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 119`** (1 nodes): `Community: Database Adapters`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 120`** (1 nodes): `CDC Pattern (Log-Based & Trigger-Based)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 121`** (1 nodes): `Graph Structure Analysis & Insights`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 122`** (1 nodes): `Analysis: False Positive Isolated Nodes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 123`** (1 nodes): `Phase 2 Completion & Exit Criteria`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewMessage()` connect `Community 1` to `Community 2`, `Community 3`, `Community 5`, `Community 6`, `Community 10`, `Community 11`, `Community 13`, `Community 27`?**
  _High betweenness centrality (0.102) - this node is a cross-community bridge._
- **Why does `LoadRouteConfig()` connect `Community 8` to `Community 1`, `Community 2`, `Community 4`, `Community 5`, `Community 7`, `Community 14`, `Community 15`?**
  _High betweenness centrality (0.056) - this node is a cross-community bridge._
- **Why does `runMultiRoute()` connect `Community 4` to `Community 1`, `Community 2`, `Community 8`, `Community 12`, `Community 17`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Are the 244 inferred relationships involving `NewMessage()` (e.g. with `TestRecordLineage()` and `TestConcurrentWrites()`) actually correct?**
  _`NewMessage()` has 244 INFERRED edges - model-reasoned connections that need verification._
- **Are the 139 inferred relationships involving `NewChannel()` (e.g. with `TestWiretapMessageCopiedToTapSink()` and `TestWiretapOriginalContinuesDownstream()`) actually correct?**
  _`NewChannel()` has 139 INFERRED edges - model-reasoned connections that need verification._
- **Are the 53 inferred relationships involving `contains()` (e.g. with `TestExportInvalidFormat()` and `TestAuthorizeStepNewAuthorizeStepInvalidMode()`) actually correct?**
  _`contains()` has 53 INFERRED edges - model-reasoned connections that need verification._
- **Are the 44 inferred relationships involving `NewExecutor()` (e.g. with `TestWiretapIntegrationWithExecutor()` and `TestFileSinkIntegrationEndToEnd()`) actually correct?**
  _`NewExecutor()` has 44 INFERRED edges - model-reasoned connections that need verification._