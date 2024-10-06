package framework

import (
	"context"
	"strings"
	"time"
	"myproject/util"
	"myproject/consts"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubecli "kubevirt.io/client-go/kubecli"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestContext holds reusable values across tests, including a random name for resources
type TestContext struct {
	Clientset  *kubernetes.Clientset
	Config     *rest.Config
	VirtClient kubecli.KubevirtClient
	Namespace  string
	RandomName string
}

// Setup initializes the environment (e.g., auth, logging) and sets the random name for each test
func Setup(namespace string) *TestContext {
	var err error

	err = util.SetLogLevel("debug")
	Expect(err).ToNot(HaveOccurred(), "Failed to initiate the logger")

	clientset, config, err := util.Authenticate()
	Expect(err).ToNot(HaveOccurred(), "Failed to authenticate with Kubernetes")

	virtClient, err := kubecli.GetKubevirtClientFromRESTConfig(config)
	Expect(err).ToNot(HaveOccurred(), "Failed to authenticate with KubeVirt")

	randomName := util.GenerateRandomName()

	return &TestContext{
		Clientset:  clientset,
		Config:     config,
		VirtClient: virtClient,
		Namespace:  namespace,
		RandomName: randomName,
	}
}

// Cleanup cleans up resources such as VMs and Pods
func (ctx *TestContext) CleanupResource(resourceName string, resourceType string) {
	switch resourceType {
	case "pod":
		err := ctx.Clientset.CoreV1().Pods(ctx.Namespace).Delete(context.TODO(), resourceName, meta_v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete pod %s", resourceName)
	case "vm":
		err := ctx.VirtClient.VirtualMachine(ctx.Namespace).Delete(context.TODO(), resourceName, meta_v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete VM %s", resourceName)
	case "service":
		err := ctx.Clientset.CoreV1().Services(ctx.Namespace).Delete(context.TODO(), resourceName, meta_v1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Failed to delete service %s", resourceName)
	}
}

// CreateTestVM creates a VM with default settings using the random name from the context
func (ctx *TestContext) CreateTestVM(scriptPath string, templateName string) {
	vmName := consts.TestPrefix + ctx.RandomName
	resourceRequirements := util.ConvertCoreV1ToKubeVirtResourceRequirements(
		util.GenerateResourceRequirements("4000m", "4000m", "4Gi", "4Gi"))

	_, err := util.CreateVM(ctx.Config, ctx.Namespace, templateName, vmName, &resourceRequirements, nil, true, scriptPath, "")
	Expect(err).ToNot(HaveOccurred(), "Failed to create VM")

	// Optionally return the VM name for other uses
}

// CreateTestPod creates a test pod using the random name from the context
func (ctx *TestContext) CreateTestPod(image string, containerConfig []util.ContainerConfig) string {
	podName := consts.TestPrefix + "-client-" + ctx.RandomName
	_, err := util.CreatePod(ctx.Config, ctx.Namespace, podName, containerConfig, nil, true)
	Expect(err).ToNot(HaveOccurred(), "Failed to create pod")
	return podName
}

// CreateTestPodWithRetry creates a test pod with retries
func (ctx *TestContext) CreateTestPodWithRetry(testPodName string, containers []util.ContainerConfig, retries int, retryInterval time.Duration, timeout time.Duration) error {
	util.LogInfo("Creating test pod %s with retry mechanism", testPodName)
	_, err := util.RetryPodCreationWithWait(ctx.Clientset, ctx.Config, ctx.Namespace, testPodName, containers, nil, retries, retryInterval, timeout)
	if err != nil {
		util.LogError("Failed to create test pod %s: %v", testPodName, err)
		return errors.Wrap(err, "failed to create test client pod after retries")
	}
	util.LogInfo("Successfully created test pod %s", testPodName)
	return nil
}

// WaitForPodAndCheckLogs waits for the pod to complete and checks logs for a substring
func (ctx *TestContext) WaitForPodAndCheckLogs(podName, logSubstring string, checkInterval, timeout time.Duration) error {
	util.LogInfo("Waiting for pod %s to complete and checking logs for substring: %s", podName, logSubstring)
	Eventually(func() (bool, error) {
		err := util.WaitForPodCompletionOrFailure(ctx.Clientset, ctx.Namespace, podName, checkInterval, timeout)
		if err != nil {
			util.LogError("Failed to wait for pod %s completion: %v", podName, err)
			return false, err
		}

		podLogs, err := util.GetPodLogs(ctx.Clientset, ctx.Namespace, podName)
		if err != nil {
			util.LogError("Failed to fetch logs for pod %s: %v", podName, err)
			return false, err
		}

		util.LogInfo("Fetched logs for pod %s. Checking for substring: %s", podName, logSubstring)
		return strings.Contains(podLogs, logSubstring), nil
	}, timeout, checkInterval).Should(BeTrue(), "Expected to find %s in pod logs for pod %s", logSubstring, podName)

	util.LogInfo("Successfully found log substring %s in pod %s", logSubstring, podName)
	return nil
}