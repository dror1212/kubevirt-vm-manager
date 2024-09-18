package util

import (
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

// AuthenticateFile sets up the Kubernetes client using a provided kubeconfig file.
// If the file doesn't exist, it returns an error.
func AuthenticateFile(kubeconfigPath string) (*kubernetes.Clientset, *rest.Config, error) {
	var config *rest.Config
	var err error

	// Check if the kubeconfig file exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("kubeconfig file %s does not exist", kubeconfigPath)
	}

	// Load the kubeconfig file
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	// Create the Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return clientset, config, nil
}

// VerifyConnection checks if the Kubernetes client can successfully communicate with the cluster.
// It attempts to list namespaces to verify the connection.
func VerifyConnection(clientset *kubernetes.Clientset) error {
	// Attempt to list namespaces to verify connection
	_, err := clientset.CoreV1().Namespaces().List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to verify connection: %v", err)
	}
	return nil
}