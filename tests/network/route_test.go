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

var _ = Describe("Create Route with service and access from the cluster", func() {
	var (
		ctx         *framework.TestContext
		podName     string
		serviceName string
		routeName   string
		routeURL    string
		image       = "quay.med.one:8443/openshift/httpd" // HTTP server image
	)

	BeforeEach(func() {
		// Initialize the TestContext and setup environment
		ctx = framework.Setup("core")

		// Generate names for the pod, service, and route using the random name from context
		podName = consts.TestPrefix + "-server-" + ctx.RandomName
		serviceName = consts.TestPrefix + "-service-" + ctx.RandomName
		routeName = consts.TestPrefix + "-route-" + ctx.RandomName

		// Define the pod to be exposed by the service (runs an HTTP server)
		containers := []util.ContainerConfig{
			util.CreateContainerConfig("httpd-container", image, nil, util.GenerateResourceRequirements("250m", "1000m", "1Gi", "1Gi")),
		}

		// Create the main test pod
		ctx.CreateTestPodHelper(podName, containers)

		// Create a ClusterIP service for the pod
		servicePorts := []corev1.ServicePort{
			util.GeneratePort("http", 80, 80, "TCP"), // Port 80 for HTTP
		}
		ctx.CreateServiceHelper(serviceName, corev1.ServiceTypeClusterIP, servicePorts, map[string]string{"app": podName})

		// Create a Route for the service to expose it externally
		ctx.CreateRouteHelper(routeName, serviceName, 80, "")
	})

	It("should access the service via the route from within the cluster", func() {
		// Fetch the route URL and ensure it's ready
		Eventually(func() (string, error) {
			routeURL = ctx.GetRouteURLHelper(routeName)
			return routeURL, nil
		}, 2*time.Minute, 10*time.Second).ShouldNot(BeEmpty(), "Expected route to get a valid URL")

		// Define the test pod that will access the route using curl
		testContainers := []util.ContainerConfig{
			util.CreateContainerConfig("curl-container", "quay.med.one:8443/openshift/ubi8/ubi", []string{
				"curl", "--fail", "--retry", "5", "-w", "HTTP Response Code: %{http_code}\n", routeURL,
			}, util.GenerateResourceRequirements("100m", "400m", "200Mi", "200Mi")),
		}

		// Create the test pod to access the route
		testPodName := consts.TestPrefix + "-client-" + ctx.RandomName
		ctx.CreateTestPodHelper(testPodName, testContainers)

		// Verify that the test pod can access the route and get an HTTP 200 response
		ctx.VerifyPodAccess(testPodName, "HTTP Response Code: 200")
	})

	AfterEach(func() {
		// Clean up resources: Delete the route, pods, and services
		ctx.CleanupResource(podName, "pod")
		ctx.CleanupResource(serviceName, "service")
		ctx.CleanupResource(routeName, "route")
	})
})
