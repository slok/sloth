package storage

// K8sMeta is the Kubernetes metadata simplified used for storage purposes.
type K8sMeta struct {
	Kind        string `validate:"required"`
	APIVersion  string `validate:"required"`
	Name        string `validate:"required"`
	UID         string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
}
