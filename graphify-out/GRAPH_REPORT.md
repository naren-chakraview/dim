# Graph Report - dim  (2026-09-06)

## Corpus Check
- 192 files · ~303,345 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1779 nodes · 4799 edges · 81 communities detected
- Extraction: 38% EXTRACTED · 62% INFERRED · 0% AMBIGUOUS · INFERRED: 2963 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
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

## God Nodes (most connected - your core abstractions)
1. `NewMessage()` - 246 edges
2. `NewChannel()` - 132 edges
3. `NewExecutor()` - 46 edges
4. `NewTracingProvider()` - 41 edges
5. `NewContractStore()` - 37 edges
6. `NewAuthorizeStep()` - 34 edges
7. `NewStore()` - 33 edges
8. `LoadRouteConfig()` - 32 edges
9. `contains()` - 28 edges
10. `NewRouteStep()` - 27 edges

## Surprising Connections (you probably didn't know these)
- `BuildStepsFromSpec()` --calls--> `buildExecutorForRoute()`  [INFERRED]
  internal/steps/factory.go → cmd/dimctl/main.go
- `NewTranslateStep()` --calls--> `BenchmarkMessageThroughput()`  [INFERRED]
  internal/steps/translate.go → examples/bench/throughput_test.go
- `NewFilterStep()` --calls--> `BenchmarkMessageThroughput()`  [INFERRED]
  internal/steps/filter.go → examples/bench/throughput_test.go
- `NewAuthorizeStep()` --calls--> `BenchmarkMessageThroughput()`  [INFERRED]
  internal/steps/authorize.go → examples/bench/throughput_test.go
- `NewContractStore()` --calls--> `buildExecutorForRoute()`  [INFERRED]
  internal/config/contracts.go → cmd/dimctl/main.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.03
Nodes (173): TestAMQPSourceBasic(), TestAMQPSourceClose(), CalculateBackoff(), TestLogBasedCDCConfigValidation(), TestTriggerBasedCDCSourceCreation(), TestTriggerCDCConfigValidation(), TestTriggerCDCWatermarkTracking(), NewTriggerBasedCDCSource() (+165 more)

### Community 1 - "Community 1"
Cohesion: 0.03
Nodes (194): TestAggregatorIntegration(), TestAggregatorMultipleOrdersIntegration(), TestAggregatorTimeoutCompletion(), TestAggregatorWithComplexMessages(), NewAggregateStep(), TestAggregateCountCompletion(), TestAggregateDrain(), TestAggregateMultipleCorrelationKeys() (+186 more)

### Community 2 - "Community 2"
Cohesion: 0.04
Nodes (61): TestAMQPDrainTimeout(), TestAMQPHotReloadGraceful(), TestAMQPInFlightTracking(), TestAMQPNoDuplicatesOnReload(), TestAMQPReliabilityAtLeastOnce(), LineageBackend, LineageRecord, NoOpLineageBackend (+53 more)

### Community 3 - "Community 3"
Cohesion: 0.05
Nodes (69): Principal, Span, SpanEvent, SpanStreamer, stepError, TailOptions, TracingProvider, Result (+61 more)

### Community 4 - "Community 4"
Cohesion: 0.04
Nodes (39): NewAlternativeRegistry(), isVersionListResponse(), NewApicurioClient(), TestApicurioDefaultGroup(), TestApicurioHealth(), TestApicurioRegisterAndFetch(), TestLoadRegistryBackedContract(), FileSource (+31 more)

### Community 5 - "Community 5"
Cohesion: 0.05
Nodes (74): ContractStore, SchemaDatasetFacet, SchemaField, GetContractViolation(), IsContractViolation(), NewContractStep(), setupContractStoreWithPaymentSchema(), setupContractStoreWithStrictContract() (+66 more)

### Community 6 - "Community 6"
Cohesion: 0.04
Nodes (78): AuthValidationMode, FragmentResolver, RouteVersion, sortedMap, extractParams(), hasImport(), NewFragmentResolver(), TestConfigMergingOverride() (+70 more)

### Community 7 - "Community 7"
Cohesion: 0.04
Nodes (50): Cluster, ClusterConfig, CoordinatedGenerationTracker, DedupStore, GenerationTracker, InMemoryDedupStore, Instance, LocalGenerationTracker (+42 more)

### Community 8 - "Community 8"
Cohesion: 0.04
Nodes (45): AMQPSink, BenchmarkAMQPMessageConversion(), TestAMQPSinkMetrics(), NewAMQPSink(), NewAMQPSinkWithConfig(), SinkConfig, DatabaseSink, NewDatabaseSink() (+37 more)

