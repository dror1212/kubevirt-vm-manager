package network_test

import (
	"strings"
	"time"
	"context"
	"myproject/util"
	"myproject/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var _ = Describe("Communicate with running VM using pod IP", func() {
	var (
		randomName 	 = util.GenerateRandomName()
		clientset    *kubernetes.Clientset
		virtClient   kubecli.KubevirtClient
		config       *rest.Config
		namespace    = "core"
		vmName       = consts.TestPrefix + randomName
		testPodName  = consts.TestPrefix + "-client-" + randomName
		imageClient  = "quay.med.one:8443/openshift/ubi8/ubi"
		vmPodIP      string
		scriptPath   = "../../scripts/httpd_install.sh"  // Path to the bash script
	)

	BeforeEach(func() {
		var err error

		err = util.SetLogLevel("debug")
		Expect(err).ToNot(HaveOccurred(), "Failed to initiate the logger")

		clientset, config, err = util.Authenticate()
		Expect(err).ToNot(HaveOccurred(), "Failed to authenticate with Kubernetes")

		virtClient, err = kubecli.GetKubevirtClientFromRESTConfig(config)
		Expect(err).ToNot(HaveOccurred(), "Failed to authenticate with KubeVirt")

		// Create the VM with the bash script passed as a command to be executed
		resourceRequirements := util.ConvertCoreV1ToKubeVirtResourceRequirements(util.GenerateResourceRequirements("4000m", "4000m", "4Gi", "4Gi"))
		_, err = util.CreateVM(config, namespace, "rhel8-4-az-a", vmName, &resourceRequirements, nil, true, scriptPath, "")
		Expect(err).ToNot(HaveOccurred(), "Failed to create VM")

		// Fetch the VM Pod IP
		Eventually(func() (string, error) {
			vmPodIP, err = util.GetVMPodIP(virtClient, namespace, vmName)
			return vmPodIP, err
		}, 5*time.Minute, 10*time.Second).ShouldNot(BeEmpty(), "Expected VM to get a pod IP")
	})

	It("should communicate with the VM using pod IP from another pod", func() {
		// Define the test pod that will access the VM's Pod IP
		testContainers := []util.ContainerConfig{
			{
				Name:    "curl-container",
				Image:   imageClient,
				Command: []string{"curl", "--fail", "--retry", "5", "-w", "HTTP Response Code: %{http_code}\n", "http://" + vmPodIP + ":80"},
				Resources: util.GenerateResourceRequirements("100m", "400m", "200Mi", "200Mi"),
			},
		}
	
		// Retry pod creation using the WaitFor-based retry mechanism
		_, err := util.RetryPodCreationWithWait(clientset, config, namespace, testPodName, testContainers, nil, 20, 15*time.Second, 5*time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Failed to create test client pod after retries")
	
		// Wait for the test pod to complete and verify its status
		Eventually(func() (bool, error) {
			err := util.WaitForPodCompletionOrFailure(clientset, namespace, testPodName, 5*time.Second, 2*time.Minute)
			if err != nil {
				return false, err
			}
	
			// Fetch pod logs and verify successful HTTP request (curl should return "200 OK")
			podLogs, err := util.GetPodLogs(clientset, namespace, testPodName)
			if err != nil {
				return false, err
			}
			return strings.Contains(podLogs, "HTTP Response Code: 200"), nil
		}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "Expected to access the VM successfully from another pod")
	})	

	AfterEach(func() {
		// Clean up resources: Delete the test pod and the VM
		err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), testPodName, meta_v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete test client pod %s", testPodName)

		err = virtClient.VirtualMachine(namespace).Delete(context.TODO(), vmName, meta_v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete VM %s", vmName)

		util.LogInfo("Successfully cleaned up resources.")
	})
})
