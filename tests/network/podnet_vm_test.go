package network_test

import (
	"time"
	"myproject/framework"
	"myproject/util"
	"myproject/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Communicate with running VM using pod IP", func() {
	var (
		ctx         *framework.TestContext
		vmName      string
		testPodName string
		imageClient = "quay.med.one:8443/openshift/ubi8/ubi"
		scriptPath  = "../../scripts/httpd_install.sh"  // Path to the bash script
		vmPodIP     string
	)

	BeforeEach(func() {
		// Initialize the TestContext and setup environment
		ctx = framework.Setup("core")

		// Generate names for the VM and test pod using the random name from context
		vmName = consts.TestPrefix + ctx.RandomName
		testPodName = consts.TestPrefix + "-client-" + ctx.RandomName

		// Create the VM
		ctx.CreateTestVM(vmName, scriptPath, "")

		// Fetch the VM Pod IP
		Eventually(func() (string, error) {
			var err error
			vmPodIP, err = util.GetVMPodIP(ctx.VirtClient, ctx.Namespace, vmName)
			return vmPodIP, err
		}, 5*time.Minute, 10*time.Second).ShouldNot(BeEmpty(), "Expected VM to get a pod IP")
	})

	It("should communicate with the VM using pod IP from another pod", func() {
		// Use util.GenerateResourceRequirements to create resource requirements
		resources := util.GenerateResourceRequirements("100m", "400m", "200Mi", "200Mi")

		// Define the test pod that will access the VM's Pod IP
		testContainers := []util.ContainerConfig{
			util.CreateContainerConfig("curl-container", imageClient, []string{"curl", "--fail", "--retry", "5", "-w", "HTTP Response Code: %{http_code}\n", "http://" + vmPodIP + ":80"}, resources),
		}

		// Create the test pod using the helper function
		ctx.CreateTestPodHelper(testPodName, testContainers)

		// Verify the pod can access the VM using the helper function
		ctx.VerifyPodAccess(testPodName, "HTTP Response Code: 200")
	})

	AfterEach(func() {
		// Clean up resources: Delete the test pod and the VM
		ctx.CleanupResource(testPodName, "pod")
		ctx.CleanupResource(vmName, "vm")
	})
})
