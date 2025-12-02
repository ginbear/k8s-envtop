package env

import (
	"context"
	"fmt"
	"sort"

	"github.com/ginbear/k8s-envtop/internal/k8s"
	corev1 "k8s.io/api/core/v1"
)

// Resolver resolves environment variables from Kubernetes workloads
type Resolver struct {
	client *k8s.Client
}

// NewResolver creates a new env resolver
func NewResolver(client *k8s.Client) *Resolver {
	return &Resolver{client: client}
}

// ResolveAppEnvVars resolves all environment variables for a given app
func (r *Resolver) ResolveAppEnvVars(ctx context.Context, app k8s.App) ([]k8s.EnvVar, error) {
	var podSpec *corev1.PodSpec

	switch app.Kind {
	case k8s.AppKindDeployment:
		deployment, err := r.client.GetDeployment(ctx, app.Namespace, app.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment %s: %w", app.Name, err)
		}
		podSpec = &deployment.Spec.Template.Spec
	case k8s.AppKindStatefulSet:
		statefulset, err := r.client.GetStatefulSet(ctx, app.Namespace, app.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get statefulset %s: %w", app.Name, err)
		}
		podSpec = &statefulset.Spec.Template.Spec
	default:
		return nil, fmt.Errorf("unsupported app kind: %s", app.Kind)
	}

	return r.resolveFromPodSpec(ctx, app.Namespace, podSpec)
}

// resolveFromPodSpec extracts env vars from a PodSpec
func (r *Resolver) resolveFromPodSpec(ctx context.Context, namespace string, podSpec *corev1.PodSpec) ([]k8s.EnvVar, error) {
	envVars := make([]k8s.EnvVar, 0)
	seen := make(map[string]bool)

	// Process all containers (including init containers)
	allContainers := append(podSpec.Containers, podSpec.InitContainers...)

	for _, container := range allContainers {
		// Process envFrom first
		for _, envFrom := range container.EnvFrom {
			vars, err := r.resolveEnvFrom(ctx, namespace, envFrom)
			if err != nil {
				// Log error but continue
				continue
			}
			for _, v := range vars {
				if !seen[v.Name] {
					seen[v.Name] = true
					envVars = append(envVars, v)
				}
			}
		}

		// Process env
		for _, env := range container.Env {
			v, err := r.resolveEnvVar(ctx, namespace, env)
			if err != nil {
				// Log error but continue
				continue
			}
			if !seen[v.Name] {
				seen[v.Name] = true
				envVars = append(envVars, v)
			}
		}
	}

	// Sort by name for consistent display
	sort.Slice(envVars, func(i, j int) bool {
		return envVars[i].Name < envVars[j].Name
	})

	return envVars, nil
}

// resolveEnvFrom resolves environment variables from envFrom sources
func (r *Resolver) resolveEnvFrom(ctx context.Context, namespace string, envFrom corev1.EnvFromSource) ([]k8s.EnvVar, error) {
	prefix := envFrom.Prefix
	vars := make([]k8s.EnvVar, 0)

	if envFrom.ConfigMapRef != nil {
		cm, err := r.client.GetConfigMap(ctx, namespace, envFrom.ConfigMapRef.Name)
		if err != nil {
			// Check if optional
			if envFrom.ConfigMapRef.Optional != nil && *envFrom.ConfigMapRef.Optional {
				return vars, nil
			}
			return nil, err
		}

		for key, value := range cm.Data {
			vars = append(vars, k8s.EnvVar{
				Name:       prefix + key,
				Value:      value,
				SourceName: cm.Name,
				SourceKind: k8s.EnvSourceConfigMap,
				ValueLen:   len(value),
			})
		}
	}

	if envFrom.SecretRef != nil {
		secret, err := r.client.GetSecret(ctx, namespace, envFrom.SecretRef.Name)
		if err != nil {
			// Check if optional
			if envFrom.SecretRef.Optional != nil && *envFrom.SecretRef.Optional {
				return vars, nil
			}
			return nil, err
		}

		// Check if this is a SealedSecret by looking for owner reference
		isSealed := r.isSealedSecret(ctx, namespace, secret.Name)

		for key, value := range secret.Data {
			sourceKind := k8s.EnvSourceSecret
			if isSealed {
				sourceKind = k8s.EnvSourceSealedSecret
			}
			vars = append(vars, k8s.EnvVar{
				Name:       prefix + key,
				RawValue:   value,
				Value:      fmt.Sprintf("HASH: %s", k8s.HashValue(value)),
				SourceName: secret.Name,
				SourceKind: sourceKind,
				IsSealed:   isSealed,
				ValueLen:   len(value),
				Hash:       k8s.HashValue(value),
			})
		}
	}

	return vars, nil
}

