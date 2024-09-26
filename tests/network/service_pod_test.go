package network_test

import (
	"time"
	"context"
	"strings"
	"myproject/util"
	"myproject/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
    meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service type LoadBalancer on Pod", func() {
	var (
		randomName 	 = util.GenerateRandomName()
		clientset    *kubernetes.Clientset
		config       *rest.Config
		namespace    = "core"
		podName      = consts.TestPrefix + "-server-" + randomName
		testPodName  = consts.TestPrefix + "-client-" + randomName
		serviceName  = consts.TestPrefix + "-lb-" + randomName
		imageClient  = "quay.med.one:8443/openshift/ubi8/ubi"
		image        = "quay.med.one:8443/openshift/httpd"
		externalIP   string
	)

	BeforeEach(func() {
		var err error

		err = util.SetLogLevel("debug")
		Expect(err).ToNot(HaveOccurred(), "Failed to initiate the logger")

		clientset, config, err = util.Authenticate()
		Expect(err).ToNot(HaveOccurred(), "Failed to authenticate with Kubernetes")

		// Define the pod to be exposed by LoadBalancer
		containers := []util.ContainerConfig{
			{
				Name:  "test-container",
				Image: image,
				Resources: util.GenerateResourceRequirements("250m", "1000m", "1Gi", "1Gi"),
			},
		}

		// Create the main test pod
		_, err = util.CreatePod(config, namespace, podName, containers, nil, true)
		Expect(err).ToNot(HaveOccurred(), "Failed to create the main pod")

		// Create a LoadBalancer service for the pod
		servicePorts := []corev1.ServicePort{
			util.GeneratePort("http", 80, 80, "TCP"),
		}
		_, err = util.CreateService(clientset, namespace, serviceName, corev1.ServiceTypeLoadBalancer, servicePorts, map[string]string{"app": podName})
		Expect(err).ToNot(HaveOccurred(), "Failed to create LoadBalancer service")
	})

	It("should expose the service with a LoadBalancer and allow access to a pod from another pod", func() {
		// Wait for the service to get an external IP
		Eventually(func() (string, error) {
			var err error
			externalIP, err = util.GetExternalIP(clientset, namespace, serviceName)
			return externalIP, err
		}, 2*time.Minute, 10*time.Second).ShouldNot(BeEmpty(), "Expected service to get an external IP")

		// Define the test pod that will access the service
		testContainers := []util.ContainerConfig{
			{
				Name:    "curl-container",
				Image:   imageClient,
				Command: []string{"curl", "--fail", "--retry", "5", "-w", "HTTP Response Code: %{http_code}\n", "http://" + externalIP},
				Resources: util.GenerateResourceRequirements("100m", "400m", "200Mi", "200Mi"),
			},
		}

		// Create the test client pod
		_, err := util.CreatePod(config, namespace, testPodName, testContainers, nil, true)
		Expect(err).ToNot(HaveOccurred(), "Failed to create test client pod")

		// Wait for the test pod to complete and verify its status
		Eventually(func() (bool, error) {
			err := util.WaitForPodCompletionOrFailure(clientset, namespace, testPodName, 5*time.Second, 2*time.Minute)
			if err != nil {
				return false, err
			}

			// Fetch pod logs and verify successful HTTP request (curl should return "200 OK")
			podLogs, err := util.GetPodLogs(clientset, namespace, testPodName) // Implement this function
			if err != nil {
				return false, err
			}
			return strings.Contains(podLogs, "HTTP Response Code: 200"), nil
		}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to access the service successfully from another pod")
	})

	AfterEach(func() {
		// Clean up resources: Delete the test pods and the service
		err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), podName, meta_v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete pod %s", podName)

		err = clientset.CoreV1().Pods(namespace).Delete(context.TODO(), testPodName, meta_v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete test client pod %s", testPodName)

		err = clientset.CoreV1().Services(namespace).Delete(context.TODO(), serviceName, meta_v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete LoadBalancer service %s", serviceName)

		util.LogInfo("Successfully cleaned up resources.")
	})
})
