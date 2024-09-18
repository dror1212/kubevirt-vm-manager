package util

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	templateclientset "github.com/openshift/client-go/template/clientset/versioned"
	templatev1 "github.com/openshift/api/template/v1"
	"k8s.io/client-go/kubernetes"
	"kubevirt.io/client-go/kubecli"
)

// WaitForPodRunning waits for a Pod to reach the Running state.
func WaitForPodRunning(clientset *kubernetes.Clientset, namespace, podName string, interval, timeout time.Duration) error {
	return WaitFor(func() (bool, error) {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, meta_v1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pod.Status.Phase == corev1.PodRunning, nil
	}, interval, timeout)
}

// WaitForVMReady waits for a KubeVirt VM to be ready.
func WaitForVMReady(virtClient kubecli.KubevirtClient, namespace, vmName string, interval, timeout time.Duration) error {
	return WaitFor(func() (bool, error) {
		// Fetch the VM from KubeVirt API with context
		vm, err := virtClient.VirtualMachine(namespace).Get(context.TODO(), vmName, meta_v1.GetOptions{}) // Pass value, not pointer
		if err != nil {
			return false, err
		}

		// Check if the VM is running
		return vm.Status.Ready, nil
	}, interval, timeout)
}

// WaitForTemplateInstanceReady waits for an OpenShift TemplateInstance to be instantiated.
func WaitForTemplateInstanceReady(templateClient *templateclientset.Clientset, namespace, templateInstanceName string, interval, timeout time.Duration) error {
	return WaitFor(func() (bool, error) {
		// Fetch the TemplateInstance from OpenShift API
		templateInstance, err := templateClient.TemplateV1().TemplateInstances(namespace).Get(context.TODO(), templateInstanceName, meta_v1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check if the TemplateInstance has been processed and is ready
		for _, condition := range templateInstance.Status.Conditions {
			if condition.Type == templatev1.TemplateInstanceReady && condition.Status == "True" {
				return true, nil
			}
		}
		return false, nil
	}, interval, timeout)
}