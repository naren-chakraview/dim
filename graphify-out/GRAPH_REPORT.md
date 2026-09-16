# Graph Report - dim  (2026-09-14)

## Corpus Check
- 254 files · ~438,996 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 7004 nodes · 21062 edges · 89 communities detected
- Extraction: 76% EXTRACTED · 24% INFERRED · 0% AMBIGUOUS · INFERRED: 4978 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
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

## God Nodes (most connected - your core abstractions)
1. `NewMessage()` - 246 edges
2. `NewChannel()` - 132 edges
3. `contains()` - 83 edges
4. `H()` - 71 edges
5. `Vt()` - 70 edges
6. `mt()` - 59 edges
7. `H()` - 56 edges
8. `H()` - 56 edges
9. `v()` - 50 edges
10. `v()` - 50 edges

## Surprising Connections (you probably didn't know these)
- `TestPluginConformance()` --calls--> `SummarizeResults()`  [INFERRED]
  pkg/sdk/conformance_test.go → internal/testing/fixture.go
- `BuildStepsFromSpec()` --calls--> `buildExecutorForRoute()`  [INFERRED]
  internal/steps/factory.go → cmd/dimctl/main.go
- `TestResolveInRouteSameDomain()` --calls--> `contains()`  [INFERRED]
  internal/secrets/resolver_test.go → cmd/dimd/e2e_test.go
- `TestResolveInRouteGlobalFallback()` --calls--> `contains()`  [INFERRED]
  internal/secrets/resolver_test.go → cmd/dimd/e2e_test.go
- `TestResolveInRouteExplicitCrossDomain()` --calls--> `contains()`  [INFERRED]
  internal/secrets/resolver_test.go → cmd/dimd/e2e_test.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.01
Nodes (577): TestAggregatorIntegration(), TestAggregatorMultipleOrdersIntegration(), TestAggregatorTimeoutCompletion(), TestAggregatorWithComplexMessages(), NewAggregateStep(), TestAggregateCountCompletion(), TestAggregateDrain(), TestAggregateMultipleCorrelationKeys() (+569 more)

### Community 1 - "Community 1"
Cohesion: 0.01
Nodes (500): _1(), kx(), M0(), Nx(), Rd(), sS(), ai(), eg() (+492 more)

### Community 2 - "Community 2"
Cohesion: 0.01
Nodes (475): iv(), lm(), Vt(), _0(), _a(), a0(), aa(), Ac() (+467 more)

### Community 3 - "Community 3"
Cohesion: 0.01
Nodes (470): _0(), _a(), a0(), aa(), Ac(), Ae(), af(), ah() (+462 more)

### Community 4 - "Community 4"
Cohesion: 0.01
Nodes (468): $d(), De(), ld(), Nr(), _(), $0(), _a(), A0() (+460 more)

### Community 5 - "Community 5"
Cohesion: 0.01
Nodes (466): ht(), On(), rm(), sv(), Vc(), mg(), mg(), om() (+458 more)

### Community 6 - "Community 6"
Cohesion: 0.01
Nodes (466): _0(), _a(), A0(), a1(), aa(), Ac(), Ae(), af() (+458 more)

### Community 7 - "Community 7"
Cohesion: 0.01
Nodes (464): nd(), sg(), _(), $0(), _a(), A0(), a1(), aa() (+456 more)

### Community 8 - "Community 8"
Cohesion: 0.01
Nodes (465): Nk(), im(), Mh(), Ov(), Ri(), _0(), _a(), A0() (+457 more)

### Community 9 - "Community 9"
Cohesion: 0.01
Nodes (461): mt(), _0(), _a(), A0(), a1(), aa(), Ac(), Ae() (+453 more)

### Community 10 - "Community 10"
Cohesion: 0.03
Nodes (103): NewAlternativeRegistry(), NewApicurioClient(), TestApicurioDefaultGroup(), TestApicurioHealth(), TestApicurioRegisterAndFetch(), runSearch(), NewApicurioClient(), NewOpenLineageClient() (+95 more)

