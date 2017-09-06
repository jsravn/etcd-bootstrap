package cloud

// Cloud represents the members of an etcd cluster.
type Cloud interface {
	// GetInstances returns all the non-terminated instances that will be part of the etcd cluster.
	GetInstances() []Instance
	// GetLocalInstance returns the local machine instance.
	GetLocalInstance() Instance
	// UpdateDNS updates the DNS provider for this cloud to add records for every member of the etcd cluster.
	UpdateDNS(name string) error
}

// Instance represents an instance inside of the auto scaling group.
type Instance struct {
	InstanceID string
	PrivateIP  string
}