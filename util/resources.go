package util

import (
	corev1 "k8s.io/api/core/v1"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// parseQuantity parses a string into a resource.Quantity and logs an error if parsing fails.
func parseQuantity(value string, resourceName string) *resource.Quantity {
	if value == "" {
		return nil // No value provided, omit the attribute
	}
	parsedQuantity, err := resource.ParseQuantity(value)
	if err != nil {
		return nil // Parsing failed, omit the attribute
	}
	return &parsedQuantity
}

// GenerateResourceRequirements generates Kubernetes corev1.ResourceRequirements for Kubernetes objects.
func GenerateResourceRequirements(cpuRequests string, cpuLimits string, memoryRequests string, memoryLimits string) corev1.ResourceRequirements {
	resourceRequirements := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}

	if cpuLimit := parseQuantity(cpuLimits, "CPU Limit"); cpuLimit != nil {
		resourceRequirements.Limits[corev1.ResourceCPU] = *cpuLimit
	}

	if memoryLimit := parseQuantity(memoryLimits, "Memory Limit"); memoryLimit != nil {
		resourceRequirements.Limits[corev1.ResourceMemory] = *memoryLimit
	}

	if cpuRequest := parseQuantity(cpuRequests, "CPU Request"); cpuRequest != nil {
		resourceRequirements.Requests[corev1.ResourceCPU] = *cpuRequest
	}

	if memoryRequest := parseQuantity(memoryRequests, "Memory Request"); memoryRequest != nil {
		resourceRequirements.Requests[corev1.ResourceMemory] = *memoryRequest
	}

	return resourceRequirements
}

// ConvertCoreV1ToKubeVirtResourceRequirements converts corev1.ResourceRequirements to kubevirtv1.ResourceRequirements
func ConvertCoreV1ToKubeVirtResourceRequirements(coreReq corev1.ResourceRequirements) kubevirtv1.ResourceRequirements {
	return kubevirtv1.ResourceRequirements{
		Limits:   coreReq.Limits,
		Requests: coreReq.Requests,
	}
}
