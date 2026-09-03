# Graph Report - dim  (2026-09-03)

## Corpus Check
- 102 files · ~148,974 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 971 nodes · 2975 edges · 28 communities detected
- Extraction: 34% EXTRACTED · 66% INFERRED · 0% AMBIGUOUS · INFERRED: 1952 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]

## God Nodes (most connected - your core abstractions)
1. `NewMessage()` - 162 edges
2. `NewChannel()` - 106 edges
3. `NewExecutor()` - 39 edges
4. `NewTracingProvider()` - 38 edges
5. `NewContractStore()` - 34 edges
6. `NewStore()` - 29 edges
7. `BuildStepsFromSpec()` - 27 edges
8. `LoadRouteConfig()` - 27 edges
9. `NewRouteStep()` - 26 edges
10. `NewAuthorizeStep()` - 25 edges

## Surprising Connections (you probably didn't know these)
- `BenchmarkContractValidation()` --calls--> `NewContractStep()`  [INFERRED]
  examples/bench/throughput_test.go → internal/steps/contract.go
- `BenchmarkMessageThroughput()` --calls--> `NewTranslateStep()`  [INFERRED]
  examples/bench/throughput_test.go → internal/steps/translate.go
- `BenchmarkMessageThroughput()` --calls--> `NewFilterStep()`  [INFERRED]
  examples/bench/throughput_test.go → internal/steps/filter.go
- `BenchmarkMessageThroughput()` --calls--> `NewAuthorizeStep()`  [INFERRED]
  examples/bench/throughput_test.go → internal/steps/authorize.go
- `buildExecutorForRoute()` --calls--> `NewContractStore()`  [INFERRED]
  cmd/midctl/main.go → internal/config/contracts.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.05
Nodes (125): NewAuthorizeStep(), TestAuthorizeStepABACBodyContextAvailable(), TestAuthorizeStepABACBodyContextFail(), TestAuthorizeStepABACComplexExpression(), TestAuthorizeStepABACExpressionFalse(), TestAuthorizeStepABACExpressionTrue(), TestAuthorizeStepABACTruthyFalsy(), TestAuthorizeStepABACWithSubject() (+117 more)

### Community 1 - "Community 1"
Cohesion: 0.05
Nodes (106): CalculateBackoff(), NewChannel(), TestChannelBackpressure(), TestChannelConcurrentSendRecv(), TestChannelContextCancellation(), TestChannelInFlightTracking(), TestChannelNilMessage(), TestChannelSendRecv() (+98 more)

### Community 2 - "Community 2"
Cohesion: 0.06
Nodes (65): Principal, Span, SpanEvent, SpanStreamer, stepError, TailOptions, TracingProvider, FindChildSpans() (+57 more)

### Community 3 - "Community 3"
Cohesion: 0.04
Nodes (67): DefaultConfig(), FromRouteConfig(), BuildStepsFromSpec(), HTTPSourceAdapter, MessageRouter, SinkAdapter, SourceAdapter, TestBuildStepsFromSpecAuthorizeStepABAC() (+59 more)

### Community 4 - "Community 4"
Cohesion: 0.06
Nodes (39): Export(), exportCSV(), exportNDJSON(), TestExportCSV(), TestExportEmptyRecords(), TestExportInvalidFormat(), TestExportNDJSON(), LineageRecord (+31 more)

### Community 5 - "Community 5"
Cohesion: 0.09
Nodes (47): ContractStore, GetContractViolation(), IsContractViolation(), NewContractStep(), setupContractStoreWithPaymentSchema(), setupContractStoreWithStrictContract(), TestContractStepComplexSchema(), TestContractStepConformingMessage() (+39 more)

### Community 6 - "Community 6"
Cohesion: 0.06
Nodes (53): TestAuthValidationAllRoutesHaveAuth(), TestAuthValidationAuthNonePassesValidation(), TestAuthValidationAuthorizeStepPassesValidation(), TestAuthValidationEmptyRoutes(), TestAuthValidationErrorMessageContainsGuidance(), TestAuthValidationMissingAuthStrictMode(), TestAuthValidationMissingAuthWarnMode(), TestAuthValidationMultipleRoutesMixed() (+45 more)

### Community 7 - "Community 7"
Cohesion: 0.05
Nodes (30): isTruthy(), Message, Metadata, Principal, deepEqual(), TestJSONataLibrarySpike(), RetentionPolicy, RetentionResolver (+22 more)

### Community 8 - "Community 8"
Cohesion: 0.11
Nodes (33): FileSinkAdapter, FileSink, PassthroughStep, TestFileSinkBackpressure(), TestFileSinkIntegrationEndToEnd(), TestFileSinkWithComplexMessageStructure(), NewFileSink(), TestFileSinkAppendMode() (+25 more)

### Community 9 - "Community 9"
Cohesion: 0.15
Nodes (25): TestViewerIntegrationFullFlow(), TestViewerIntegrationHighLoad(), TestViewerIntegrationMultipleRoutes(), TestViewerIntegrationTimestamp(), Message, NewRingBuffer(), NewRouteViewer(), NewViewerServer() (+17 more)

### Community 10 - "Community 10"
Cohesion: 0.13
Nodes (27): FragmentResolver, hasImport(), NewFragmentResolver(), TestConfigMergingOverride(), TestConfigMergingSourcesAndSinks(), TestConfigMergingSteps(), TestCycleDetection(), TestFragmentEquivalenceRouteVersion() (+19 more)