### Community 11 - "Community 11"
Cohesion: 0.04
Nodes (88): isTruthy(), redactFields(), redactInNested(), redactPath(), contains(), findSubstringIndex(), TestScenario_AuditTrailForCrossDomain(), TestScenario_DomainIsolation() (+80 more)

### Community 12 - "Community 12"
Cohesion: 0.04
Nodes (85): getPercentile(), NewMetricsCollector(), BenchmarkMetricsCollector_GetMetrics(), BenchmarkMetricsCollector_RecordMessageSuccess(), TestMetricsCollector_Disable(), TestMetricsCollector_GetMetrics_PrometheusFormat(), TestMetricsCollector_LatencyAverage(), TestMetricsCollector_LatencyPercentiles() (+77 more)

### Community 13 - "Community 13"
Cohesion: 0.03
Nodes (95): CritiqueFinding, CritiquePattern, CritiqueRouteRequest, CritiqueRouteResponse, IntentSignals, MCPOperation, MCPServer, OperationSchema (+87 more)

### Community 14 - "Community 14"
Cohesion: 0.03
Nodes (63): AMQPSink, TestClaimCheckCleanup(), TestClaimCheckMultipleAttachments(), TestClaimCheckOrderWithAttachmentWorkflow(), TestClaimCheckVerificationFailure(), marshalPayload(), NewClaimCheckStep(), ResolveClaimCheck() (+55 more)

### Community 15 - "Community 15"
Cohesion: 0.04
Nodes (86): TestAuthValidationAllRoutesHaveAuth(), TestAuthValidationAuthNonePassesValidation(), TestAuthValidationAuthorizeStepPassesValidation(), TestAuthValidationEmptyRoutes(), TestAuthValidationErrorMessageContainsGuidance(), TestAuthValidationMissingAuthStrictMode(), TestAuthValidationMissingAuthWarnMode(), TestAuthValidationMultipleRoutesMixed() (+78 more)

### Community 16 - "Community 16"
Cohesion: 0.04
Nodes (55): Cluster, ClusterConfig, CoordinatedGenerationTracker, DedupStore, GenerationTracker, InMemoryDedupStore, Instance, LineageBackend (+47 more)

### Community 17 - "Community 17"
Cohesion: 0.07
Nodes (36): NewManager(), parseTenanConfig(), TestManagerDefaultTenant(), TestManagerMultipleTenants(), TestManagerRegisterTenant(), TestParseTenanConfig(), TestUsageUnlimitedQuota(), TestUsageUtilization() (+28 more)

### Community 18 - "Community 18"
Cohesion: 0.14
Nodes (25): TestViewerIntegrationFullFlow(), TestViewerIntegrationHighLoad(), TestViewerIntegrationMultipleRoutes(), TestViewerIntegrationTimestamp(), Message, NewRingBuffer(), NewRouteViewer(), NewViewerServer() (+17 more)

### Community 19 - "Community 19"
Cohesion: 0.07
Nodes (28): AuthorizeSpec, ClaimCheckSpec, ClaimResolveSpec, ContractSpec, ContractStepSpec, ErrorPathSpec, FilterSpec, IdempotentSpec (+20 more)

### Community 20 - "Community 20"
Cohesion: 0.12
Nodes (23): JWTValidator, HTTPSource, convertAttributesToClaims(), generateHTTPCorrelationID(), CreateTestToken(), ExtractBearerToken(), ExtractPrincipalFromHeader(), NewJWTValidator() (+15 more)

### Community 21 - "Community 21"
Cohesion: 0.11
Nodes (16): AgentMCPClient, main(), OperationError, Simulate test_route operation, Simulate scaffold_domain operation, Parse MCP response into ResponseEnvelope, Agent for validating and testing routes, Validate a route configuration.          Returns True if valid, False otherwise. (+8 more)

