package util

import (
	"context"
	"time"
	"bytes"
	"io"

	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"myproject/consts"
)

// ContainerConfig defines the configuration for a container in a Pod
type ContainerConfig struct {
	Name      string
	Image     string
	Command   []string
	Args      []string
	Resources corev1.ResourceRequirements
}

// CreatePod creates a Pod with multiple containers specified by the ContainerConfig list.
func CreatePod(config *rest.Config, namespace, podName string, containerConfigs []ContainerConfig, labels map[string]string, waitForCreation bool) (*corev1.Pod, error) {
	if podName == "" {
		podName = GenerateRandomName()
		LogInfo("Generated random Pod name: %s", podName)
	}

	if labels == nil {
		labels = consts.DefaultLabels
		labels["app"] = podName
		LogInfo("Using default labels for VM: %s", podName)
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		LogError("Failed to create Kubernetes client: %v", err)
		return nil, err
	}

	// Generate containers from the container configurations
	containers := []corev1.Container{}
	for _, config := range containerConfigs {
		containers = append(containers, generateContainerFromConfig(config))
	}

	// Define the Pod object
	pod := &corev1.Pod{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers:    containers,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	// Create the Pod in Kubernetes
	createdPod, err := clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, meta_v1.CreateOptions{})
	if err != nil {
		LogError("Failed to create Pod: %v", err)
		return nil, err
	}

	LogInfo("Pod %s created successfully", podName)

	// Wait for the Pod to be ready, if requested
	if waitForCreation {
		err = WaitForPodRunning(clientset, namespace, podName, 5*time.Second, 120*time.Second)
		if err != nil {
			LogError("Error waiting for Pod to be ready: %v", err)
			return nil, err
		}
		LogInfo("Pod %s is ready", podName)
	}

	return createdPod, nil
}

// GetPodLogs fetches the logs from a specific pod
func GetPodLogs(clientset *kubernetes.Clientset, namespace, podName string) (string, error) {
    podLogOpts := corev1.PodLogOptions{}
    podLogRequest := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)
    podLogs, err := podLogRequest.Stream(context.TODO())
    if err != nil {
        return "", err
    }
    defer podLogs.Close()

    buf := new(bytes.Buffer)
    _, err = io.Copy(buf, podLogs)
    if err != nil {
        return "", err
    }
    return buf.String(), nil
}

// generateContainerFromConfig creates a container spec from the given ContainerConfig
func generateContainerFromConfig(config ContainerConfig) corev1.Container {
	return corev1.Container{
		Name:      config.Name,
		Image:     config.Image,
		Command:   config.Command,
		Args:      config.Args,
		Resources: config.Resources,
	}
}
