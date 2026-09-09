package steps

import (
	"fmt"
	"os"

	"github.com/naren-chakraview/dim/internal/adapters/claimcheck"
	"github.com/naren-chakraview/dim/internal/cluster"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// DedupStore is an alias for cluster.DedupStore to make it available here
type DedupStore = cluster.DedupStore

// newClaimCheckStore creates a ClaimCheckStore based on environment configuration.
// Checks CLAIM_CHECK_STORE_TYPE env var (default: "memory", supported: "s3")
// For S3, requires S3_BUCKET and optionally S3_REGION and S3_PREFIX.
func newClaimCheckStore() (claimcheck.ClaimCheckStore, error) {
	storeType := os.Getenv("CLAIM_CHECK_STORE_TYPE")
	if storeType == "" {
		storeType = "memory"
	}

	switch storeType {
	case "memory":
		return claimcheck.NewInMemoryClaimCheckStore(), nil
	case "s3":
		bucket := os.Getenv("S3_BUCKET")
		if bucket == "" {
			return nil, fmt.Errorf("S3 claim-check store requires S3_BUCKET environment variable")
		}
		region := os.Getenv("S3_REGION")
		if region == "" {
			region = "us-east-1"
		}
		prefix := os.Getenv("S3_PREFIX")
		if prefix == "" {
			prefix = "claim-check/"
		}
		return claimcheck.NewS3ClaimCheckStore(claimcheck.S3StoreConfig{
			Bucket: bucket,
			Region: region,
			Prefix: prefix,
		})
	default:
		return nil, fmt.Errorf("unknown claim-check store type: %s (supported: memory, s3)", storeType)
	}
}

// BuildStepsFromSpec creates step instances from a slice of step specifications.
// Returns (steps, stepNames, error).
// stepNames is a parallel slice mapping step indices to their type names (e.g., "filter", "translate").
//
// Supported step types:
// - filter: Boolean predicate to drop messages (Phase 0+)
// - translate: JSONata expression to transform message body (Phase 0+)
// - route: Content-based router (M0.2.3+)
// - wiretap: Copy to secondary sink (M0.2.4+)
// - authorize: RBAC/ABAC policy enforcement (M0.2.6+)
// - contract: Data contract validation with JSON Schema (M0.3.5+)
// - idempotent: Message deduplication (M0.2.5, requires dedupStore parameter)
//
// dedupStore may be nil, in which case idempotent steps use in-memory dedup (single-instance mode).
// For cluster mode, pass the cluster's DedupStore to enable cross-instance deduplication.
func BuildStepsFromSpec(stepSpecs []config.StepSpec, contractStore *config.ContractStore, routeName string, dedupStore DedupStore) ([]engine.Step, []string, error) {
	var steps []engine.Step
	var stepNames []string

	for i, spec := range stepSpecs {
		// Count how many step types are defined (should be exactly 1)
		stepTypeCount := 0

		if spec.Filter != nil {
			stepTypeCount++
		}
		if spec.Translate != nil {
			stepTypeCount++
		}
		if spec.Route != nil {
			stepTypeCount++
		}
		if spec.Wiretap != nil {
			stepTypeCount++
		}
		if spec.Idempotent != nil {
			stepTypeCount++
		}
		if spec.Authorize != nil {
			stepTypeCount++
		}
		if spec.Contract != nil {
			stepTypeCount++
		}
		if spec.ClaimCheck != nil {
			stepTypeCount++
		}
		if spec.ClaimResolve != nil {
			stepTypeCount++
		}

		if stepTypeCount == 0 {
			return nil, nil, fmt.Errorf("step %d: no step type specified", i)
		}
		if stepTypeCount > 1 {
			return nil, nil, fmt.Errorf("step %d: multiple step types specified (exactly one required)", i)
		}

		// Build the step based on type
		var step engine.Step
		var err error

		switch {
		case spec.Filter != nil:
			step, err = NewFilterStep(spec.Filter.Expr)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (filter): %w", i, err)
			}
			stepNames = append(stepNames, "filter")

		case spec.Translate != nil:
			step, err = NewTranslateStep(spec.Translate.Expr)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (translate): %w", i, err)
			}
			stepNames = append(stepNames, "translate")

		case spec.Route != nil:
			// Convert RouteCase objects to a map[string]string
			casesMap := make(map[string]string)
			for caseName, routeCase := range spec.Route.Cases {
				casesMap[caseName] = routeCase.Target
			}

			step, err = NewRouteStep(spec.Route.Expr, casesMap, spec.Route.Default)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (route): %w", i, err)
			}
			stepNames = append(stepNames, "route")
		case spec.Wiretap != nil:
			step, err = NewWiretapStep(spec.Wiretap.Sink)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (wiretap): %w", i, err)
			}
			stepNames = append(stepNames, "wiretap")
		case spec.Idempotent != nil:
			// Build idempotent step with optional cluster dedup store
			if dedupStore != nil {
				// Cluster mode: use shared dedup store
				step, err = NewIdempotentStepWithStore(spec.Idempotent.KeyExpr, dedupStore)
				if err != nil {
					return nil, nil, fmt.Errorf("step %d (idempotent): %w", i, err)
				}
			} else {
				// Single-instance mode: use in-memory dedup (default 60 minutes)
				step, err = NewIdempotentStep(spec.Idempotent.KeyExpr, 60)
				if err != nil {
					return nil, nil, fmt.Errorf("step %d (idempotent): %w", i, err)
				}
			}
			stepNames = append(stepNames, "idempotent")
		case spec.Authorize != nil:
			pdpEndpoint := ""
			pdpTimeout := 5000
			if spec.Authorize.PDP != nil {
				pdpEndpoint = spec.Authorize.PDP.Endpoint
				if spec.Authorize.PDP.Timeout > 0 {
					pdpTimeout = spec.Authorize.PDP.Timeout
				}
			}
			step, err = NewAuthorizeStep(spec.Authorize.Mode, spec.Authorize.RequireRoles, spec.Authorize.Expr, pdpEndpoint, pdpTimeout)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (authorize): %w", i, err)
			}
			stepNames = append(stepNames, "authorize")
		case spec.Contract != nil:
			if contractStore == nil {
				return nil, nil, fmt.Errorf("step %d (contract): contract store is nil", i)
			}
			step, err = NewContractStep(contractStore, routeName, spec.Contract.ID)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (contract): %w", i, err)
			}
			stepNames = append(stepNames, "contract")
		case spec.ClaimCheck != nil:
			claimStore, err := newClaimCheckStore()
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (claim_check): failed to initialize store: %w", i, err)
			}
			step, err = NewClaimCheckStep(spec.ClaimCheck, claimStore)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (claim_check): %w", i, err)
			}
			stepNames = append(stepNames, "claim_check")
		case spec.ClaimResolve != nil:
			claimStore, err := newClaimCheckStore()
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (claim_resolve): failed to initialize store: %w", i, err)
			}
			step, err = NewClaimResolveStep(spec.ClaimResolve, claimStore)
			if err != nil {
				return nil, nil, fmt.Errorf("step %d (claim_resolve): %w", i, err)
			}
			stepNames = append(stepNames, "claim_resolve")
		}

		steps = append(steps, step)
	}

	return steps, stepNames, nil
}