### Community 22 - "Community 22"
Cohesion: 0.12
Nodes (13): DatabaseSink, SinkConfig, Dataset, Job, OpenLineageEvent, Run, SchemaDatasetFacet, SchemaField (+5 more)

### Community 23 - "Community 23"
Cohesion: 0.16
Nodes (21): GetOrderingMode(), GetWorkerCount(), OrderingMode, ParseOrdering(), slowMockStep, BenchmarkOrdering(), TestBackwardCompatibility(), TestConfigParsing() (+13 more)

### Community 24 - "Community 24"
Cohesion: 0.1
Nodes (20): CapabilitiesRequest, CapabilitiesResponse, Capability, LineageRecord, LineageRequest, LineageResponse, OperationErr, ProvenanceRecord (+12 more)

### Community 25 - "Community 25"
Cohesion: 0.15
Nodes (13): DefaultConformanceSuite(), TestConformanceSuiteStructure(), TestConformanceTimeout(), TestDeterminismCheck(), TestErrorHandling(), TestMultipleArguments(), TestPluginConformance(), TestPlugin() (+5 more)

### Community 26 - "Community 26"
Cohesion: 0.33
Nodes (7): NewOPAAdapter(), TestOPAAdapterAllow(), TestOPAAdapterDeny(), TestOPAAdapterMissingResult(), TestOPAAdapterObligations(), TestOPAAdapterRequestTranslation(), OPAAdapter

### Community 27 - "Community 27"
Cohesion: 0.29
Nodes (7): luhn_check(), malloc(), read_string(), transform_phone(), validate_credit_card(), validate_email(), write_string()

### Community 28 - "Community 28"
Cohesion: 0.33
Nodes (3): sd(), sd(), JSONataEditor

### Community 29 - "Community 29"
Cohesion: 0.32
Nodes (1): KafkaSink

### Community 30 - "Community 30"
Cohesion: 0.25
Nodes (2): S3Sink, SinkConfig

### Community 31 - "Community 31"
Cohesion: 0.32
Nodes (1): RouteCanvas

### Community 33 - "Community 33"
Cohesion: 0.38
Nodes (4): containsCommentChar(), countComments(), TestCommentPreservationIssue(), TestYAMLv3NodeStructure()

### Community 34 - "Community 34"
Cohesion: 0.33
Nodes (5): ApicurioArtifact, ApicurioResponse, CatalogResult, OpenLineageDataset, SearchResults

### Community 35 - "Community 35"
Cohesion: 0.4
Nodes (1): FormGenerator

### Community 36 - "Community 36"
Cohesion: 0.4
Nodes (4): CallResult, Function, FunctionDef, FunctionType

### Community 37 - "Community 37"
Cohesion: 0.5
Nodes (3): Result, Sink, Source

### Community 38 - "Community 38"
Cohesion: 0.5
Nodes (1): TestPrincipalWithoutAttributes()

### Community 39 - "Community 39"
Cohesion: 0.5
Nodes (3): Placeholder, ScaffoldFromIntentRequest, ScaffoldFromIntentResponse

### Community 41 - "Community 41"
Cohesion: 0.67
Nodes (2): PluginLoader, Registry

### Community 43 - "Community 43"
Cohesion: 1.0
Nodes (1): Principal

### Community 46 - "Community 46"
Cohesion: 1.0
Nodes (2): M2.3.1 Polling Query Source, Database Adapters Design

### Community 47 - "Community 47"
Cohesion: 1.0
Nodes (2): M2.1 Aggregator Step, Phase 2 Track B (M2.1-M2.7)

### Community 48 - "Community 48"
Cohesion: 1.0
Nodes (2): Authorization & Redaction Pattern, M2.5 Obligations Example

### Community 75 - "Community 75"
Cohesion: 1.0
Nodes (1): Fragment Parameterization

### Community 76 - "Community 76"
Cohesion: 1.0
Nodes (1): Parameter Syntax ${PARAM:name}

