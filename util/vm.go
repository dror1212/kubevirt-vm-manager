package util

import (
	"context"
	"io/ioutil"
	"strings"
	"time"
    "fmt"

	templatev1 "github.com/openshift/api/template/v1"
	templateclientset "github.com/openshift/client-go/template/clientset/versioned"
	kubevirtv1 "kubevirt.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubecli "kubevirt.io/client-go/kubecli"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"myproject/consts"
)

// readExternalScript reads the content of an external script file
func readExternalScript(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		LogError("Failed to read external script: %v", err)
		return "", err
	}
	return string(content), nil
}

// formatScriptContent ensures correct indentation and YAML formatting for the script
func formatScriptContent(script string) string {
	lines := strings.Split(script, "\n")
	for i, line := range lines {
		lines[i] = "        " + line // Indent by 8 spaces for correct YAML indentation under 'content:'
	}
	return strings.Join(lines, "\n")
}

// mergeOrCreateCloudInit modifies or creates cloud-init to include the provided script
func mergeOrCreateCloudInit(existingData, scriptContent string) string {
	existingData = strings.TrimSpace(existingData)

	if !strings.HasPrefix(existingData, "#cloud-config") {
		existingData = "#cloud-config\n" + existingData
	}

	if strings.Contains(existingData, "write_files:") {
		existingData = strings.Replace(existingData, "write_files:", "write_files:\n  - path: /tmp/myscript.sh\n    permissions: '0755'\n    content: |\n"+formatScriptContent(scriptContent), 1)
	} else {
		existingData += "\nwrite_files:\n  - path: /tmp/myscript.sh\n    permissions: '0755'\n    content: |\n" + formatScriptContent(scriptContent)
	}

	if strings.Contains(existingData, "runcmd:") {
		existingData = strings.Replace(existingData, "runcmd:", "runcmd:\n  - bash /tmp/myscript.sh", 1)
	} else {
		existingData += "\nruncmd:\n  - bash /tmp/myscript.sh"
	}

	return existingData
}

// addSSHKeyToCloudInit modifies the cloud-init to include the SSH public key
func addSSHKeyToCloudInit(existingData, sshPublicKey string) string {
	existingData = strings.TrimSpace(existingData)

	if !strings.HasPrefix(existingData, "#cloud-config") {
		existingData = "#cloud-config\n" + existingData
	}

	if strings.Contains(existingData, "ssh_authorized_keys:") {
        existingData = strings.Replace(existingData, "ssh_authorized_keys:", fmt.Sprintf("ssh_authorized_keys:\n  - %s", sshPublicKey), 1)
	} else {
        existingData += fmt.Sprintf("\nssh_authorized_keys:\n  - %s", sshPublicKey)
	}

	return existingData
}

