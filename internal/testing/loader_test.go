package testing

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadFixturesFromFile tests loading fixtures from a single YAML file
func TestLoadFixturesFromFile(t *testing.T) {
	// Create a temporary fixture file
	tmpDir := t.TempDir()
	fixtureFile := filepath.Join(tmpDir, "test.yaml")

	content := `fixtures:
  - name: "test1"
    input:
      value: 42
    expected_output:
      value: 42
  - name: "test2"
    input:
      value: 100
    expected_output:
      value: 100
`

	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	fixtures, err := LoadFixturesFromFile(fixtureFile)
	if err != nil {
		t.Fatalf("failed to load fixtures: %v", err)
	}

	if len(fixtures) != 2 {
		t.Errorf("expected 2 fixtures, got %d", len(fixtures))
	}

	if fixtures[0].Name != "test1" {
		t.Errorf("expected first fixture name to be 'test1', got '%s'", fixtures[0].Name)
	}

	if fixtures[1].Name != "test2" {
		t.Errorf("expected second fixture name to be 'test2', got '%s'", fixtures[1].Name)
	}
}

// TestLoadFixturesFromFileWithDefaults tests loading fixtures with file-level defaults
func TestLoadFixturesFromFileWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	fixtureFile := filepath.Join(tmpDir, "test.yaml")

	content := `route: default-route
timeout_ms: 5000
fixtures:
  - name: "test1"
    input:
      value: 42
    expected_output:
      value: 42
  - name: "test2"
    input:
      value: 100
    expected_output:
      value: 100
    expected_route: custom-route
`

	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	fixtures, err := LoadFixturesFromFile(fixtureFile)
	if err != nil {
		t.Fatalf("failed to load fixtures: %v", err)
	}

	if len(fixtures) != 2 {
		t.Errorf("expected 2 fixtures, got %d", len(fixtures))
	}

	// First fixture should inherit file-level defaults
	if fixtures[0].ExpectedRoute != "default-route" {
		t.Errorf("expected first fixture route to be 'default-route', got '%s'", fixtures[0].ExpectedRoute)
	}

	if fixtures[0].TimeoutMs != 5000 {
		t.Errorf("expected first fixture timeout to be 5000, got %d", fixtures[0].TimeoutMs)
	}

	// Second fixture should override file-level route
	if fixtures[1].ExpectedRoute != "custom-route" {
		t.Errorf("expected second fixture route to be 'custom-route', got '%s'", fixtures[1].ExpectedRoute)
	}

	// Second fixture should still inherit timeout
	if fixtures[1].TimeoutMs != 5000 {
		t.Errorf("expected second fixture timeout to be 5000, got %d", fixtures[1].TimeoutMs)
	}
}

// TestLoadFixturesFromFileNotFound tests error handling for missing files
func TestLoadFixturesFromFileNotFound(t *testing.T) {
	_, err := LoadFixturesFromFile("/nonexistent/path/file.yaml")
	if err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}
}

// TestLoadFixturesFromFileInvalidYAML tests error handling for invalid YAML
func TestLoadFixturesFromFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	fixtureFile := filepath.Join(tmpDir, "invalid.yaml")

	content := `invalid: yaml: content: [
`

	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	_, err := LoadFixturesFromFile(fixtureFile)
	if err == nil {
		t.Errorf("expected error for invalid YAML, got nil")
	}
}

// TestLoadFixturesFromFileNoName tests error handling for fixtures without name
func TestLoadFixturesFromFileNoName(t *testing.T) {
	tmpDir := t.TempDir()
	fixtureFile := filepath.Join(tmpDir, "test.yaml")

	content := `fixtures:
  - input:
      value: 42
    expected_output:
      value: 42
`

	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	_, err := LoadFixturesFromFile(fixtureFile)
	if err == nil {
		t.Errorf("expected error for fixture without name, got nil")
	}
}

// TestLoadFixturesFromFileNoInput tests error handling for fixtures without input
func TestLoadFixturesFromFileNoInput(t *testing.T) {
	tmpDir := t.TempDir()
	fixtureFile := filepath.Join(tmpDir, "test.yaml")

	content := `fixtures:
  - name: "test1"
    expected_output:
      value: 42
`

	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	_, err := LoadFixturesFromFile(fixtureFile)
	if err == nil {
		t.Errorf("expected error for fixture without input, got nil")
	}
}

// TestLoadFixturesFromDirectory tests loading all fixtures from a directory
func TestLoadFixturesFromDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first fixture file
	fixture1 := filepath.Join(tmpDir, "test1.yaml")
	content1 := `fixtures:
  - name: "fixture1"
    input:
      value: 1
    expected_output:
      value: 1
`
	if err := os.WriteFile(fixture1, []byte(content1), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	// Create second fixture file
	fixture2 := filepath.Join(tmpDir, "test2.yaml")
	content2 := `fixtures:
  - name: "fixture2"
    input:
      value: 2
    expected_output:
      value: 2
  - name: "fixture3"
    input:
      value: 3
    expected_output:
      value: 3
`
	if err := os.WriteFile(fixture2, []byte(content2), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	// Create a non-YAML file (should be skipped)
	other := filepath.Join(tmpDir, "other.txt")
	if err := os.WriteFile(other, []byte("some text"), 0644); err != nil {
		t.Fatalf("failed to write other file: %v", err)
	}

	fixtures, err := LoadFixturesFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("failed to load fixtures from directory: %v", err)
	}

	if len(fixtures) != 3 {
		t.Errorf("expected 3 fixtures total, got %d", len(fixtures))
	}
}