### Community 77 - "Community 77"
Cohesion: 1.0
Nodes (1): Fragment Defaults and Overrides

### Community 78 - "Community 78"
Cohesion: 1.0
Nodes (1): Composition Semantics - Late Binding

### Community 79 - "Community 79"
Cohesion: 1.0
Nodes (1): M2.3.2 Upsert/Insert Sink

### Community 80 - "Community 80"
Cohesion: 1.0
Nodes (1): M2.3.3 CDC Spike Evaluation

### Community 81 - "Community 81"
Cohesion: 1.0
Nodes (1): M2.3.4 CDC Implementation

### Community 82 - "Community 82"
Cohesion: 1.0
Nodes (1): M2.3.4a Trigger-Based CDC Source

### Community 83 - "Community 83"
Cohesion: 1.0
Nodes (1): M2.3.4b Log-Based CDC via Kafka

### Community 84 - "Community 84"
Cohesion: 1.0
Nodes (1): Watermark-Based Incremental Pull Pattern

### Community 85 - "Community 85"
Cohesion: 1.0
Nodes (1): Static Contract Conformance Checking

### Community 86 - "Community 86"
Cohesion: 1.0
Nodes (1): Analyzable Transform Subset

### Community 87 - "Community 87"
Cohesion: 1.0
Nodes (1): Runtime Enforcement enforce:true

### Community 88 - "Community 88"
Cohesion: 1.0
Nodes (1): Purge-Log Auto-Export on Expiry Warning

### Community 89 - "Community 89"
Cohesion: 1.0
Nodes (1): PurgeEvent with Retention Tracking

### Community 90 - "Community 90"
Cohesion: 1.0
Nodes (1): Expiry Warning Lead Days Mechanism

### Community 91 - "Community 91"
Cohesion: 1.0
Nodes (1): Secrets vs Parameters Distinction

### Community 92 - "Community 92"
Cohesion: 1.0
Nodes (1): Log-Based CDC Architecture Decision

### Community 93 - "Community 93"
Cohesion: 1.0
Nodes (1): Trigger-Based CDC Architecture Decision

### Community 94 - "Community 94"
Cohesion: 1.0
Nodes (1): M2.2 Splitter Step

### Community 95 - "Community 95"
Cohesion: 1.0
Nodes (1): M2.3 Database Adapters (JDBC, CDC)

### Community 96 - "Community 96"
Cohesion: 1.0
Nodes (1): M2.5 Authorization Obligations & Redaction

### Community 97 - "Community 97"
Cohesion: 1.0
Nodes (1): M2.6 Purge-Log Auto-Export to S3

### Community 98 - "Community 98"
Cohesion: 1.0
Nodes (1): M2.7 Static Contract Conformance Checking

### Community 99 - "Community 99"
Cohesion: 1.0
Nodes (1): Phase 0 Production Ready (v0.5.0)

### Community 100 - "Community 100"
Cohesion: 1.0
Nodes (1): Phase 1 Track A (R1-R21)

### Community 101 - "Community 101"
Cohesion: 1.0
Nodes (1): Phase 1 Track B (M1.2-M1.8)

### Community 102 - "Community 102"
Cohesion: 1.0
Nodes (1): dim — Object Knowledge Framework

### Community 103 - "Community 103"
Cohesion: 1.0
Nodes (1): Architectural Decision: Language — Go

### Community 104 - "Community 104"
Cohesion: 1.0
Nodes (1): Architectural Decision: Hot Reload with Generation Takeover

### Community 105 - "Community 105"
Cohesion: 1.0
Nodes (1): Architectural Decision: Per-Instance Embedded Lineage Store

### Community 106 - "Community 106"
Cohesion: 1.0
Nodes (1): Architectural Decision: Structural Authorization Declaration

### Community 107 - "Community 107"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Channels and Backpressure

### Community 108 - "Community 108"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Executor Worker Pool

