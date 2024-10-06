package network_test

import (
	"time"
	"myproject/framework"
	"myproject/util"
	"myproject/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Service type LoadBalancer on Pod", func() {
	var (
		ctx         *framework.TestContext
		podName     string
		testPodName string
		serviceName string
		imageClient = "quay.med.one:8443/openshift/ubi8/ubi"
		image       = "quay.med.one:8443/openshift/httpd"
		externalIP  string
	)

	BeforeEach(func() {
		// Initialize the TestContext and setup environment
		ctx = framework.Setup("core")

		// Generate names for the pod, test pod, and service using the random name from context
		podName = consts.TestPrefix + "-server-" + ctx.RandomName
		testPodName = consts.TestPrefix + "-client-" + ctx.RandomName
		serviceName = consts.TestPrefix + "-lb-" + ctx.RandomName

		// Define the pod to be exposed by the LoadBalancer
		containers := []util.ContainerConfig{
			util.CreateContainerConfig("test-container", image, nil, util.GenerateResourceRequirements("250m", "1000m", "1Gi", "1Gi")),
		}

		// Create the main test pod
		err := ctx.CreateTestPodWithRetry(podName, containers, 20, 15*time.Second, 5*time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to create the main pod")

		// Create a LoadBalancer service for the pod
		servicePorts := []corev1.ServicePort{
			util.GeneratePort("http", 80, 80, "TCP"),
		}
		_, err = util.CreateService(ctx.Clientset, ctx.Namespace, serviceName, corev1.ServiceTypeLoadBalancer, servicePorts, map[string]string{"app": podName})
		Expect(err).ToNot(HaveOccurred(), "Failed to create LoadBalancer service")
	})

	It("should expose the service with a LoadBalancer and allow access to a pod from another pod", func() {
		// Wait for the service to get an external IP
		Eventually(func() (string, error) {
			var err error
			externalIP, err = util.GetExternalIP(ctx.Clientset, ctx.Namespace, serviceName)
			return externalIP, err
		}, 2*time.Minute, 10*time.Second).ShouldNot(BeEmpty(), "Expected service to get an external IP")

		// Define the test pod that will access the service
		testContainers := []util.ContainerConfig{
			util.CreateContainerConfig("curl-container", imageClient, []string{"curl", "--fail", "--retry", "5", "-w", "HTTP Response Code: %{http_code}\n", "http://" + externalIP}, util.GenerateResourceRequirements("100m", "400m", "200Mi", "200Mi")),
		}

		// Create the test pod using the retry mechanism
		err := ctx.CreateTestPodWithRetry(testPodName, testContainers, 20, 15*time.Second, 5*time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to create test client pod")

		// Wait for the test pod to complete and verify its status
		err = ctx.WaitForPodAndCheckLogs(testPodName, "HTTP Response Code: 200", 5*time.Second, 5*time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Expected to access the service successfully from another pod")
	})

	AfterEach(func() {
		// Clean up resources: Delete the test pods and the service
		ctx.CleanupResource(podName, "pod")
		ctx.CleanupResource(testPodName, "pod")
		ctx.CleanupResource(serviceName, "service")
	})
})
