package v1alpha1

// SecretReference TODO
type SecretRef struct {
	Name        string `json:"name"`
	KeyUsername string `json:"keyUsername"`
	KeyPassword string `json:"keyPassword"`
}
