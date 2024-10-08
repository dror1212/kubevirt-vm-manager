package util

import (
	"context"
	"time"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	templateclientset "github.com/openshift/client-go/template/clientset/versioned"
	templatev1 "github.com/openshift/api/template/v1"
	"k8s.io/client-go/kubernetes"
	"kubevirt.io/client-go/kubecli"
)

func WaitForPodCompletionOrFailure(clientset *kubernetes.Clientset, namespace, podName string, interval, timeout time.Duration) error {
	return WaitFor(func() (bool, error) {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			LogError("Error fetching pod: %v", err)
			return false, err
		}

		// Check if the pod is either completed successfully or failed
		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			LogInfo("Pod %s has completed successfully.", podName)
			return true, nil
		case corev1.PodFailed:
			LogError("Pod %s has failed.", podName)
			return true, fmt.Errorf("pod %s has failed", podName) // Return true here to stop the waiting
		}

		// Log the current phase of the pod if it hasn't completed
		LogInfo("Pod %s is in phase: %s", podName, pod.Status.Phase)
		return false, nil
	}, interval, timeout)
}

// WaitForPodRunning waits for a Pod to reach the Running or Completed state.
func WaitForPodRunning(clientset *kubernetes.Clientset, namespace, podName string, interval, timeout time.Duration) error {
	return WaitFor(func() (bool, error) {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			LogError("Error fetching pod: %v", err)
			return false, err
		}
		switch pod.Status.Phase {
		case corev1.PodRunning:
			LogInfo("Pod %s is now running.", podName)
			return true, nil
		case corev1.PodSucceeded:
			LogInfo("Pod %s has completed.", podName)
			return true, nil
		case corev1.PodFailed:
			LogError("Pod %s has failed.", podName)
			return false, fmt.Errorf("pod %s has failed", podName)
		default:
			LogInfo("Pod %s is in phase %s.", podName, pod.Status.Phase)
			return false, nil
		}
	}, interval, timeout)
}

// WaitForVMReady waits for a KubeVirt VM to be ready.
func WaitForVMReady(virtClient kubecli.KubevirtClient, namespace, vmName string, interval, timeout time.Duration) error {
	return WaitFor(func() (bool, error) {
		vm, err := virtClient.VirtualMachine(namespace).Get(context.TODO(), vmName, metav1.GetOptions{})
		if err != nil {
			LogError("Error fetching VM: %v", err)
			return false, err
		}

		if vm.Status.Ready {
			LogInfo("VM %s is ready.", vmName)
		}

		return vm.Status.Ready, nil
	}, interval, timeout)
}

// WaitForTemplateInstanceReady waits for an OpenShift TemplateInstance to be instantiated.
func WaitForTemplateInstanceReady(templateClient *templateclientset.Clientset, namespace, templateInstanceName string, interval, timeout time.Duration) error {
	return WaitFor(func() (bool, error) {
		templateInstance, err := templateClient.TemplateV1().TemplateInstances(namespace).Get(context.TODO(), templateInstanceName, metav1.GetOptions{})
		if err != nil {
			LogError("Error fetching TemplateInstance: %v", err)
			return false, err
		}

		for _, condition := range templateInstance.Status.Conditions {
			if condition.Type == templatev1.TemplateInstanceReady && condition.Status == "True" {
				LogInfo("TemplateInstance %s is ready.", templateInstanceName)
				return true, nil
			}
		}
		return false, nil
	}, interval, timeout)
}

// WaitForServiceReady waits for a LoadBalancer service to have an external IP assigned.
func WaitForServiceReady(clientset *kubernetes.Clientset, namespace, serviceName string, interval, timeout time.Duration) error {
	return WaitFor(func() (bool, error) {
		service, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			LogError("Error fetching service: %v", err)
			return false, err
		}

		// For LoadBalancer type services, check if an external IP is assigned
		if len(service.Status.LoadBalancer.Ingress) > 0 && service.Status.LoadBalancer.Ingress[0].IP != "" {
			LogInfo("Service %s has an external IP: %s", serviceName, service.Status.LoadBalancer.Ingress[0].IP)
			return true, nil
		}

		LogInfo("Waiting for service %s to get an external IP...", serviceName)
		return false, nil
	}, interval, timeout)
}
