# Graph Report - dim  (2026-09-05)

## Corpus Check
- 164 files · ~246,870 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1506 nodes · 4396 edges · 34 communities detected
- Extraction: 37% EXTRACTED · 63% INFERRED · 0% AMBIGUOUS · INFERRED: 2788 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 33|Community 33]]

## God Nodes (most connected - your core abstractions)
1. `NewMessage()` - 237 edges
2. `NewChannel()` - 132 edges
3. `NewExecutor()` - 46 edges
4. `NewTracingProvider()` - 41 edges
5. `NewContractStore()` - 37 edges
6. `NewAuthorizeStep()` - 34 edges
7. `NewStore()` - 33 edges
8. `LoadRouteConfig()` - 32 edges
9. `NewRouteStep()` - 27 edges
10. `BuildStepsFromSpec()` - 26 edges

## Surprising Connections (you probably didn't know these)
- `NewContractStep()` --calls--> `BenchmarkContractValidation()`  [INFERRED]
  internal/steps/contract.go → examples/bench/throughput_test.go
- `NewTranslateStep()` --calls--> `BenchmarkMessageThroughput()`  [INFERRED]
  internal/steps/translate.go → examples/bench/throughput_test.go
- `NewFilterStep()` --calls--> `BenchmarkMessageThroughput()`  [INFERRED]
  internal/steps/filter.go → examples/bench/throughput_test.go
- `NewAuthorizeStep()` --calls--> `BenchmarkMessageThroughput()`  [INFERRED]
  internal/steps/authorize.go → examples/bench/throughput_test.go
- `LoadRouteConfig()` --calls--> `runDaemon()`  [INFERRED]
  internal/config/loader.go → cmd/dimd/main.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.03
Nodes (168): TestAMQPSourceClose(), isVersionListResponse(), CalculateBackoff(), TestLogBasedCDCConfigValidation(), TestTriggerBasedCDCSourceCreation(), TestTriggerCDCConfigValidation(), NewTriggerBasedCDCSource(), NewChannel() (+160 more)

### Community 1 - "Community 1"
Cohesion: 0.04
Nodes (192): TestAggregatorIntegration(), TestAggregatorMultipleOrdersIntegration(), TestAggregatorTimeoutCompletion(), TestAggregatorWithComplexMessages(), NewAggregateStep(), TestAggregateCountCompletion(), TestAggregateDrain(), TestAggregateMultipleCorrelationKeys() (+184 more)

### Community 2 - "Community 2"
Cohesion: 0.04
Nodes (93): ContractStore, SchemaDatasetFacet, SchemaField, setupContractStoreWithStrictContract(), computeInlineContractVersion(), computeRegistryContractVersion(), TestR18InlineContractVersionComputation(), TestR18RegistryBackedContractVersionComputation() (+85 more)

### Community 3 - "Community 3"
Cohesion: 0.04
Nodes (54): TestAMQPDrainTimeout(), TestAMQPHotReloadGraceful(), TestAMQPInFlightTracking(), TestAMQPNoDuplicatesOnReload(), TestAMQPReliabilityAtLeastOnce(), Export(), exportCSV(), exportNDJSON() (+46 more)

### Community 4 - "Community 4"
Cohesion: 0.06
Nodes (64): Principal, Span, SpanEvent, SpanStreamer, stepError, TailOptions, TracingProvider, FindChildSpans() (+56 more)

### Community 5 - "Community 5"
Cohesion: 0.05
Nodes (67): TestAuthValidationAllRoutesHaveAuth(), TestAuthValidationAuthNonePassesValidation(), TestAuthValidationAuthorizeStepPassesValidation(), TestAuthValidationEmptyRoutes(), TestAuthValidationErrorMessageContainsGuidance(), TestAuthValidationMissingAuthStrictMode(), TestAuthValidationMissingAuthWarnMode(), TestAuthValidationMultipleRoutesMixed() (+59 more)

### Community 6 - "Community 6"
Cohesion: 0.05
Nodes (33): AMQPSink, BenchmarkAMQPMessageConversion(), TestAMQPSinkMetrics(), TestAMQPSourceBasic(), NewAMQPSink(), NewAMQPSinkWithConfig(), SinkConfig, FileSink (+25 more)

### Community 7 - "Community 7"
Cohesion: 0.08
Nodes (36): DefaultConfig(), FromRouteConfig(), HTTPSource, convertAttributesToClaims(), generateHTTPCorrelationID(), init(), runDaemon(), runMultiRoute() (+28 more)