// resolveEnvVar resolves a single environment variable
func (r *Resolver) resolveEnvVar(ctx context.Context, namespace string, env corev1.EnvVar) (k8s.EnvVar, error) {
	// Inline value
	if env.Value != "" {
		return k8s.EnvVar{
			Name:       env.Name,
			Value:      env.Value,
			SourceKind: k8s.EnvSourceInline,
			ValueLen:   len(env.Value),
		}, nil
	}

	if env.ValueFrom == nil {
		return k8s.EnvVar{
			Name:       env.Name,
			Value:      "",
			SourceKind: k8s.EnvSourceInline,
		}, nil
	}

	// ConfigMap key reference
	if env.ValueFrom.ConfigMapKeyRef != nil {
		ref := env.ValueFrom.ConfigMapKeyRef
		cm, err := r.client.GetConfigMap(ctx, namespace, ref.Name)
		if err != nil {
			if ref.Optional != nil && *ref.Optional {
				return k8s.EnvVar{
					Name:       env.Name,
					Value:      "(optional, not found)",
					SourceName: ref.Name,
					SourceKind: k8s.EnvSourceConfigMap,
				}, nil
			}
			return k8s.EnvVar{}, err
		}

		value := cm.Data[ref.Key]
		return k8s.EnvVar{
			Name:       env.Name,
			Value:      value,
			SourceName: cm.Name,
			SourceKind: k8s.EnvSourceConfigMap,
			ValueLen:   len(value),
		}, nil
	}

	// Secret key reference
	if env.ValueFrom.SecretKeyRef != nil {
		ref := env.ValueFrom.SecretKeyRef
		secret, err := r.client.GetSecret(ctx, namespace, ref.Name)
		if err != nil {
			if ref.Optional != nil && *ref.Optional {
				return k8s.EnvVar{
					Name:       env.Name,
					Value:      "(optional, not found)",
					SourceName: ref.Name,
					SourceKind: k8s.EnvSourceSecret,
				}, nil
			}
			return k8s.EnvVar{}, err
		}

		value := secret.Data[ref.Key]
		isSealed := r.isSealedSecret(ctx, namespace, secret.Name)
		sourceKind := k8s.EnvSourceSecret
		if isSealed {
			sourceKind = k8s.EnvSourceSealedSecret
		}

		return k8s.EnvVar{
			Name:       env.Name,
			RawValue:   value,
			Value:      fmt.Sprintf("HASH: %s", k8s.HashValue(value)),
			SourceName: secret.Name,
			SourceKind: sourceKind,
			IsSealed:   isSealed,
			ValueLen:   len(value),
			Hash:       k8s.HashValue(value),
		}, nil
	}

	// Field reference (e.g., metadata.name)
	if env.ValueFrom.FieldRef != nil {
		return k8s.EnvVar{
			Name:       env.Name,
			Value:      fmt.Sprintf("fieldRef: %s", env.ValueFrom.FieldRef.FieldPath),
			SourceKind: k8s.EnvSourceFieldRef,
		}, nil
	}

	// Resource field reference (e.g., limits.cpu)
	if env.ValueFrom.ResourceFieldRef != nil {
		return k8s.EnvVar{
			Name:       env.Name,
			Value:      fmt.Sprintf("resourceFieldRef: %s", env.ValueFrom.ResourceFieldRef.Resource),
			SourceKind: k8s.EnvSourceResourceRef,
		}, nil
	}

	return k8s.EnvVar{
		Name:       env.Name,
		Value:      "(unknown source)",
		SourceKind: k8s.EnvSourceInline,
	}, nil
}

// isSealedSecret checks if a secret is managed by SealedSecret controller
func (r *Resolver) isSealedSecret(ctx context.Context, namespace, secretName string) bool {
	// Try to get the corresponding SealedSecret
	_, err := r.client.GetSealedSecret(ctx, namespace, secretName)
	return err == nil
}

// DiffResult represents a comparison result for a single env var
type DiffResult struct {
	Name      string
	EnvA      *k8s.EnvVar // nil if only in B
	EnvB      *k8s.EnvVar // nil if only in A
	Status    DiffStatus
}

// DiffStatus represents the comparison status
type DiffStatus string

const (
	DiffStatusSame      DiffStatus = "SAME"
	DiffStatusValueDiff DiffStatus = "VALUE_DIFF"
	DiffStatusOnlyInA   DiffStatus = "ONLY_IN_A"
	DiffStatusOnlyInB   DiffStatus = "ONLY_IN_B"
)

// CompareEnvVars compares two lists of env vars and returns the diff
func CompareEnvVars(envsA, envsB []k8s.EnvVar) []DiffResult {
	results := make([]DiffResult, 0)
	mapA := make(map[string]*k8s.EnvVar)
	mapB := make(map[string]*k8s.EnvVar)

	for i := range envsA {
		mapA[envsA[i].Name] = &envsA[i]
	}
	for i := range envsB {
		mapB[envsB[i].Name] = &envsB[i]
	}

	// Collect all unique names
	allNames := make(map[string]bool)
	for name := range mapA {
		allNames[name] = true
	}
	for name := range mapB {
		allNames[name] = true
	}

	// Convert to sorted slice
	names := make([]string, 0, len(allNames))
	for name := range allNames {
		names = append(names, name)
	}
	sort.Strings(names)

	// Compare
	for _, name := range names {
		a, hasA := mapA[name]
		b, hasB := mapB[name]

		result := DiffResult{Name: name, EnvA: a, EnvB: b}

		switch {
		case !hasA:
			result.Status = DiffStatusOnlyInB
		case !hasB:
			result.Status = DiffStatusOnlyInA
		case a.IsSecret() || b.IsSecret():
			// Compare by hash for secrets
			if a.Hash == b.Hash {
				result.Status = DiffStatusSame
			} else {
				result.Status = DiffStatusValueDiff
			}
		case a.Value == b.Value:
			result.Status = DiffStatusSame
		default:
			result.Status = DiffStatusValueDiff
		}

		results = append(results, result)
	}

	return results
}