// TestLoadFixturesFromDirectoryNotFound tests error handling for missing directory
func TestLoadFixturesFromDirectoryNotFound(t *testing.T) {
	_, err := LoadFixturesFromDirectory("/nonexistent/directory")
	if err == nil {
		t.Errorf("expected error for non-existent directory, got nil")
	}
}

// TestLoadFixturesFromDirectoryNotADirectory tests error handling for non-directory path
func TestLoadFixturesFromDirectoryNotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")

	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := LoadFixturesFromDirectory(filePath)
	if err == nil {
		t.Errorf("expected error when path is not a directory, got nil")
	}
}

// TestLoadFixturesFromDirectoryEmpty tests loading from empty directory
func TestLoadFixturesFromDirectoryEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	fixtures, err := LoadFixturesFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("failed to load fixtures from empty directory: %v", err)
	}

	if len(fixtures) != 0 {
		t.Errorf("expected 0 fixtures from empty directory, got %d", len(fixtures))
	}
}

// TestLoadFixturesFromDirectoryYmlExtension tests loading .yml files
func TestLoadFixturesFromDirectoryYmlExtension(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fixture file with .yml extension
	fixtureFile := filepath.Join(tmpDir, "test.yml")
	content := `fixtures:
  - name: "yml-fixture"
    input:
      value: 42
    expected_output:
      value: 42
`

	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	fixtures, err := LoadFixturesFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("failed to load fixtures from directory: %v", err)
	}

	if len(fixtures) != 1 {
		t.Errorf("expected 1 fixture, got %d", len(fixtures))
	}

	if fixtures[0].Name != "yml-fixture" {
		t.Errorf("expected fixture name 'yml-fixture', got '%s'", fixtures[0].Name)
	}
}

// TestValidateFixtureValid tests validation of a valid fixture
func TestValidateFixtureValid(t *testing.T) {
	fixture := &Fixture{
		Name:           "valid",
		Input:          map[string]interface{}{"value": 42},
		ExpectedOutput: map[string]interface{}{"value": 42},
	}

	err := ValidateFixture(fixture)
	if err != nil {
		t.Errorf("expected no error for valid fixture, got %v", err)
	}
}

// TestValidateFixtureNil tests validation of nil fixture
func TestValidateFixtureNil(t *testing.T) {
	err := ValidateFixture(nil)
	if err == nil {
		t.Errorf("expected error for nil fixture, got nil")
	}
}

// TestValidateFixtures tests batch validation
func TestValidateFixtures(t *testing.T) {
	fixtures := []*Fixture{
		{
			Name:           "valid1",
			Input:          map[string]interface{}{"value": 1},
			ExpectedOutput: map[string]interface{}{"value": 1},
		},
		{
			Name:           "valid2",
			Input:          map[string]interface{}{"value": 2},
			ExpectedOutput: map[string]interface{}{"value": 2},
		},
	}

	err := ValidateFixtures(fixtures)
	if err != nil {
		t.Errorf("expected no error for valid fixtures, got %v", err)
	}
}

// TestValidateFixturesWithInvalid tests batch validation with invalid fixture
func TestValidateFixturesWithInvalid(t *testing.T) {
	fixtures := []*Fixture{
		{
			Name:           "valid",
			Input:          map[string]interface{}{"value": 1},
			ExpectedOutput: map[string]interface{}{"value": 1},
		},
		{
			Name:   "invalid",
			Input:  map[string]interface{}{"value": 2},
			// No expected output or other expectation
		},
	}

	err := ValidateFixtures(fixtures)
	if err == nil {
		t.Errorf("expected error for batch with invalid fixture, got nil")
	}
}

// TestFixtureWithDroppedExpectation tests fixture with expected_dropped
func TestFixtureWithDroppedExpectation(t *testing.T) {
	tmpDir := t.TempDir()
	fixtureFile := filepath.Join(tmpDir, "test.yaml")

	content := `fixtures:
  - name: "dropped-test"
    input:
      value: 42
    expected_dropped: true
`

	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	fixtures, err := LoadFixturesFromFile(fixtureFile)
	if err != nil {
		t.Fatalf("failed to load fixtures: %v", err)
	}

	if !fixtures[0].ExpectedDropped {
		t.Errorf("expected ExpectedDropped to be true")
	}

	// Validate - should pass since expected_dropped is set
	err = ValidateFixture(fixtures[0])
	if err != nil {
		t.Errorf("expected validation to pass for fixture with expected_dropped, got %v", err)
	}
}

// TestFixtureWithErrorExpectation tests fixture with expected_error
func TestFixtureWithErrorExpectation(t *testing.T) {
	tmpDir := t.TempDir()
	fixtureFile := filepath.Join(tmpDir, "test.yaml")

	content := `fixtures:
  - name: "error-test"
    input:
      value: 42
    expected_error: true
    expected_error_type: "ValidationError"
`

	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	fixtures, err := LoadFixturesFromFile(fixtureFile)
	if err != nil {
		t.Fatalf("failed to load fixtures: %v", err)
	}

	if !fixtures[0].ExpectedError {
		t.Errorf("expected ExpectedError to be true")
	}

	if fixtures[0].ExpectedErrorType != "ValidationError" {
		t.Errorf("expected ExpectedErrorType to be 'ValidationError', got '%s'", fixtures[0].ExpectedErrorType)
	}

	// Validate - should pass since expected_error is set
	err = ValidateFixture(fixtures[0])
	if err != nil {
		t.Errorf("expected validation to pass for fixture with expected_error, got %v", err)
	}
}