### Community 8 - "Community 8"
Cohesion: 0.06
Nodes (27): deepEqual(), TestJSONataLibrarySpike(), RetentionPolicy, RetentionResolver, NewReaper(), TestReaperCreation(), TestReaperStart(), NewRetentionResolver() (+19 more)

### Community 9 - "Community 9"
Cohesion: 0.07
Nodes (31): Evaluator, FunctionDef, FunctionImpl, FunctionType, PluginInterface, PluginRuntime, Registry, WASMRuntime (+23 more)

### Community 10 - "Community 10"
Cohesion: 0.07
Nodes (26): DatabaseSink, NewDatabaseSink(), SinkConfig, TestDatabaseSinkInsert(), TestDatabaseSinkUpsert(), TestSinkConfigValidation(), Dataset, Job (+18 more)

### Community 11 - "Community 11"
Cohesion: 0.1
Nodes (23): NewAlternativeRegistry(), NewApicurioClient(), TestApicurioDefaultGroup(), TestApicurioHealth(), TestApicurioRegisterAndFetch(), TestLoadRegistryBackedContract(), NewMockRegistry(), TestMockRegistryDefaultGroup() (+15 more)

### Community 12 - "Community 12"
Cohesion: 0.08
Nodes (15): DatabaseSource, SourceConfig, FileSource, NewSFTPSource(), resolveSecret(), SourceConfig, S3Source, SourceConfig (+7 more)

### Community 13 - "Community 13"
Cohesion: 0.14
Nodes (25): TestViewerIntegrationFullFlow(), TestViewerIntegrationHighLoad(), TestViewerIntegrationMultipleRoutes(), TestViewerIntegrationTimestamp(), Message, NewRingBuffer(), NewRouteViewer(), NewViewerServer() (+17 more)

### Community 14 - "Community 14"
Cohesion: 0.07
Nodes (26): AuthorizeSpec, ContractSpec, ContractStepSpec, ErrorPathSpec, FilterSpec, IdempotentSpec, MetricsConfig, ObservabilityConfig (+18 more)

### Community 15 - "Community 15"
Cohesion: 0.1
Nodes (12): NewLogBasedCDCSource(), TestDebeziumEventParsing(), TestDebeziumOperationMapping(), TestMaxwellEventParsing(), TestTriggerCDCWatermarkTracking(), DebeziumEvent, LogBasedCDCConfig, LogBasedCDCSource (+4 more)

### Community 16 - "Community 16"
Cohesion: 0.13
Nodes (25): TestFixtureValidationMissingInput(), TestFixtureValidationMissingName(), TestFixtureValidationNoExpectation(), LoadFixturesFromDirectory(), LoadFixturesFromFile(), TestFixtureWithDroppedExpectation(), TestFixtureWithErrorExpectation(), TestLoadFixturesFromDirectory() (+17 more)

### Community 17 - "Community 17"
Cohesion: 0.14
Nodes (20): RouteVersion, sortedMap, canonicalJSON(), ComputeRouteVersion(), newSortedMap(), routeToSortedMap(), sortKeys(), stepToSortedMap() (+12 more)

### Community 18 - "Community 18"
Cohesion: 0.18
Nodes (20): JWTValidator, CreateTestToken(), ExtractBearerToken(), ExtractPrincipalFromHeader(), NewJWTValidator(), parseRSAPublicKey(), TestCreateTestToken(), TestExtractBearerToken() (+12 more)

### Community 19 - "Community 19"
Cohesion: 0.15
Nodes (12): isTruthy(), redactFields(), redactInNested(), redactPath(), AuthorizeStep, PDPContext, PDPDecisionRequest, PDPDecisionResponse (+4 more)

### Community 20 - "Community 20"
Cohesion: 0.23
Nodes (12): isNumber(), NewContractChecker(), TestContractCheck_ExtraFields(), TestContractCheck_MatchingContract(), TestContractCheck_MissingRequiredField(), TestContractCheck_NotAnalyzable(), TestContractCheck_TypeMismatch(), TestInferType() (+4 more)

### Community 21 - "Community 21"
Cohesion: 0.2
Nodes (10): makeDecisionRequest(), mockHandlerForAlice(), TestPBACContractConsistency_StubVsMock(), PBACFixture, StubPDP, NewStubPDP(), TestStubPDPConformance_Allow(), TestStubPDPConformance_Deny() (+2 more)

