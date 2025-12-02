package k8s

// AppKind represents the type of Kubernetes workload
type AppKind string

const (
	AppKindDeployment  AppKind = "Deployment"
	AppKindStatefulSet AppKind = "StatefulSet"
)

// App represents a Kubernetes workload (Deployment/StatefulSet)
type App struct {
	Name      string
	Namespace string
	Kind      AppKind
}

// EnvSourceKind represents the source type of an environment variable
type EnvSourceKind string

const (
	EnvSourceConfigMap     EnvSourceKind = "ConfigMap"
	EnvSourceSecret        EnvSourceKind = "Secret"
	EnvSourceSealedSecret  EnvSourceKind = "SealedSecret"
	EnvSourceFieldRef      EnvSourceKind = "FieldRef"
	EnvSourceResourceRef   EnvSourceKind = "ResourceRef"
	EnvSourceInline        EnvSourceKind = "Inline"
)

// EnvVar represents an environment variable with its source information
type EnvVar struct {
	Name       string
	Value      string        // actual value for ConfigMap/Inline, hash for Secret/SealedSecret
	RawValue   []byte        // raw value (base64 decoded) for secrets
	SourceName string        // name of the ConfigMap/Secret
	SourceKind EnvSourceKind
	IsSealed   bool
	ValueLen   int
	Hash       string        // SHA256 hash prefix for secrets
}

// IsSecret returns true if the env var comes from a Secret or SealedSecret
func (e *EnvVar) IsSecret() bool {
	return e.SourceKind == EnvSourceSecret || e.SourceKind == EnvSourceSealedSecret
}
