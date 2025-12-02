package k8s

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes client operations
type Client struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	context       string
}

// NewClient creates a new Kubernetes client using kubeconfig
func NewClient() (*Client, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Get current context name
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfig
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw config: %w", err)
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		context:       rawConfig.CurrentContext,
	}, nil
}

// GetCurrentContext returns the current Kubernetes context name
func (c *Client) GetCurrentContext() string {
	return c.context
}

// ListNamespaces returns a list of all namespaces
func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	nsList, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaces := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Name)
	}
	return namespaces, nil
}

// ListApps returns a list of Deployments and StatefulSets in the given namespace
func (c *Client) ListApps(ctx context.Context, namespace string) ([]App, error) {
	apps := make([]App, 0)

	// List Deployments
	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	for _, d := range deployments.Items {
		apps = append(apps, App{
			Name:      d.Name,
			Namespace: namespace,
			Kind:      AppKindDeployment,
		})
	}

	// List StatefulSets
	statefulsets, err := c.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}
	for _, s := range statefulsets.Items {
		apps = append(apps, App{
			Name:      s.Name,
			Namespace: namespace,
			Kind:      AppKindStatefulSet,
		})
	}

	return apps, nil
}

// GetDeployment returns a Deployment by name
func (c *Client) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetStatefulSet returns a StatefulSet by name
func (c *Client) GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetConfigMap returns a ConfigMap by name
func (c *Client) GetConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetSecret returns a Secret by name
func (c *Client) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	return c.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// SealedSecretGVR is the GroupVersionResource for SealedSecrets
var SealedSecretGVR = schema.GroupVersionResource{
	Group:    "bitnami.com",
	Version:  "v1alpha1",
	Resource: "sealedsecrets",
}

// GetSealedSecret returns a SealedSecret by name
func (c *Client) GetSealedSecret(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(SealedSecretGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// IsSealedSecretAvailable checks if SealedSecret CRD is available in the cluster
func (c *Client) IsSealedSecretAvailable(ctx context.Context) bool {
	_, err := c.dynamicClient.Resource(SealedSecretGVR).List(ctx, metav1.ListOptions{Limit: 1})
	return err == nil
}

// HashValue returns a SHA256 hash prefix of the given value
func HashValue(value []byte) string {
	hash := sha256.Sum256(value)
	return fmt.Sprintf("%x", hash[:4]) // First 8 hex characters
}

// DecodeBase64 decodes a base64 encoded string
func DecodeBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// EncodeBase64 encodes bytes to base64 string
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