### Community 9 - "Community 9"
Cohesion: 0.04
Nodes (41): TestClaimCheckCleanup(), TestClaimCheckVerificationFailure(), marshalPayload(), ResolveClaimCheck(), TestClaimCheckStoreInMemory(), ClaimCheckMetadata, ClaimCheckStore, InMemoryClaimCheckStore (+33 more)

### Community 10 - "Community 10"
Cohesion: 0.06
Nodes (42): DefaultConfig(), FromRouteConfig(), HTTPSource, convertAttributesToClaims(), generateHTTPCorrelationID(), BenchmarkKafkaMessageConversion(), TestKafkaSinkMetrics(), KafkaSink (+34 more)

### Community 11 - "Community 11"
Cohesion: 0.07
Nodes (32): Evaluator, FunctionDef, FunctionImpl, FunctionType, PluginInterface, PluginRuntime, Registry, WASMRuntime (+24 more)

### Community 12 - "Community 12"
Cohesion: 0.06
Nodes (40): FileSinkAdapter, HTTPSourceAdapter, SinkAdapter, SourceAdapter, buildExecutorForRoute(), GetOrderingMode(), GetWorkerCount(), IsOrderingRequired() (+32 more)

### Community 13 - "Community 13"
Cohesion: 0.1
Nodes (30): NewManager(), parseTenanConfig(), TestManagerDefaultTenant(), TestManagerMultipleTenants(), TestManagerRegisterTenant(), TestParseTenanConfig(), TestUsageUnlimitedQuota(), TestUsageUtilization() (+22 more)

### Community 14 - "Community 14"
Cohesion: 0.14
Nodes (25): TestViewerIntegrationFullFlow(), TestViewerIntegrationHighLoad(), TestViewerIntegrationMultipleRoutes(), TestViewerIntegrationTimestamp(), Message, NewRingBuffer(), NewRouteViewer(), NewViewerServer() (+17 more)

### Community 15 - "Community 15"
Cohesion: 0.08
Nodes (20): DefaultConformanceSuite(), TestConformanceSuiteStructure(), TestConformanceTimeout(), TestDeterminismCheck(), TestErrorHandling(), TestMultipleArguments(), TestPluginConformance(), TestSummarizeResults() (+12 more)

### Community 16 - "Community 16"
Cohesion: 0.07
Nodes (26): AuthorizeSpec, ContractSpec, ContractStepSpec, ErrorPathSpec, FilterSpec, IdempotentSpec, MetricsConfig, ObservabilityConfig (+18 more)

### Community 17 - "Community 17"
Cohesion: 0.13
Nodes (25): TestFixtureValidationMissingInput(), TestFixtureValidationMissingName(), TestFixtureValidationNoExpectation(), LoadFixturesFromDirectory(), LoadFixturesFromFile(), TestFixtureWithDroppedExpectation(), TestFixtureWithErrorExpectation(), TestLoadFixturesFromDirectory() (+17 more)

### Community 18 - "Community 18"
Cohesion: 0.18
Nodes (20): JWTValidator, CreateTestToken(), ExtractBearerToken(), ExtractPrincipalFromHeader(), NewJWTValidator(), parseRSAPublicKey(), TestCreateTestToken(), TestExtractBearerToken() (+12 more)

### Community 19 - "Community 19"
Cohesion: 0.15
Nodes (12): isTruthy(), redactFields(), redactInNested(), redactPath(), AuthorizeStep, PDPContext, PDPDecisionRequest, PDPDecisionResponse (+4 more)

### Community 20 - "Community 20"
Cohesion: 0.16
Nodes (8): NewLogBasedCDCSource(), TestDebeziumEventParsing(), TestDebeziumOperationMapping(), TestMaxwellEventParsing(), DebeziumEvent, LogBasedCDCConfig, LogBasedCDCSource, Source

### Community 21 - "Community 21"
Cohesion: 0.19
Nodes (11): makeDecisionRequest(), mockHandlerForAlice(), TestPBACContractConsistency_StubVsMock(), TestPBACWithStubPDP_Fixture(), PBACFixture, StubPDP, NewStubPDP(), TestStubPDPConformance_Allow() (+3 more)

### Community 22 - "Community 22"
Cohesion: 0.19
Nodes (4): KafkaSource, NewKafkaSource(), NewKafkaSourceWithConfig(), SourceConfig

### Community 23 - "Community 23"
Cohesion: 0.31
Nodes (7): NewOPAAdapter(), TestOPAAdapterAllow(), TestOPAAdapterDeny(), TestOPAAdapterMissingResult(), TestOPAAdapterObligations(), TestOPAAdapterRequestTranslation(), OPAAdapter

