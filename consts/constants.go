package consts

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
    // Namespace and Template names
    DefaultTemplateNamespace = "openshift"
    DefaultTemplateName = "rhel8-4-az-a"
)

const (
    // Tests Consts
    TestPrefix = "functional-test"
)

// DefaultResources defines the default resource requests and limits for VMs
var DefaultResources = corev1.ResourceRequirements{
    Requests: corev1.ResourceList{
        corev1.ResourceCPU:    resource.MustParse("500m"),
        corev1.ResourceMemory: resource.MustParse("2Gi"),
    },
    Limits: corev1.ResourceList{
        corev1.ResourceCPU:    resource.MustParse("1000m"),
        corev1.ResourceMemory: resource.MustParse("2Gi"),
    },
}

var DefaultLabels = map[string]string{
    "managed": "openshift-testing",
}