### Community 11 - "Community 11"
Cohesion: 0.16
Nodes (22): getPercentile(), NewMetricsCollector(), BenchmarkMetricsCollector_GetMetrics(), BenchmarkMetricsCollector_RecordMessageSuccess(), TestMetricsCollector_Disable(), TestMetricsCollector_GetMetrics_PrometheusFormat(), TestMetricsCollector_LatencyAverage(), TestMetricsCollector_LatencyPercentiles() (+14 more)

### Community 12 - "Community 12"
Cohesion: 0.12
Nodes (23): JWTValidator, HTTPSource, convertAttributesToClaims(), generateHTTPCorrelationID(), CreateTestToken(), ExtractBearerToken(), ExtractPrincipalFromHeader(), NewJWTValidator() (+15 more)

### Community 13 - "Community 13"
Cohesion: 0.13
Nodes (25): TestFixtureValidationMissingInput(), TestFixtureValidationMissingName(), TestFixtureValidationNoExpectation(), LoadFixturesFromDirectory(), LoadFixturesFromFile(), TestFixtureWithDroppedExpectation(), TestFixtureWithErrorExpectation(), TestLoadFixturesFromDirectory() (+17 more)

### Community 14 - "Community 14"
Cohesion: 0.1
Nodes (19): AuthorizeSpec, ContractSpec, ContractStepSpec, ErrorPathSpec, FilterSpec, IdempotentSpec, MetricsConfig, ObservabilityConfig (+11 more)

### Community 15 - "Community 15"
Cohesion: 0.24
Nodes (2): DeadLetterEnvelope, TestPrincipalWithoutAttributes()

### Community 16 - "Community 16"
Cohesion: 1.0
Nodes (1): Principal

### Community 29 - "Community 29"
Cohesion: 1.0
Nodes (1): Data Contracts

### Community 30 - "Community 30"
Cohesion: 1.0
Nodes (1): Declarative Integration Middleware

### Community 31 - "Community 31"
Cohesion: 1.0
Nodes (1): Route (Message Pipeline)

### Community 32 - "Community 32"
Cohesion: 1.0
Nodes (1): M0.1 Walking Skeleton

### Community 33 - "Community 33"
Cohesion: 1.0
Nodes (1): M0.2 Reliability

### Community 34 - "Community 34"
Cohesion: 1.0
Nodes (1): Authorize Step

### Community 35 - "Community 35"
Cohesion: 1.0
Nodes (1): Filter Step

### Community 36 - "Community 36"
Cohesion: 1.0
Nodes (1): Route Step

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (1): Translate Step

### Community 38 - "Community 38"
Cohesion: 1.0
Nodes (1): EIP Step Types

### Community 39 - "Community 39"
Cohesion: 1.0
Nodes (1): Hot Reload

## Knowledge Gaps
- **56 isolated node(s):** `PurgeOpts`, `LineageRecord`, `PurgeEvent`, `writeRequest`, `ProvenanceChain` (+51 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 15`** (7 nodes): `DeadLetterEnvelope`, `.MarshalJSON()`, `.UnmarshalJSON()`, `principal_test.go`, `TestPrincipalCreation()`, `TestPrincipalWithoutAttributes()`, `TestPrincipalWithoutRoles()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 16`** (2 nodes): `Principal`, `principal.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 29`** (1 nodes): `Data Contracts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 30`** (1 nodes): `Declarative Integration Middleware`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (1 nodes): `Route (Message Pipeline)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (1 nodes): `M0.1 Walking Skeleton`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (1 nodes): `M0.2 Reliability`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (1 nodes): `Authorize Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (1 nodes): `Filter Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (1 nodes): `Route Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (1 nodes): `Translate Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (1 nodes): `EIP Step Types`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (1 nodes): `Hot Reload`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewMessage()` connect `Community 0` to `Community 1`, `Community 4`, `Community 5`, `Community 7`, `Community 8`?**
  _High betweenness centrality (0.112) - this node is a cross-community bridge._
- **Why does `runMultiRoute()` connect `Community 3` to `Community 0`, `Community 1`, `Community 2`, `Community 6`, `Community 8`, `Community 11`?**
  _High betweenness centrality (0.110) - this node is a cross-community bridge._
- **Why does `LoadRouteConfig()` connect `Community 6` to `Community 10`, `Community 3`?**
  _High betweenness centrality (0.067) - this node is a cross-community bridge._
- **Are the 160 inferred relationships involving `NewMessage()` (e.g. with `TestRecordLineage()` and `TestConcurrentWrites()`) actually correct?**
  _`NewMessage()` has 160 INFERRED edges - model-reasoned connections that need verification._
- **Are the 105 inferred relationships involving `NewChannel()` (e.g. with `TestWiretapMessageCopiedToTapSink()` and `TestWiretapOriginalContinuesDownstream()`) actually correct?**
  _`NewChannel()` has 105 INFERRED edges - model-reasoned connections that need verification._
- **Are the 37 inferred relationships involving `NewExecutor()` (e.g. with `TestWiretapIntegrationWithExecutor()` and `TestFileSinkIntegrationEndToEnd()`) actually correct?**
  _`NewExecutor()` has 37 INFERRED edges - model-reasoned connections that need verification._
- **Are the 37 inferred relationships involving `NewTracingProvider()` (e.g. with `TestTracingIntegrationEndToEnd()` and `TestTracingIntegrationErrorHandling()`) actually correct?**
  _`NewTracingProvider()` has 37 INFERRED edges - model-reasoned connections that need verification._