### Community 24 - "Community 24"
Cohesion: 0.29
Nodes (7): luhn_check(), malloc(), read_string(), transform_phone(), validate_credit_card(), validate_email(), write_string()

### Community 25 - "Community 25"
Cohesion: 0.25
Nodes (4): AMQPSource, NewAMQPSource(), NewAMQPSourceWithConfig(), SourceConfig

### Community 26 - "Community 26"
Cohesion: 0.36
Nodes (9): TestAuthValidationAllRoutesHaveAuth(), TestAuthValidationAuthNonePassesValidation(), TestAuthValidationAuthorizeStepPassesValidation(), TestAuthValidationEmptyRoutes(), TestAuthValidationErrorMessageContainsGuidance(), TestAuthValidationMissingAuthStrictMode(), TestAuthValidationMissingAuthWarnMode(), TestAuthValidationMultipleRoutesMixed() (+1 more)

### Community 27 - "Community 27"
Cohesion: 0.27
Nodes (2): DatabaseSource, SourceConfig

### Community 28 - "Community 28"
Cohesion: 0.36
Nodes (2): TriggerBasedCDCSource, TriggerCDCConfig

### Community 29 - "Community 29"
Cohesion: 0.4
Nodes (4): CallResult, Function, FunctionDef, FunctionType

### Community 30 - "Community 30"
Cohesion: 0.5
Nodes (3): Result, Sink, Source

### Community 32 - "Community 32"
Cohesion: 0.67
Nodes (2): PluginLoader, Registry

### Community 33 - "Community 33"
Cohesion: 1.0
Nodes (1): Principal

### Community 34 - "Community 34"
Cohesion: 1.0
Nodes (2): M2.3.1 Polling Query Source, Database Adapters Design

### Community 35 - "Community 35"
Cohesion: 1.0
Nodes (2): M2.1 Aggregator Step, Phase 2 Track B (M2.1-M2.7)

### Community 36 - "Community 36"
Cohesion: 1.0
Nodes (2): Authorization & Redaction Pattern, M2.5 Obligations Example

### Community 49 - "Community 49"
Cohesion: 1.0
Nodes (1): Fragment Parameterization

### Community 50 - "Community 50"
Cohesion: 1.0
Nodes (1): Parameter Syntax ${PARAM:name}

### Community 51 - "Community 51"
Cohesion: 1.0
Nodes (1): Fragment Defaults and Overrides

### Community 52 - "Community 52"
Cohesion: 1.0
Nodes (1): Composition Semantics - Late Binding

### Community 53 - "Community 53"
Cohesion: 1.0
Nodes (1): M2.3.2 Upsert/Insert Sink

### Community 54 - "Community 54"
Cohesion: 1.0
Nodes (1): M2.3.3 CDC Spike Evaluation

### Community 55 - "Community 55"
Cohesion: 1.0
Nodes (1): M2.3.4 CDC Implementation

### Community 56 - "Community 56"
Cohesion: 1.0
Nodes (1): M2.3.4a Trigger-Based CDC Source

### Community 57 - "Community 57"
Cohesion: 1.0
Nodes (1): M2.3.4b Log-Based CDC via Kafka

### Community 58 - "Community 58"
Cohesion: 1.0
Nodes (1): Watermark-Based Incremental Pull Pattern

### Community 59 - "Community 59"
Cohesion: 1.0
Nodes (1): Static Contract Conformance Checking

### Community 60 - "Community 60"
Cohesion: 1.0
Nodes (1): Analyzable Transform Subset

### Community 61 - "Community 61"
Cohesion: 1.0
Nodes (1): Runtime Enforcement enforce:true

### Community 62 - "Community 62"
Cohesion: 1.0
Nodes (1): Purge-Log Auto-Export on Expiry Warning

### Community 63 - "Community 63"
Cohesion: 1.0
Nodes (1): PurgeEvent with Retention Tracking

### Community 64 - "Community 64"
Cohesion: 1.0
Nodes (1): Expiry Warning Lead Days Mechanism

### Community 65 - "Community 65"
Cohesion: 1.0
Nodes (1): Secrets vs Parameters Distinction

### Community 66 - "Community 66"
Cohesion: 1.0
Nodes (1): Log-Based CDC Architecture Decision

### Community 67 - "Community 67"
Cohesion: 1.0
Nodes (1): Trigger-Based CDC Architecture Decision

### Community 68 - "Community 68"
Cohesion: 1.0
Nodes (1): M2.2 Splitter Step