### Community 109 - "Community 109"
Cohesion: 1.0
Nodes (1): Concurrency Pattern: Hot Reload State Machine

### Community 110 - "Community 110"
Cohesion: 1.0
Nodes (1): Obligation Type: Redaction (redact_fields)

### Community 111 - "Community 111"
Cohesion: 1.0
Nodes (1): Obligation Feature: Field-Path Resolution

### Community 112 - "Community 112"
Cohesion: 1.0
Nodes (1): Obligation Feature: Lineage Tracking (Obligation Facets)

### Community 113 - "Community 113"
Cohesion: 1.0
Nodes (1): Community: Authorization & Obligations

### Community 114 - "Community 114"
Cohesion: 1.0
Nodes (1): Community: Aggregator Step

### Community 115 - "Community 115"
Cohesion: 1.0
Nodes (1): Community: Database Adapters

### Community 116 - "Community 116"
Cohesion: 1.0
Nodes (1): CDC Pattern (Log-Based & Trigger-Based)

### Community 117 - "Community 117"
Cohesion: 1.0
Nodes (1): Graph Structure Analysis & Insights

### Community 118 - "Community 118"
Cohesion: 1.0
Nodes (1): Analysis: False Positive Isolated Nodes

### Community 119 - "Community 119"
Cohesion: 1.0
Nodes (1): Phase 2 Completion & Exit Criteria

