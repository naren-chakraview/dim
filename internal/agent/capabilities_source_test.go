package agent

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/naren-chakraview/dim/schemas"
)

const committedManifestPath = "../../docs/schemas/capability-manifest.json"

// normalized round-trips capabilities through JSON so Go types compare the same
// way the committed file does.
func normalized(t *testing.T, caps []Capability) []interface{} {
	t.Helper()
	b, err := json.Marshal(caps)
	if err != nil {
		t.Fatal(err)
	}
	var out []interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func committedCapabilities(t *testing.T) []interface{} {
	t.Helper()
	b, err := os.ReadFile(committedManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Capabilities []interface{} `json:"capabilities"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m.Capabilities
}

// manifestMatchesCommitted is the drift comparison: full entries, not just names.
func manifestMatchesCommitted(t *testing.T, caps []Capability) bool {
	t.Helper()
	return reflect.DeepEqual(normalized(t, caps), committedCapabilities(t))
}

// The served catalog (GetCapabilities, which `dimctl capabilities` also uses)
// must equal the committed manifest exactly.
func TestServedCapabilitiesMatchCommittedManifest(t *testing.T) {
	resp, err := GetCapabilities(context.Background(), CapabilitiesRequest{})
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err.Message)
	}
	if !manifestMatchesCommitted(t, resp.Capabilities) {
		t.Fatal("served capabilities differ from docs/schemas/capability-manifest.json; regenerate with `dimctl manifest generate schemas/route.schema.json`")
	}
}

// Simulates the reviewer case: a real adapter plus its schema entry is added.
// The committed manifest must no longer match, so the drift check fails.
func TestDriftCheckCatchesAddedAdapter(t *testing.T) {
	var schema map[string]interface{}
	if err := json.Unmarshal(schemas.RouteSchema, &schema); err != nil {
		t.Fatal(err)
	}
	defs := schema["definitions"].(map[string]interface{})
	for _, dir := range []string{"source", "sink"} {
		props := defs[dir].(map[string]interface{})["properties"].(map[string]interface{})
		typ := props["type"].(map[string]interface{})
		typ["enum"] = append(typ["enum"].([]interface{}), "mqtt")
	}
	modified, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := BuildCapabilities(modified, "")
	if err != nil {
		t.Fatalf("BuildCapabilities on modified schema: %v", err)
	}
	if manifestMatchesCommitted(t, caps) {
		t.Fatal("drift not detected: adding an adapter to the schema left the committed manifest matching")
	}
	names := map[string]bool{}
	for _, c := range caps {
		names[c.Name] = true
	}
	if !names["mqtt-source"] || !names["mqtt-sink"] {
		t.Fatalf("new adapter not served: %v", names)
	}
}

// The schema adapter enum must be exactly the adapters that exist in internal/adapters.
func TestSchemaAdapterEnumMatchesImplementedAdapters(t *testing.T) {
	got, err := SchemaAdapterTypes()
	if err != nil {
		t.Fatal(err)
	}
	// internal/adapters/{http,file,s3,kafka,amqp,database}: each has a source and a sink.
	// sftp is a source type served by the file adapter's SFTP mode (file.NewSFTPSource).
	want := []string{"amqp", "database", "file", "http", "kafka", "s3", "sftp"}
	sorted := append([]string(nil), got...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(sorted, want) {
		t.Fatalf("schema adapter enum = %v, implemented adapters = %v", sorted, want)
	}
}

// Steps must be the real StepSpec set; aggregate and split are not loadable from a route.
func TestServedCatalogHasNoPhantomSteps(t *testing.T) {
	caps, err := ServedCapabilities("step")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range caps {
		names = append(names, c.Name)
	}
	for _, phantom := range []string{"aggregate-step", "split-step"} {
		for _, n := range names {
			if n == phantom {
				t.Errorf("phantom step %q is served", phantom)
			}
		}
	}
	want := []string{"filter-step", "translate-step", "route-step", "wiretap-step", "idempotent-step",
		"authorize-step", "contract-step", "claim_check-step", "claim_resolve-step"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("steps = %v, want %v", names, want)
	}
}

func TestServedCatalogCounts(t *testing.T) {
	resp, err := GetCapabilities(context.Background(), CapabilitiesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, c := range resp.Capabilities {
		counts[c.Type]++
	}
	if counts["adapter"] != 14 || counts["step"] != 9 || counts["eip"] != 4 {
		t.Errorf("counts = %v, want adapter=14 step=9 eip=4", counts)
	}
}

func TestGetCapabilitiesRejectsUnknownFilter(t *testing.T) {
	_, err := GetCapabilities(context.Background(), CapabilitiesRequest{FilterType: "bogus"})
	if err == nil || !strings.Contains(err.Message, "unknown capability filter") {
		t.Fatalf("expected unknown-filter error, got %v", err)
	}
}

func TestBuildCapabilitiesFailsOnBrokenEIPBinding(t *testing.T) {
	broken := []byte(`{"definitions":{"source":{"properties":{"type":{"enum":["http"]}}},"sink":{"properties":{"type":{"enum":["http"]}}},"step":{"oneOf":[]}}}`)
	if _, err := BuildCapabilities(broken, "eip"); err == nil {
		t.Fatal("expected error for EIP bound to a missing definition")
	}
}
