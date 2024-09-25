package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

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
		return "", fmt.Errorf("failed to read external script: %w", err)
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
    // Generate random VM name if not provided
    if vmName == "" {
        vmName = generateRandomName()
    }

    // Use default resource requirements if none are provided
    if resourceRequirements == nil {
        defaultResources := ConvertCoreV1ToKubeVirtResourceRequirements(consts.DefaultResources)
        resourceRequirements = &defaultResources // Take the address of the default value
    }

    // Set default labels if none are provided
    if labels == nil {
        labels = consts.DefaultLabels
        labels["app"] = vmName
    }

    // Read the external script from a file if the scriptPath is provided
    var externalScript string
    if scriptPath != "" {
        var err error
        externalScript, err = readExternalScript(scriptPath)
        if err != nil {
            return nil, fmt.Errorf("error reading external script: %v", err)
        }
    }

    // Create a client for the OpenShift template API using the provided config
    templateClient, err := templateclientset.NewForConfig(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create template client: %v", err)
    }

    // Fetch the template
    template, err := templateClient.TemplateV1().Templates(consts.DefaultTemplateNamespace).Get(context.TODO(), templateName, meta_v1.GetOptions{})
    if err != nil {
        return nil, fmt.Errorf("failed to get template: %v", err)
    }

    // Create a decoder to handle RawExtension objects
    scheme := runtime.NewScheme()
    _ = kubevirtv1.AddToScheme(scheme)
    decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

    var vm *kubevirtv1.VirtualMachine

    // Iterate through the template objects and find the VirtualMachine
    for i, obj := range template.Objects {
        decodedObj, _, err := decoder.Decode(obj.Raw, nil, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to decode object in template: %v", err)
        }

        // Check if it's a VirtualMachine object
        decodedVM, ok := decodedObj.(*kubevirtv1.VirtualMachine)
        if ok {
            vm = decodedVM

            // Set resource requests and limits for the VM
            vm.Spec.Template.Spec.Domain.Resources = *resourceRequirements

            // Set the VM name within the template object
            vm.ObjectMeta.Name = vmName

            if vm.ObjectMeta.Labels == nil {
                vm.ObjectMeta.Labels = labels
            } else {
                for key, value := range labels {
                    vm.ObjectMeta.Labels[key] = value
                }
            }

            // Set labels in the PodTemplateSpec so that the VM's pod inherits the same labels
            if vm.Spec.Template.ObjectMeta.Labels == nil {
                vm.Spec.Template.ObjectMeta.Labels = labels
            } else {
                for key, value := range labels {
                    vm.Spec.Template.ObjectMeta.Labels[key] = value
                }
            }

            // Ensure the VM starts automatically by setting 'Running' to true
            running := true
            vm.Spec.Running = &running

            // Modify the existing cloud-init data to include the script if provided
            for _, volume := range vm.Spec.Template.Spec.Volumes {
                if volume.CloudInitNoCloud != nil {
                    // Add the SSH public key to cloud-init if the path is provided
                    if sshPublicKeyPath != "" {
                        sshPublicKey, err := ioutil.ReadFile(sshPublicKeyPath)
                        if err != nil {
                            return nil, fmt.Errorf("failed to read SSH public key: %v", err)
                        }
                        volume.CloudInitNoCloud.UserData = addSSHKeyToCloudInit(volume.CloudInitNoCloud.UserData, string(sshPublicKey))
                    }

                    // Add external script if provided
                    if scriptPath != "" {
                        mergedCloudInit := mergeOrCreateCloudInit(volume.CloudInitNoCloud.UserData, externalScript)
                        volume.CloudInitNoCloud.UserData = mergedCloudInit
                    }
                    break
                }
            }

            // Convert the modified VM object back to RawExtension
            raw, err := runtime.Encode(serializer.NewCodecFactory(scheme).LegacyCodec(kubevirtv1.SchemeGroupVersion), vm)
            if err != nil {
                return nil, fmt.Errorf("failed to encode VM object: %v", err)
            }

            // Update the template's object with the modified VM
            template.Objects[i].Raw = raw
        }
    }

    if vm == nil {
        return nil, fmt.Errorf("no VirtualMachine object found in the template")
    }

    // Create a TemplateInstance
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
        return nil, fmt.Errorf("failed to create KubeVirt client: %v", err)
    }

    // Create the TemplateInstance
    _, err = templateClient.TemplateV1().TemplateInstances(namespace).Create(context.TODO(), templateInstance, meta_v1.CreateOptions{})
    if err != nil {
        return nil, fmt.Errorf("failed to create TemplateInstance: %v", err)
    }

    // Wait for the VM creation if requested
    if waitForCreation {
        fmt.Printf("Waiting for the TemplateInstance %s to be created...\n", vmName)
        err = WaitForTemplateInstanceReady(templateClient, namespace, vmName, 5*time.Second, 120*time.Second)
        if err != nil {
            return nil, fmt.Errorf("error waiting for VM to be ready: %v", err)
        }
        fmt.Printf("TemplateInstance %s has been created successfully.\n", vmName)

        fmt.Printf("Waiting for the VM %s to be created...\n", vmName)
        err = WaitForVMReady(virtClient, namespace, vmName, 5*time.Second, 120*time.Second)
        if err != nil {
            return nil, fmt.Errorf("error waiting for VM to be ready: %v", err)
        }
        fmt.Printf("VM %s has been created successfully.\n", vmName)
    }

    return vm, nil // Return the created VM object
}