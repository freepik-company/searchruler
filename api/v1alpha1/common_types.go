package v1alpha1

// SecretRef TODO
type SecretRef struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	KeyUsername string `json:"keyUsername"`
	KeyPassword string `json:"keyPassword"`
}
