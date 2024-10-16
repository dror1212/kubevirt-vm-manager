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

var _ = Describe("NetworkPolicy to restrict access to ports", func() {
	var (
		ctx           *framework.TestContext
		ctxHelper     *framework.TestContext
		serverPodName string
		clientPodName string
		serviceName   string
		policyName    string
		serviceIP     string
		imageClient   = consts.ClientImage
		image         = consts.HttpdImage
	)

	BeforeEach(func() {
		// Initialize the TestContext and setup environment in the current namespace
		ctx = framework.Setup("core")
		ctxHelper = framework.Setup("test-4")

		// Generate names for the pod, test pod, service, and network policy using the random name from context
		serverPodName = consts.TestPrefix + "-server-" + ctx.RandomName
		clientPodName = consts.TestPrefix + "-client-" + ctx.RandomName
		serviceName = consts.TestPrefix + "-lb-" + ctx.RandomName
		policyName = consts.TestPrefix + "-np-" + ctx.RandomName

		// Define the pod to be exposed by the ClusterIP service
		containers := []util.ContainerConfig{
			util.CreateContainerConfig("test-container", image, nil, util.GenerateResourceRequirements("250m", "1000m", "1Gi", "1Gi")),
		}

		// Create the main test pod
		ctx.CreateTestPodHelper(serverPodName, containers, 3)

		// Create a ClusterIP service for the pod
		servicePorts := []corev1.ServicePort{
			util.GeneratePort("http", 80, 80, "TCP"),
		}

		ctx.CreateServiceHelper(serviceName, "ClusterIP", servicePorts, map[string]string{"app": serverPodName})
	})

	FIt("should deny traffic from other namespaces on port 80 before applying NetworkPolicy, then allow after applying NetworkPolicy", func() {
		// Fetch the service IP
		serviceIP = ctx.WaitForServiceIP(serviceName, 2*time.Minute, 10*time.Second)

		// Define the test pod that will access the service from another namespace
		testContainers := []util.ContainerConfig{
			util.CreateContainerConfig("curl-container", imageClient, []string{"curl", "--max-time", "5", "-w", "HTTP Response Code: %{http_code}\n", "http://" + serviceIP}, util.GenerateResourceRequirements("100m", "400m", "200Mi", "200Mi")),
		}

		// Create the test pod in a different namespace, expecting it to fail due to lack of NetworkPolicy
		ctxHelper.CreateTestPodExpectingFailureHelper(clientPodName, testContainers, 3)

		// Verify that access is denied (i.e., "HTTP Response Code: 000" or no response)
		ctxHelper.VerifyPodResponse(clientPodName, "HTTP Response Code: 000", 3)

		// Clean up the failing pod
		ctxHelper.CleanupResource(clientPodName, "pod")

		// Apply the NetworkPolicy to allow traffic from other namespaces on port 80
		networkPorts := util.CreateNetworkPolicyPort(80, "TCP")
		_, err := util.CreateNetworkPolicyWithNamespaceAllow(ctx.KubeClient, ctx.Namespace, policyName, networkPorts)
		Expect(err).ToNot(HaveOccurred(), "Failed to create network policy with allow rule")

		// Wait a bit for the policy to take effect
		time.Sleep(10 * time.Second)

		// Create the test pod again, after applying the NetworkPolicy
		ctxHelper.CreateTestPodHelper(clientPodName, testContainers, 3)

		// Verify that access is allowed after applying NetworkPolicy
		ctxHelper.VerifyPodResponse(clientPodName, "HTTP Response Code: 200", 3)
	})

	AfterEach(func() {
		// Clean up resources: Delete pods, services, and network policies
		ctxHelper.CleanupResource(clientPodName, "pod")
		ctx.CleanupResource(serverPodName, "pod")
		ctx.CleanupResource(serviceName, "service")
		ctx.CleanupResource(policyName, "networkPolicy")
	})
})
