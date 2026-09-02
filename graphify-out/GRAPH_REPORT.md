# Graph Report - dim  (2026-09-01)

## Corpus Check
- 58 files · ~51,877 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 273 nodes · 642 edges · 24 communities detected
- Extraction: 34% EXTRACTED · 66% INFERRED · 0% AMBIGUOUS · INFERRED: 424 edges (avg confidence: 0.8)
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
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]

## God Nodes (most connected - your core abstractions)
1. `NewMessage()` - 66 edges
2. `NewChannel()` - 46 edges
3. `NewTranslateStep()` - 20 edges
4. `LoadRouteConfig()` - 19 edges
5. `BuildStepsFromSpec()` - 16 edges
6. `NewExecutor()` - 15 edges
7. `NewFilterStep()` - 14 edges
8. `NewFileSink()` - 14 edges
9. `BuildPipeline()` - 13 edges
10. `NewHTTPSource()` - 10 edges

## Surprising Connections (you probably didn't know these)
- `TestMissingConfigFile()` --calls--> `LoadRouteConfig()`  [INFERRED]
  cmd/midctl/run_test.go → internal/config/loader.go
- `TestInvalidConfigFile()` --calls--> `LoadRouteConfig()`  [INFERRED]
  cmd/midctl/run_test.go → internal/config/loader.go
- `TestRunCommandEndToEnd()` --calls--> `BuildPipeline()`  [INFERRED]
  cmd/midctl/run_test.go → internal/factory/pipeline.go
- `TestMessagePrincipal()` --calls--> `NewMessage()`  [INFERRED]
  internal/engine/message_test.go → internal/engine/message.go
- `TestNewTranslateStep()` --calls--> `NewTranslateStep()`  [INFERRED]
  internal/steps/translate_test.go → internal/steps/translate.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.17
Nodes (35): NewChannel(), TestChannelBackpressure(), TestChannelConcurrentSendRecv(), TestChannelContextCancellation(), TestChannelInFlightTracking(), TestChannelNilMessage(), TestChannelSendRecv(), TestChannelTryRecv() (+27 more)

### Community 1 - "Community 1"
Cohesion: 0.18
Nodes (32): mockStep, NewFilterStep(), TestFilterStepAlwaysFalse(), TestFilterStepAlwaysTrue(), TestFilterStepBodyBasedPredicateAccept(), TestFilterStepBodyBasedPredicateReject(), TestFilterStepEvaluationError(), TestFilterStepHeadersAvailable() (+24 more)

### Community 2 - "Community 2"
Cohesion: 0.09
Nodes (14): Message, Metadata, Principal, Evaluator, CompileExpression(), deepEqual(), TestJSONataLibrarySpike(), generateCorrelationID() (+6 more)

### Community 3 - "Community 3"
Cohesion: 0.15
Nodes (21): getSchemaDir(), LoadRouteConfig(), loadSchema(), TestLoadConfigFileNotFound(), TestLoadConfigInvalidYAML(), TestLoadConfigMissingAuth(), TestLoadConfigMissingErrorPath(), TestLoadConfigMissingErrorPathTarget() (+13 more)

### Community 4 - "Community 4"
Cohesion: 0.19
Nodes (16): FileSinkAdapter, PassthroughStep, TestFileSinkBackpressure(), TestFileSinkIntegrationEndToEnd(), TestFileSinkWithComplexMessageStructure(), NewFileSink(), TestFileSinkAppendMode(), TestFileSinkBasic() (+8 more)

### Community 5 - "Community 5"
Cohesion: 0.17
Nodes (10): HTTPSourceAdapter, SinkAdapter, SourceAdapter, BuildPipeline(), TestBuildPipelineErrorSinkMissing(), TestBuildPipelineInvalidFilterStep(), TestBuildPipelineInvalidTranslateStep(), TestBuildPipelineNoOutputSink() (+2 more)

### Community 6 - "Community 6"
Cohesion: 0.24
Nodes (13): BuildStepsFromSpec(), TestBuildStepsFromSpecAuthorizeStepNotImplemented(), TestBuildStepsFromSpecEmptySteps(), TestBuildStepsFromSpecFilter(), TestBuildStepsFromSpecIdempotentStepNotImplemented(), TestBuildStepsFromSpecInvalidFilterExpression(), TestBuildStepsFromSpecInvalidTranslateExpression(), TestBuildStepsFromSpecMultiple() (+5 more)

### Community 7 - "Community 7"
Cohesion: 0.13
Nodes (14): AuthorizeSpec, ErrorPathSpec, FilterSpec, IdempotentSpec, RetryPolicy, RouteCase, RouteConfig, RouteSpec (+6 more)

