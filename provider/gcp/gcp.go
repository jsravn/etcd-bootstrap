package gcp

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/sky-uk/etcd-bootstrap/provider"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
)

// Config is the configuration required to talk to GCP APIs to fetch a list of nodes
type Config struct {
	// ProjectID is the name of the project to query
	ProjectID string
	// Environment tag to filter by
	Environment string
	// Role tag to filter by
	Role string
}

type gcpMembers struct {
	instances []provider.Instance
	instance  provider.Instance
}

// GetInstances will return the gcp etcd instances
func (m *gcpMembers) GetInstances() []provider.Instance {
	return m.instances
}

// GetLocalInstance will get the gcp instance etcd bootstrap is running on
func (m *gcpMembers) GetLocalInstance() provider.Instance {
	return m.instance
}

// NewGCP returns the Members matching the cfg.
func NewGCP(cfg *Config) (provider.Provider, error) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	c, err := newClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create GCP compute API client: %v", err)
	}

	instances, err := findAllInstances(c, cfg)
	if err != nil {
		return nil, err
	}

	instance, err := findThisInstance()
	if err != nil {
		return nil, err
	}

	members := &gcpMembers{
		instances: instances,
		instance:  *instance,
	}

	return members, nil
}

func findThisInstance() (*provider.Instance, error) {
	ip, err := metadata.InternalIP()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve local IP metadata: %v", err)
	}
	name, err := metadata.InstanceName()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve local Name metadata: %v", err)
	}
	local := &provider.Instance{
		InstanceID: name,
		PrivateIP:  ip,
	}
	return local, nil
}

func newClient(ctx context.Context, cfg *Config) (*compute.Service, error) {
	client, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	computeService, err := compute.New(client)
	if err != nil {
		return nil, err
	}
	return computeService, err
}

func findAllInstances(client *compute.Service, cfg *Config) ([]provider.Instance, error) {
	zones, err := client.Zones.List(cfg.ProjectID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list zones for project %q: %v", cfg.ProjectID, err)
	}

	var instances []provider.Instance
	for _, zone := range zones.Items {
		// https://provider.google.com/sdk/gcloud/reference/topic/filters
		filters := []string{
			fmt.Sprintf("labels.environment=%s", cfg.Environment),
			fmt.Sprintf("labels.role=%s", cfg.Role),
			"status != TERMINATED",
		}
		byEnvironmentAndRole := fmt.Sprintf(strings.Join(filters, " AND "))
		result, err := client.Instances.List(cfg.ProjectID, zone.Name).Filter(byEnvironmentAndRole).Do()
		if err != nil {
			return nil, fmt.Errorf("unable to list instances for project %q, zone %v: %v", cfg.ProjectID, zone, err)
		}

		for _, instance := range result.Items {
			// Taking the first available network interface in case there are multiple.
			// The networkInterface.NetworkIP will only contain private IPs:
			// https://provider.google.com/compute/docs/reference/rest/v1/instances/list
			if len(instance.NetworkInterfaces) > 0 {
				instances = append(instances, provider.Instance{
					InstanceID: instance.Name,
					PrivateIP:  instance.NetworkInterfaces[0].NetworkIP,
				})
			} else {
				return nil, fmt.Errorf("unable to find network interfaces for instance %q", instance.Name)
			}
		}
	}
	return instances, nil
}