### Community 69 - "Community 69"
Cohesion: 1.0
Nodes (1): M2.3 Database Adapters (JDBC, CDC)

### Community 70 - "Community 70"
Cohesion: 1.0
Nodes (1): M2.5 Authorization Obligations & Redaction

### Community 71 - "Community 71"
Cohesion: 1.0
Nodes (1): M2.6 Purge-Log Auto-Export to S3

### Community 72 - "Community 72"
Cohesion: 1.0
Nodes (1): M2.7 Static Contract Conformance Checking

### Community 73 - "Community 73"
Cohesion: 1.0
Nodes (1): Phase 0 Production Ready (v0.5.0)

### Community 74 - "Community 74"
Cohesion: 1.0
Nodes (1): Phase 1 Track A (R1-R21)

### Community 75 - "Community 75"
Cohesion: 1.0
Nodes (1): Phase 1 Track B (M1.2-M1.8)

### Community 76 - "Community 76"
Cohesion: 1.0
Nodes (1): dim — Object Knowledge Framework

### Community 77 - "Community 77"
Cohesion: 1.0
Nodes (1): Architectural Decision: Language — Go

### Community 78 - "Community 78"
Cohesion: 1.0
Nodes (1): Architectural Decision: Hot Reload with Generation Takeover

### Community 79 - "Community 79"
Cohesion: 1.0
Nodes (1): Architectural Decision: Per-Instance Embedded Lineage Store

### Community 80 - "Community 80"
Cohesion: 1.0
Nodes (1): Architectural Decision: Structural Authorization Declaration

### Community 81 - "Community 81"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Channels and Backpressure

### Community 82 - "Community 82"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Executor Worker Pool

### Community 83 - "Community 83"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Hot Reload State Machine

### Community 84 - "Community 84"
Cohesion: 1.0
Nodes (1): Obligation Type: Redaction (redact_fields)

### Community 85 - "Community 85"
Cohesion: 1.0
Nodes (1): Obligation Feature: Field-Path Resolution

### Community 86 - "Community 86"
Cohesion: 1.0
Nodes (1): Obligation Feature: Lineage Tracking (Obligation Facets)

### Community 87 - "Community 87"
Cohesion: 1.0
Nodes (1): Community: Authorization & Obligations

### Community 88 - "Community 88"
Cohesion: 1.0
Nodes (1): Community: Aggregator Step

### Community 89 - "Community 89"
Cohesion: 1.0
Nodes (1): Community: Database Adapters

### Community 90 - "Community 90"
Cohesion: 1.0
Nodes (1): CDC Pattern (Log-Based & Trigger-Based)

### Community 91 - "Community 91"
Cohesion: 1.0
Nodes (1): Graph Structure Analysis & Insights

### Community 92 - "Community 92"
Cohesion: 1.0
Nodes (1): Analysis: False Positive Isolated Nodes

### Community 93 - "Community 93"
Cohesion: 1.0
Nodes (1): Phase 2 Completion & Exit Criteria