### Community 22 - "Community 22"
Cohesion: 0.19
Nodes (4): KafkaSource, NewKafkaSource(), NewKafkaSourceWithConfig(), SourceConfig

### Community 23 - "Community 23"
Cohesion: 0.31
Nodes (7): NewOPAAdapter(), TestOPAAdapterAllow(), TestOPAAdapterDeny(), TestOPAAdapterMissingResult(), TestOPAAdapterObligations(), TestOPAAdapterRequestTranslation(), OPAAdapter

### Community 24 - "Community 24"
Cohesion: 0.25
Nodes (4): AMQPSource, NewAMQPSource(), NewAMQPSourceWithConfig(), SourceConfig

### Community 25 - "Community 25"
Cohesion: 0.25
Nodes (5): GetContractViolation(), IsContractViolation(), ContractStep, ContractViolationError, ViolationInfo

### Community 26 - "Community 26"
Cohesion: 0.25
Nodes (2): S3Sink, SinkConfig

### Community 27 - "Community 27"
Cohesion: 0.25
Nodes (6): Message, Metadata, Principal, ReplayEntry, generateCorrelationID(), BenchmarkMessageCloning()

### Community 28 - "Community 28"
Cohesion: 0.38
Nodes (4): Result, TestResultSummary(), TestResultSummaryLargeNumbers(), TestResultSummaryZeros()

### Community 29 - "Community 29"
Cohesion: 0.5
Nodes (1): TestPrincipalWithoutAttributes()

### Community 30 - "Community 30"
Cohesion: 0.5
Nodes (3): Result, Sink, Source

### Community 31 - "Community 31"
Cohesion: 0.67
Nodes (1): ErrorType

### Community 32 - "Community 32"
Cohesion: 0.67
Nodes (1): DeadLetterEnvelope

### Community 33 - "Community 33"
Cohesion: 1.0
Nodes (1): Principal

## Knowledge Gaps
- **96 isolated node(s):** `PurgeOpts`, `ExportRecord`, `S3ExportConfig`, `LineageRecord`, `PurgeEvent` (+91 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 26`** (8 nodes): `s3_sink.go`, `S3Sink`, `.Close()`, `.HealthCheck()`, `.Start()`, `.Write()`, `NewS3Sink()`, `SinkConfig`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 29`** (4 nodes): `principal_test.go`, `TestPrincipalCreation()`, `TestPrincipalWithoutAttributes()`, `TestPrincipalWithoutRoles()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (3 nodes): `ErrorType`, `.String()`, `errors.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (3 nodes): `DeadLetterEnvelope`, `.MarshalJSON()`, `.UnmarshalJSON()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (2 nodes): `Principal`, `principal.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewMessage()` connect `Community 1` to `Community 0`, `Community 3`, `Community 6`, `Community 8`, `Community 10`, `Community 12`, `Community 15`, `Community 22`, `Community 24`, `Community 27`?**
  _High betweenness centrality (0.172) - this node is a cross-community bridge._
- **Why does `LoadRouteConfig()` connect `Community 5` to `Community 0`, `Community 17`, `Community 3`, `Community 7`?**
  _High betweenness centrality (0.094) - this node is a cross-community bridge._
- **Why does `runMultiRoute()` connect `Community 7` to `Community 0`, `Community 1`, `Community 2`, `Community 4`, `Community 5`?**
  _High betweenness centrality (0.049) - this node is a cross-community bridge._
- **Are the 235 inferred relationships involving `NewMessage()` (e.g. with `TestRecordLineage()` and `TestConcurrentWrites()`) actually correct?**
  _`NewMessage()` has 235 INFERRED edges - model-reasoned connections that need verification._
- **Are the 131 inferred relationships involving `NewChannel()` (e.g. with `TestWiretapMessageCopiedToTapSink()` and `TestWiretapOriginalContinuesDownstream()`) actually correct?**
  _`NewChannel()` has 131 INFERRED edges - model-reasoned connections that need verification._
- **Are the 44 inferred relationships involving `NewExecutor()` (e.g. with `TestWiretapIntegrationWithExecutor()` and `TestFileSinkIntegrationEndToEnd()`) actually correct?**
  _`NewExecutor()` has 44 INFERRED edges - model-reasoned connections that need verification._
- **Are the 40 inferred relationships involving `NewTracingProvider()` (e.g. with `TestTracingIntegrationEndToEnd()` and `TestTracingIntegrationErrorHandling()`) actually correct?**
  _`NewTracingProvider()` has 40 INFERRED edges - model-reasoned connections that need verification._