// CreateVM creates a VM using the given parameters and optionally adds an SSH public key
func CreateVM(config *rest.Config, namespace, templateName, vmName string, resourceRequirements *kubevirtv1.ResourceRequirements, labels map[string]string, waitForCreation bool, scriptPath, sshPublicKeyPath string) (*kubevirtv1.VirtualMachine, error) {
    if templateName == "" {
		templateName = consts.DefaultTemplateName
		LogInfo("Using the default template", templateName)
	}
    
    if vmName == "" {
		vmName = GenerateRandomName()
		LogInfo("Generated random VM name: %s", vmName)
	}

	if resourceRequirements == nil {
		defaultResources := ConvertCoreV1ToKubeVirtResourceRequirements(consts.DefaultResources)
		resourceRequirements = &defaultResources
		LogInfo("Using default resource requirements")
	}

	if labels == nil {
		labels = consts.DefaultLabels
		labels["app"] = vmName
		LogInfo("Using default labels for VM: %s", vmName)
	}

	var externalScript string
	if scriptPath != "" {
		var err error
		externalScript, err = readExternalScript(scriptPath)
		if err != nil {
			return nil, err
		}
		LogInfo("Successfully read external script from %s", scriptPath)
	}

	templateClient, err := templateclientset.NewForConfig(config)
	if err != nil {
		LogError("Failed to create template client: %v", err)
		return nil, err
	}

	template, err := templateClient.TemplateV1().Templates(consts.DefaultTemplateNamespace).Get(context.TODO(), templateName, meta_v1.GetOptions{})
	if err != nil {
		LogError("Failed to fetch template: %v", err)
		return nil, err
	}

	scheme := runtime.NewScheme()
	_ = kubevirtv1.AddToScheme(scheme)
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	var vm *kubevirtv1.VirtualMachine
	for i, obj := range template.Objects {
		decodedObj, _, err := decoder.Decode(obj.Raw, nil, nil)
		if err != nil {
			LogError("Failed to decode object in template: %v", err)
			return nil, err
		}

		decodedVM, ok := decodedObj.(*kubevirtv1.VirtualMachine)
		if ok {
			vm = decodedVM

			vm.Spec.Template.Spec.Domain.Resources = *resourceRequirements
			vm.ObjectMeta.Name = vmName

			if vm.ObjectMeta.Labels == nil {
				vm.ObjectMeta.Labels = labels
			} else {
				for key, value := range labels {
					vm.ObjectMeta.Labels[key] = value
				}
			}

			if vm.Spec.Template.ObjectMeta.Labels == nil {
				vm.Spec.Template.ObjectMeta.Labels = labels
			} else {
				for key, value := range labels {
					vm.Spec.Template.ObjectMeta.Labels[key] = value
				}
			}

			running := true
			vm.Spec.Running = &running

			for _, volume := range vm.Spec.Template.Spec.Volumes {
				if volume.CloudInitNoCloud != nil {
					if sshPublicKeyPath != "" {
						sshPublicKey, err := ioutil.ReadFile(sshPublicKeyPath)
						if err != nil {
							LogError("Failed to read SSH public key: %v", err)
							return nil, err
						}
						volume.CloudInitNoCloud.UserData = addSSHKeyToCloudInit(volume.CloudInitNoCloud.UserData, string(sshPublicKey))
					}

					if scriptPath != "" {
						mergedCloudInit := mergeOrCreateCloudInit(volume.CloudInitNoCloud.UserData, externalScript)
						volume.CloudInitNoCloud.UserData = mergedCloudInit
					}
					break
				}
			}

			raw, err := runtime.Encode(serializer.NewCodecFactory(scheme).LegacyCodec(kubevirtv1.SchemeGroupVersion), vm)
			if err != nil {
				LogError("Failed to encode VM object: %v", err)
				return nil, err
			}

			template.Objects[i].Raw = raw
		}
	}

	if vm == nil {
        errMsg := "No VirtualMachine object found in the template"
		LogError(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	templateInstance := &templatev1.TemplateInstance{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      vmName,
			Namespace: namespace,
		},
		Spec: templatev1.TemplateInstanceSpec{
			Template: *template,
		},
	}

	virtClient, err := kubecli.GetKubevirtClientFromRESTConfig(config)
	if err != nil {
		LogError("Failed to create KubeVirt client: %v", err)
		return nil, err
	}

	_, err = templateClient.TemplateV1().TemplateInstances(namespace).Create(context.TODO(), templateInstance, meta_v1.CreateOptions{})
	if err != nil {
		LogError("Failed to create TemplateInstance: %v", err)
		return nil, err
	}

	if waitForCreation {
		LogInfo("Waiting for the TemplateInstance %s to be created", vmName)
		err = WaitForTemplateInstanceReady(templateClient, namespace, vmName, 5*time.Second, 120*time.Second)
		if err != nil {
			LogError("Error waiting for TemplateInstance: %v", err)
			return nil, err
		}
		LogInfo("TemplateInstance %s has been created", vmName)

		LogInfo("Waiting for the VM %s to be created", vmName)
		err = WaitForVMReady(virtClient, namespace, vmName, 5*time.Second, 120*time.Second)
		if err != nil {
			LogError("Error waiting for VM: %v", err)
			return nil, err
		}
		LogInfo("VM %s has been created successfully", vmName)
	}

	return vm, nil
}

// GetVMPodIP fetches the Pod IP associated with the given VM
func GetVMPodIP(virtClient kubecli.KubevirtClient, namespace, vmName string) (string, error) {
	// Fetch the VM object
	vm, err := virtClient.VirtualMachine(namespace).Get(context.TODO(), vmName, meta_v1.GetOptions{}) // Correct: pass by value
	if err != nil {
		LogError("Failed to fetch VM: %v", err)
		return "", fmt.Errorf("failed to fetch VM: %v", err)
	}

	// Ensure the VM is running
	if vm.Status.PrintableStatus != "Running" {
		errMsg := fmt.Sprintf("VM %s is not in 'Running' state", vmName)
		LogError(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// Fetch the pod associated with the VM
	podList, err := virtClient.CoreV1().Pods(namespace).List(context.TODO(), meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", vmName),
	})
	if err != nil {
		LogError("Failed to list pods for VM %s: %v", vmName, err)
		return "", fmt.Errorf("failed to list pods for VM %s: %v", vmName, err)
	}

	if len(podList.Items) == 0 {
		errMsg := fmt.Sprintf("No pods found for VM %s", vmName)
		LogError(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// Extract the Pod IP from the first matching pod
	pod := podList.Items[0]
	if pod.Status.PodIP == "" {
		errMsg := fmt.Sprintf("Pod for VM %s does not have a valid Pod IP", vmName)
		LogError(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	LogInfo("Pod IP for VM %s is %s", vmName, pod.Status.PodIP)
	return pod.Status.PodIP, nil
}