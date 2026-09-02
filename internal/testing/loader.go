package testing

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FixtureFile represents the top-level structure of a fixture YAML file
type FixtureFile struct {
	// Fixtures is the list of test fixtures
	Fixtures []*Fixture `yaml:"fixtures"`

	// Route is an optional route name for all fixtures in this file
	Route string `yaml:"route,omitempty"`

	// Timeout is an optional default timeout for all fixtures
	TimeoutMs int `yaml:"timeout_ms,omitempty"`
}

// LoadFixturesFromFile loads all fixtures from a single YAML file
func LoadFixturesFromFile(path string) ([]*Fixture, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file %s: %w", path, err)
	}

	// Parse YAML
	var fixtureFile FixtureFile
	if err := yaml.Unmarshal(data, &fixtureFile); err != nil {
		return nil, fmt.Errorf("failed to parse fixture file %s: %w", path, err)
	}

	// Validate and apply defaults
	fixtures := make([]*Fixture, 0, len(fixtureFile.Fixtures))
	for i, f := range fixtureFile.Fixtures {
		if f == nil {
			return nil, fmt.Errorf("fixture file %s: fixture at index %d is nil", path, i)
		}

		// Apply defaults
		if f.Name == "" {
			return nil, fmt.Errorf("fixture file %s: fixture at index %d has no name", path, i)
		}

		if f.Input == nil {
			return nil, fmt.Errorf("fixture file %s: fixture %q has no input", path, f.Name)
		}

		// Apply file-level route if fixture doesn't have one
		if f.ExpectedRoute == "" && fixtureFile.Route != "" {
			f.ExpectedRoute = fixtureFile.Route
		}

		// Apply file-level timeout if fixture doesn't have one
		if f.TimeoutMs == 0 && fixtureFile.TimeoutMs > 0 {
			f.TimeoutMs = fixtureFile.TimeoutMs
		}

		fixtures = append(fixtures, f)
	}

	return fixtures, nil
}

// LoadFixturesFromDirectory loads all .yaml fixture files from a directory
func LoadFixturesFromDirectory(dirPath string) ([]*Fixture, error) {
	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat directory %s: %w", dirPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dirPath)
	}

	// List all .yaml files in directory
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	var allFixtures []*Fixture
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if file ends with .yaml
		filename := entry.Name()
		if filepath.Ext(filename) != ".yaml" && filepath.Ext(filename) != ".yml" {
			continue
		}

		// Load fixtures from this file
		filePath := filepath.Join(dirPath, filename)
		fixtures, err := LoadFixturesFromFile(filePath)
		if err != nil {
			// Return error with context
			return nil, fmt.Errorf("failed to load fixtures from %s: %w", filePath, err)
		}

		allFixtures = append(allFixtures, fixtures...)
	}

	return allFixtures, nil
}

// ValidateFixture checks that a fixture has required fields
func ValidateFixture(fixture *Fixture) error {
	if fixture == nil {
		return fmt.Errorf("fixture is nil")
	}

	if fixture.Name == "" {
		return fmt.Errorf("fixture has no name")
	}

	if fixture.Input == nil {
		return fmt.Errorf("fixture %q has no input", fixture.Name)
	}

	// At least one expectation must be specified
	hasExpectation := fixture.ExpectedOutput != nil ||
		fixture.ExpectedDropped ||
		fixture.ExpectedError ||
		fixture.ExpectedRoute != ""

	if !hasExpectation {
		return fmt.Errorf("fixture %q has no expectations (at least one of expected_output, expected_dropped, expected_error, expected_route must be specified)", fixture.Name)
	}

	// Validate timeout if specified
	if fixture.TimeoutMs < 0 {
		return fmt.Errorf("fixture %q has negative timeout", fixture.Name)
	}

	return nil
}

// ValidateFixtures validates all fixtures in a list
func ValidateFixtures(fixtures []*Fixture) error {
	for i, fixture := range fixtures {
		if err := ValidateFixture(fixture); err != nil {
			return fmt.Errorf("fixture at index %d: %w", i, err)
		}
	}
	return nil
}
