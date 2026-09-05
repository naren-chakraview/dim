package cluster

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Instance represents a single dimd instance in the cluster.
type Instance struct {
	ID        string    // Unique instance identifier
	Host      string    // Hostname or IP address
	Port      int       // Port number
	LastSeen  time.Time // Last heartbeat/health check
	Healthy   bool      // Current health status
	GenerationVersion int64 // Current generation version (for coordinated reload)
}

// ClusterConfig defines the cluster topology and discovery mechanism.
type ClusterConfig struct {
	Mode          string      // "static" (config-based), "env" (environment variables), or "coordinator" (etcd/Consul)
	Instances     []Instance  // List of cluster instances (for static mode)
	CurrentID     string      // This instance's unique ID
	HeartbeatTTL  time.Duration
	DiscoveryAddr string      // Coordinator address (for coordinator mode, optional)
}

// NewClusterConfigFromStatic creates a cluster config with static instance list.
// Used when cluster membership is defined in configuration.
func NewClusterConfigFromStatic(instances []Instance, currentID string) (*ClusterConfig, error) {
	if currentID == "" {
		return nil, fmt.Errorf("currentID is required")
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("at least one instance required in cluster")
	}

	// Verify current instance exists in list
	found := false
	for _, inst := range instances {
		if inst.ID == currentID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("current instance ID %q not found in cluster instances", currentID)
	}

	return &ClusterConfig{
		Mode:         "static",
		Instances:    instances,
		CurrentID:    currentID,
		HeartbeatTTL: 30 * time.Second,
	}, nil
}

// NewClusterConfigFromEnv creates a cluster config from environment variables.
// Expects:
// - DIMD_INSTANCE_ID: unique instance identifier
// - DIMD_CLUSTER_HOSTS: comma-separated list of host:port pairs (e.g., "10.0.1.5:8080,10.0.1.6:8080")
func NewClusterConfigFromEnv() (*ClusterConfig, error) {
	instanceID := os.Getenv("DIMD_INSTANCE_ID")
	if instanceID == "" {
		return nil, fmt.Errorf("DIMD_INSTANCE_ID environment variable not set")
	}

	hostsStr := os.Getenv("DIMD_CLUSTER_HOSTS")
	if hostsStr == "" {
		return nil, fmt.Errorf("DIMD_CLUSTER_HOSTS environment variable not set (format: host1:port1,host2:port2)")
	}

	hosts := strings.Split(hostsStr, ",")
	if len(hosts) == 0 {
		return nil, fmt.Errorf("DIMD_CLUSTER_HOSTS is empty")
	}

	instances := []Instance{}
	for i, hostPort := range hosts {
		hostPort = strings.TrimSpace(hostPort)
		parts := strings.Split(hostPort, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid DIMD_CLUSTER_HOSTS format %q (expected host:port)", hostPort)
		}

		host := parts[0]
		var port int
		_, err := fmt.Sscanf(parts[1], "%d", &port)
		if err != nil {
			return nil, fmt.Errorf("invalid port in DIMD_CLUSTER_HOSTS %q: %v", hostPort, err)
		}

		// Generate instance ID if not explicitly set (use host:port as fallback)
		id := fmt.Sprintf("dimd-%d", i)
		if i == 0 {
			// Use provided instance ID as first instance ID
			id = instanceID
		}

		instances = append(instances, Instance{
			ID:       id,
			Host:     host,
			Port:     port,
			Healthy:  true,
			LastSeen: time.Now(),
		})
	}

	// Verify current instance is in list (match by host:port if not exact ID match)
	found := false
	for _, inst := range instances {
		if inst.ID == instanceID {
			found = true
			break
		}
	}
	if !found && len(instances) > 0 {
		// First instance defaults to current ID
		instances[0].ID = instanceID
		found = true
	}

	if !found {
		return nil, fmt.Errorf("current instance ID %q not found in cluster", instanceID)
	}

	log.Printf("[INFO] cluster: loaded %d instances from environment, current instance: %s", len(instances), instanceID)

	return &ClusterConfig{
		Mode:         "env",
		Instances:    instances,
		CurrentID:    instanceID,
		HeartbeatTTL: 30 * time.Second,
	}, nil
}

// SingleInstanceConfig creates a non-clustered config (single instance only).
func SingleInstanceConfig(instanceID string) *ClusterConfig {
	return &ClusterConfig{
		Mode:      "single",
		CurrentID: instanceID,
		Instances: []Instance{
			{
				ID:      instanceID,
				Host:    "localhost",
				Port:    8080,
				Healthy: true,
			},
		},
		HeartbeatTTL: 30 * time.Second,
	}
}

// IsClustered returns true if this is a multi-instance cluster.
func (cc *ClusterConfig) IsClustered() bool {
	return len(cc.Instances) > 1
}

// CurrentInstance returns this instance's entry from the cluster config.
func (cc *ClusterConfig) CurrentInstance() *Instance {
	for i := range cc.Instances {
		if cc.Instances[i].ID == cc.CurrentID {
			return &cc.Instances[i]
		}
	}
	return nil
}

// OtherInstances returns all instances except the current one.
func (cc *ClusterConfig) OtherInstances() []Instance {
	others := []Instance{}
	for _, inst := range cc.Instances {
		if inst.ID != cc.CurrentID {
			others = append(others, inst)
		}
	}
	return others
}