## Knowledge Gaps
- **168 isolated node(s):** `ConformanceTest`, `ConformanceResult`, `ConformanceSuite`, `Registry`, `PluginLoader` (+163 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 27`** (10 nodes): `DatabaseSource`, `.Close()`, `.GetLastPollTime()`, `.GetWatermark()`, `.HealthCheck()`, `.poll()`, `.SetWatermark()`, `.Start()`, `SourceConfig`, `database_source.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 28`** (8 nodes): `TriggerBasedCDCSource`, `.Close()`, `.GetLastPollTime()`, `.HealthCheck()`, `.pollChangelog()`, `.Start()`, `TriggerCDCConfig`, `cdc_trigger.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (3 nodes): `registry.go`, `PluginLoader`, `Registry`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (2 nodes): `Principal`, `principal.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (2 nodes): `M2.3.1 Polling Query Source`, `Database Adapters Design`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (2 nodes): `M2.1 Aggregator Step`, `Phase 2 Track B (M2.1-M2.7)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (2 nodes): `Authorization & Redaction Pattern`, `M2.5 Obligations Example`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (1 nodes): `Fragment Parameterization`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (1 nodes): `Parameter Syntax ${PARAM:name}`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (1 nodes): `Fragment Defaults and Overrides`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (1 nodes): `Composition Semantics - Late Binding`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 53`** (1 nodes): `M2.3.2 Upsert/Insert Sink`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (1 nodes): `M2.3.3 CDC Spike Evaluation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (1 nodes): `M2.3.4 CDC Implementation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (1 nodes): `M2.3.4a Trigger-Based CDC Source`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 57`** (1 nodes): `M2.3.4b Log-Based CDC via Kafka`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (1 nodes): `Watermark-Based Incremental Pull Pattern`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (1 nodes): `Static Contract Conformance Checking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (1 nodes): `Analyzable Transform Subset`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 61`** (1 nodes): `Runtime Enforcement enforce:true`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 62`** (1 nodes): `Purge-Log Auto-Export on Expiry Warning`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 63`** (1 nodes): `PurgeEvent with Retention Tracking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 64`** (1 nodes): `Expiry Warning Lead Days Mechanism`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (1 nodes): `Secrets vs Parameters Distinction`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (1 nodes): `Log-Based CDC Architecture Decision`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 67`** (1 nodes): `Trigger-Based CDC Architecture Decision`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 68`** (1 nodes): `M2.2 Splitter Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 69`** (1 nodes): `M2.3 Database Adapters (JDBC, CDC)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (1 nodes): `M2.5 Authorization Obligations & Redaction`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 71`** (1 nodes): `M2.6 Purge-Log Auto-Export to S3`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 72`** (1 nodes): `M2.7 Static Contract Conformance Checking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (1 nodes): `Phase 0 Production Ready (v0.5.0)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 74`** (1 nodes): `Phase 1 Track A (R1-R21)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 75`** (1 nodes): `Phase 1 Track B (M1.2-M1.8)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 76`** (1 nodes): `dim — Object Knowledge Framework`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (1 nodes): `Architectural Decision: Language — Go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (1 nodes): `Architectural Decision: Hot Reload with Generation Takeover`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 79`** (1 nodes): `Architectural Decision: Per-Instance Embedded Lineage Store`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 80`** (1 nodes): `Architectural Decision: Structural Authorization Declaration`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 81`** (1 nodes): `Concurrency Pattern: Channels and Backpressure`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 82`** (1 nodes): `Concurrency Pattern: Executor Worker Pool`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (1 nodes): `Concurrency Pattern: Hot Reload State Machine`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (1 nodes): `Obligation Type: Redaction (redact_fields)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 85`** (1 nodes): `Obligation Feature: Field-Path Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 86`** (1 nodes): `Obligation Feature: Lineage Tracking (Obligation Facets)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 87`** (1 nodes): `Community: Authorization & Obligations`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 88`** (1 nodes): `Community: Aggregator Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 89`** (1 nodes): `Community: Database Adapters`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 90`** (1 nodes): `CDC Pattern (Log-Based & Trigger-Based)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 91`** (1 nodes): `Graph Structure Analysis & Insights`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 92`** (1 nodes): `Analysis: False Positive Isolated Nodes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 93`** (1 nodes): `Phase 2 Completion & Exit Criteria`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewMessage()` connect `Community 1` to `Community 0`, `Community 2`, `Community 4`, `Community 5`, `Community 8`, `Community 9`, `Community 10`, `Community 20`, `Community 22`, `Community 25`, `Community 27`, `Community 28`?**
  _High betweenness centrality (0.154) - this node is a cross-community bridge._
- **Why does `LoadRouteConfig()` connect `Community 6` to `Community 0`, `Community 2`, `Community 10`?**
  _High betweenness centrality (0.052) - this node is a cross-community bridge._
- **Why does `TestPhase0FullPipelineWithAllFeatures()` connect `Community 0` to `Community 1`, `Community 2`, `Community 3`, `Community 10`, `Community 18`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Are the 244 inferred relationships involving `NewMessage()` (e.g. with `TestRecordLineage()` and `TestConcurrentWrites()`) actually correct?**
  _`NewMessage()` has 244 INFERRED edges - model-reasoned connections that need verification._
- **Are the 131 inferred relationships involving `NewChannel()` (e.g. with `TestWiretapMessageCopiedToTapSink()` and `TestWiretapOriginalContinuesDownstream()`) actually correct?**
  _`NewChannel()` has 131 INFERRED edges - model-reasoned connections that need verification._
- **Are the 44 inferred relationships involving `NewExecutor()` (e.g. with `TestWiretapIntegrationWithExecutor()` and `TestFileSinkIntegrationEndToEnd()`) actually correct?**
  _`NewExecutor()` has 44 INFERRED edges - model-reasoned connections that need verification._
- **Are the 40 inferred relationships involving `NewTracingProvider()` (e.g. with `TestTracingIntegrationEndToEnd()` and `TestTracingIntegrationErrorHandling()`) actually correct?**
  _`NewTracingProvider()` has 40 INFERRED edges - model-reasoned connections that need verification._