### Community 8 - "Community 8"
Cohesion: 0.21
Nodes (9): HTTPSource, generateHTTPCorrelationID(), NewHTTPSource(), TestHTTPSourceContextCancellation(), TestHTTPSourceCorrelationID(), TestHTTPSourceEmptyBody(), TestHTTPSourceMalformedJSON(), TestHTTPSourceMethodNotAllowed() (+1 more)

### Community 9 - "Community 9"
Cohesion: 0.23
Nodes (7): NewDeadLetterEnvelope(), TestDeadLetterEnvelopeCreation(), TestDeadLetterEnvelopeCreationFilterFailure(), TestDeadLetterEnvelopeCreationInvalidExpressionError(), TestDeadLetterEnvelopeCreationTranslateFailure(), TestDeadLetterEnvelopeMarshalJSON(), DeadLetterEnvelope

### Community 10 - "Community 10"
Cohesion: 0.5
Nodes (1): FileSink

### Community 11 - "Community 11"
Cohesion: 0.5
Nodes (1): main()

### Community 40 - "Community 40"
Cohesion: 1.0
Nodes (1): Data Contracts

### Community 41 - "Community 41"
Cohesion: 1.0
Nodes (1): Declarative Integration Middleware

### Community 42 - "Community 42"
Cohesion: 1.0
Nodes (1): Route (Message Pipeline)

### Community 43 - "Community 43"
Cohesion: 1.0
Nodes (1): Lineage Store

### Community 44 - "Community 44"
Cohesion: 1.0
Nodes (1): M0.1 Walking Skeleton

### Community 45 - "Community 45"
Cohesion: 1.0
Nodes (1): M0.2 Reliability

### Community 46 - "Community 46"
Cohesion: 1.0
Nodes (1): Authorize Step

### Community 47 - "Community 47"
Cohesion: 1.0
Nodes (1): Filter Step

### Community 48 - "Community 48"
Cohesion: 1.0
Nodes (1): Route Step

### Community 49 - "Community 49"
Cohesion: 1.0
Nodes (1): Translate Step

### Community 50 - "Community 50"
Cohesion: 1.0
Nodes (1): EIP Step Types

### Community 51 - "Community 51"
Cohesion: 1.0
Nodes (1): Hot Reload

## Knowledge Gaps
- **31 isolated node(s):** `RouteConfig`, `SourceSpec`, `SinkSpec`, `RouteSpec`, `ErrorPathSpec` (+26 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 10`** (5 nodes): `FileSink`, `.Close()`, `.run()`, `.Start()`, `file_sink.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 11`** (4 nodes): `main.go`, `main.go`, `init()`, `main()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (1 nodes): `Data Contracts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (1 nodes): `Declarative Integration Middleware`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (1 nodes): `Route (Message Pipeline)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (1 nodes): `Lineage Store`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (1 nodes): `M0.1 Walking Skeleton`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (1 nodes): `M0.2 Reliability`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (1 nodes): `Authorize Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (1 nodes): `Filter Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (1 nodes): `Route Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (1 nodes): `Translate Step`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (1 nodes): `EIP Step Types`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (1 nodes): `Hot Reload`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewMessage()` connect `Community 1` to `Community 0`, `Community 9`, `Community 2`, `Community 4`?**
  _High betweenness centrality (0.175) - this node is a cross-community bridge._
- **Why does `TestRunCommandEndToEnd()` connect `Community 4` to `Community 0`, `Community 3`, `Community 5`?**
  _High betweenness centrality (0.134) - this node is a cross-community bridge._
- **Why does `BuildPipeline()` connect `Community 5` to `Community 0`, `Community 8`, `Community 4`, `Community 6`?**
  _High betweenness centrality (0.121) - this node is a cross-community bridge._
- **Are the 64 inferred relationships involving `NewMessage()` (e.g. with `TestTranslateSimpleFieldAccess()` and `TestTranslateObjectConstruction()`) actually correct?**
  _`NewMessage()` has 64 INFERRED edges - model-reasoned connections that need verification._
- **Are the 45 inferred relationships involving `NewChannel()` (e.g. with `TestFileSinkIntegrationEndToEnd()` and `TestFileSinkBackpressure()`) actually correct?**
  _`NewChannel()` has 45 INFERRED edges - model-reasoned connections that need verification._
- **Are the 19 inferred relationships involving `NewTranslateStep()` (e.g. with `TestNewTranslateStep()` and `TestNewTranslateStepInvalidExpression()`) actually correct?**
  _`NewTranslateStep()` has 19 INFERRED edges - model-reasoned connections that need verification._
- **Are the 17 inferred relationships involving `LoadRouteConfig()` (e.g. with `TestLoadValidRouteConfig()` and `TestLoadConfigMissingVersion()`) actually correct?**
  _`LoadRouteConfig()` has 17 INFERRED edges - model-reasoned connections that need verification._