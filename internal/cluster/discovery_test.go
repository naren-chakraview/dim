package cluster

import (
	"os"
	"testing"
)

func TestStaticClusterConfig(t *testing.T) {
	instances := []Instance{
		{ID: "dimd-1", Host: "10.0.1.5", Port: 8080},
		{ID: "dimd-2", Host: "10.0.1.6", Port: 8080},
		{ID: "dimd-3", Host: "10.0.1.7", Port: 8080},
	}

	config, err := NewClusterConfigFromStatic(instances, "dimd-1")
	if err != nil {
		t.Fatalf("failed to create cluster config: %v", err)
	}

	if config.Mode != "static" {
		t.Errorf("expected mode=static, got %s", config.Mode)
	}

	if config.CurrentID != "dimd-1" {
		t.Errorf("expected CurrentID=dimd-1, got %s", config.CurrentID)
	}

	if !config.IsClustered() {
		t.Errorf("expected IsClustered=true")
	}

	if len(config.OtherInstances()) != 2 {
		t.Errorf("expected 2 other instances, got %d", len(config.OtherInstances()))
	}

	if config.CurrentInstance() == nil {
		t.Errorf("expected current instance to exist")
	}
}

func TestStaticClusterConfig_InvalidCurrentID(t *testing.T) {
	instances := []Instance{
		{ID: "dimd-1", Host: "10.0.1.5", Port: 8080},
		{ID: "dimd-2", Host: "10.0.1.6", Port: 8080},
	}

	_, err := NewClusterConfigFromStatic(instances, "dimd-99")
	if err == nil {
		t.Errorf("expected error for invalid current ID")
	}
}

func TestEnvClusterConfig(t *testing.T) {
	os.Setenv("DIMD_INSTANCE_ID", "dimd-1")
	os.Setenv("DIMD_CLUSTER_HOSTS", "10.0.1.5:8080,10.0.1.6:8080,10.0.1.7:8080")
	defer func() {
		os.Unsetenv("DIMD_INSTANCE_ID")
		os.Unsetenv("DIMD_CLUSTER_HOSTS")
	}()

	config, err := NewClusterConfigFromEnv()
	if err != nil {
		t.Fatalf("failed to create cluster config from env: %v", err)
	}

	if config.Mode != "env" {
		t.Errorf("expected mode=env, got %s", config.Mode)
	}

	if config.CurrentID != "dimd-1" {
		t.Errorf("expected CurrentID=dimd-1, got %s", config.CurrentID)
	}

	if len(config.Instances) != 3 {
		t.Errorf("expected 3 instances, got %d", len(config.Instances))
	}

	if !config.IsClustered() {
		t.Errorf("expected IsClustered=true")
	}
}

func TestEnvClusterConfig_InvalidFormat(t *testing.T) {
	os.Setenv("DIMD_INSTANCE_ID", "dimd-1")
	os.Setenv("DIMD_CLUSTER_HOSTS", "10.0.1.5:invalid_port")
	defer func() {
		os.Unsetenv("DIMD_INSTANCE_ID")
		os.Unsetenv("DIMD_CLUSTER_HOSTS")
	}()

	_, err := NewClusterConfigFromEnv()
	if err == nil {
		t.Errorf("expected error for invalid port")
	}
}

func TestSingleInstanceConfig(t *testing.T) {
	config := SingleInstanceConfig("dimd-1")

	if config.Mode != "single" {
		t.Errorf("expected mode=single, got %s", config.Mode)
	}

	if config.IsClustered() {
		t.Errorf("expected IsClustered=false for single instance")
	}

	if len(config.Instances) != 1 {
		t.Errorf("expected 1 instance, got %d", len(config.Instances))
	}
}
