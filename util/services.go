package util

import (
	"context"
	"fmt"

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
			Type:  serviceType,
			Ports: ports,
			Selector: labels, // Use the provided labels for the selector
		},
	}

	// Create the service in Kubernetes
	service, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), service, meta_v1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %v", err)
	}

	return service, nil
}