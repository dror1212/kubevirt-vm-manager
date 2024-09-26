package util

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GeneratePort(name string, port, targetPort int, protocol string) corev1.ServicePort {
	// Set the protocol type based on the input
	var protocolType corev1.Protocol
	switch protocol {
	case "TCP":
		protocolType = corev1.ProtocolTCP
	case "UDP":
		protocolType = corev1.ProtocolUDP
	default:
		protocolType = corev1.ProtocolTCP // Default to TCP if protocol is not recognized
	}

	// Create and return the ServicePort
	return corev1.ServicePort{
		Name:       name,
		Port:       int32(port),               // Port should be of type int32
		TargetPort: intstr.FromInt(targetPort), // TargetPort is an IntOrString
		Protocol:   protocolType,              // Set the protocol
	}
}

// CreateService creates a Kubernetes service of a specified type (ClusterIP, NodePort, LoadBalancer)
func CreateService(clientset *kubernetes.Clientset, namespace, serviceName string, serviceType corev1.ServiceType, ports []corev1.ServicePort, labels map[string]string) (*corev1.Service, error) {
	// Use the service name as the default label if no labels are provided
	if labels == nil {
		labels = map[string]string{
			"app": serviceName,
		}
	}

	// Define the service object
	service := &corev1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     serviceType,
			Ports:    ports,
			Selector: labels, // Use the provided labels for the selector
		},
	}

	// Create the service in Kubernetes
	service, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), service, meta_v1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %v", err)
	}

	// If the service type is LoadBalancer, wait for the external IP to be assigned
	if serviceType == corev1.ServiceTypeLoadBalancer {
		LogInfo("Waiting for the LoadBalancer service %s to get an external IP...\n", serviceName)
		err := WaitForServiceReady(clientset, namespace, serviceName, 5*time.Second, 120*time.Second)
		if err != nil {
			return nil, fmt.Errorf("error waiting for service to be ready: %v", err)
		}

		// Fetch the service again to get the updated external IP
		service, err = clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, meta_v1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get service after waiting: %v", err)
		}

		LogInfo("Service %s is ready with external IP: %s\n", serviceName, service.Status.LoadBalancer.Ingress[0].IP)
	}

	return service, nil
}

// GetExternalIP fetches the external IP of a LoadBalancer service
func GetExternalIP(clientset *kubernetes.Clientset, namespace, serviceName string) (string, error) {
    service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, meta_v1.GetOptions{})
    if err != nil {
        return "", err
    }
    if len(service.Status.LoadBalancer.Ingress) > 0 {
        return service.Status.LoadBalancer.Ingress[0].IP, nil
    }
    return "", nil
}