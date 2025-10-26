package model

// K8sMeta is the Kubernetes simplified metadata used on different parts of Sloth logic like K8s storage.
type K8sMeta struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
}
