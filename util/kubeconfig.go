package util

import (
	"os"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Authenticate sets up the Kubernetes client using in-cluster config or a kubeconfig file.
// It first attempts to use in-cluster config (for running inside Kubernetes), and if that fails,
// it falls back to using the kubeconfig file from the environment or home directory.
func Authenticate() (*kubernetes.Clientset, *rest.Config, error) {
	var config *rest.Config
	var err error

	// Try in-cluster configuration (for use inside Kubernetes)
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to using kubeconfig from the environment or home directory
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath != "" {
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		} else {
			// Use default kubeconfig in the home directory
			config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		}
		if err != nil {
			return nil, nil, err
		}
	}

	// Create the Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return clientset, config, nil
}