## Knowledge Gaps
- **218 isolated node(s):** `ConformanceTest`, `ConformanceResult`, `ConformanceSuite`, `Registry`, `PluginLoader` (+213 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 29`** (8 nodes): `KafkaSink`, `.Close()`, `.dimMessageToKafkaMessage()`, `.GetMetrics()`, `.HealthCheck()`, `.recordError()`, `.Start()`, `.Write()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 30`** (8 nodes): `s3_sink.go`, `S3Sink`, `.Close()`, `.HealthCheck()`, `.Start()`, `.Write()`, `NewS3Sink()`, `SinkConfig`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (8 nodes): `RouteCanvas`, `.constructor()`, `.createNode()`, `.init()`, `.onNodeSelected()`, `.onZoom()`, `.render()`, `canvas.js`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (6 nodes): `forms.js`, `FormGenerator`, `.constructor()`, `.generateForm()`, `.load()`, `.onSave()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (4 nodes): `principal_test.go`, `TestPrincipalCreation()`, `TestPrincipalWithoutAttributes()`, `TestPrincipalWithoutRoles()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (3 nodes): `registry.go`, `PluginLoader`, `Registry`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (2 nodes): `Principal`, `principal.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (2 nodes): `M2.3.1 Polling Query Source`, `Database Adapters Design`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (2 nodes): `M2.1 Aggregator Step`, `Phase 2 Track B (M2.1-M2.7)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (2 nodes): `Authorization & Redaction Pattern`, `M2.5 Obligations Example`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 75`** (1 nodes): `Fragment Parameterization`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 76`** (1 nodes): `Parameter Syntax ${PARAM:name}`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (1 nodes): `Fragment Defaults and Overrides`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (1 nodes): `Composition Semantics - Late Binding`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 79`** (1 nodes): `M2.3.2 Upsert/Insert Sink`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 80`** (1 nodes): `M2.3.3 CDC Spike Evaluation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 81`** (1 nodes): `M2.3.4 CDC Implementation`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 82`** (1 nodes): `M2.3.4a Trigger-Based CDC Source`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (1 nodes): `M2.3.4b Log-Based CDC via Kafka`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (1 nodes): `Watermark-Based Incremental Pull Pattern`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 85`** (1 nodes): `Static Contract Conformance Checking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 86`** (1 nodes): `Analyzable Transform Subset`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 87`** (1 nodes): `Runtime Enforcement enforce:true`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 88`** (1 nodes): `Purge-Log Auto-Export on Expiry Warning`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 89`** (1 nodes): `PurgeEvent with Retention Tracking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 90`** (1 nodes): `Expiry Warning Lead Days Mechanism`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 91`** (1 nodes): `Secrets vs Parameters Distinction`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 92`** (1 nodes): `Log-Based CDC Architecture Decision`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 93`** (1 nodes): `Trigger-Based CDC Architecture Decision`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 94`** (1 nodes): `M2.2 Splitter Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 95`** (1 nodes): `M2.3 Database Adapters (JDBC, CDC)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 96`** (1 nodes): `M2.5 Authorization Obligations & Redaction`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 97`** (1 nodes): `M2.6 Purge-Log Auto-Export to S3`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 98`** (1 nodes): `M2.7 Static Contract Conformance Checking`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 99`** (1 nodes): `Phase 0 Production Ready (v0.5.0)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 100`** (1 nodes): `Phase 1 Track A (R1-R21)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 101`** (1 nodes): `Phase 1 Track B (M1.2-M1.8)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 102`** (1 nodes): `dim — Object Knowledge Framework`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 103`** (1 nodes): `Architectural Decision: Language — Go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 104`** (1 nodes): `Architectural Decision: Hot Reload with Generation Takeover`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 105`** (1 nodes): `Architectural Decision: Per-Instance Embedded Lineage Store`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 106`** (1 nodes): `Architectural Decision: Structural Authorization Declaration`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 107`** (1 nodes): `Concurrency Pattern: Channels and Backpressure`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 108`** (1 nodes): `Concurrency Pattern: Executor Worker Pool`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 109`** (1 nodes): `Concurrency Pattern: Hot Reload State Machine`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 110`** (1 nodes): `Obligation Type: Redaction (redact_fields)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 111`** (1 nodes): `Obligation Feature: Field-Path Resolution`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 112`** (1 nodes): `Obligation Feature: Lineage Tracking (Obligation Facets)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 113`** (1 nodes): `Community: Authorization & Obligations`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 114`** (1 nodes): `Community: Aggregator Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 115`** (1 nodes): `Community: Database Adapters`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 116`** (1 nodes): `CDC Pattern (Log-Based & Trigger-Based)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 117`** (1 nodes): `Graph Structure Analysis & Insights`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 118`** (1 nodes): `Analysis: False Positive Isolated Nodes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 119`** (1 nodes): `Phase 2 Completion & Exit Criteria`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `LoadRouteConfig()` connect `Community 15` to `Community 0`, `Community 13`?**
  _High betweenness centrality (0.042) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 0` to `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 11`, `Community 12`, `Community 13`, `Community 14`, `Community 15`, `Community 16`, `Community 27`?**
  _High betweenness centrality (0.028) - this node is a cross-community bridge._
- **Why does `NewMessage()` connect `Community 0` to `Community 10`, `Community 11`, `Community 14`?**
  _High betweenness centrality (0.026) - this node is a cross-community bridge._
- **Are the 244 inferred relationships involving `NewMessage()` (e.g. with `TestRecordLineage()` and `TestConcurrentWrites()`) actually correct?**
  _`NewMessage()` has 244 INFERRED edges - model-reasoned connections that need verification._
- **Are the 131 inferred relationships involving `NewChannel()` (e.g. with `TestWiretapMessageCopiedToTapSink()` and `TestWiretapOriginalContinuesDownstream()`) actually correct?**
  _`NewChannel()` has 131 INFERRED edges - model-reasoned connections that need verification._
- **Are the 82 inferred relationships involving `contains()` (e.g. with `TestExportInvalidFormat()` and `TestAuthorizeStepNewAuthorizeStepInvalidMode()`) actually correct?**
  _`contains()` has 82 INFERRED edges - model-reasoned connections that need verification._
- **Are the 17 inferred relationships involving `H()` (e.g. with `n()` and `b()`) actually correct?**
  _`H()` has 17 INFERRED edges - model-reasoned connections that need verification._