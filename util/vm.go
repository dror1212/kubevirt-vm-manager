package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	templatev1 "github.com/openshift/api/template/v1"
	templateclientset "github.com/openshift/client-go/template/clientset/versioned"
	kubevirtv1 "kubevirt.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// CreateVM creates a VM using the given parameters and optionally waits for the VM creation to complete
func CreateVM(config *rest.Config, scriptPath, namespace, templateName, vmName string, memory, cpuRequest, cpuLimit string, waitForCreation bool) error {
	// Read the external script from a file
	externalScript, err := readExternalScript(scriptPath)
	if err != nil {
		return fmt.Errorf("error reading external script: %v", err)
	}

	// Use constant for namespace where templates reside
	templateNamespace := consts.DefaultTemplateNamespace

	// Create a client for the OpenShift template API using the provided config
	templateClient, err := templateclientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create template client: %v", err)
	}

	// Fetch the template
	template, err := templateClient.TemplateV1().Templates(templateNamespace).Get(context.TODO(), templateName, meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get template: %v", err)
	}
	// Create a decoder to handle RawExtension objects
	scheme := runtime.NewScheme()
	_ = kubevirtv1.AddToScheme(scheme) // Register the KubeVirt scheme
	decoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	// Iterate through the template objects and find the VirtualMachine
	for i, obj := range template.Objects {
		decodedObj, _, err := decoder.Decode(obj.Raw, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to decode object in template: %v", err)
		}

		// Check if it's a VirtualMachine object
		vm, ok := decodedObj.(*kubevirtv1.VirtualMachine)
		if ok {
			// Set resource requests and limits for the VM
			vm.Spec.Template.Spec.Domain.Resources = kubevirtv1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse(memory),
					corev1.ResourceCPU:    resource.MustParse(cpuRequest),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse(memory),
					corev1.ResourceCPU:    resource.MustParse(cpuLimit),
				},
			}

			// Ensure the VM starts automatically by setting 'Running' to true
			vm.Spec.Running = new(bool)
			*vm.Spec.Running = true

			// Modify the existing cloud-init data to include the script
			for _, volume := range vm.Spec.Template.Spec.Volumes {
				if volume.CloudInitNoCloud != nil {
					mergedCloudInit := mergeOrCreateCloudInit(volume.CloudInitNoCloud.UserData, externalScript)
					volume.CloudInitNoCloud.UserData = mergedCloudInit
					break
				}
			}

			// Convert the modified VM object back to RawExtension
			raw, err := runtime.Encode(serializer.NewCodecFactory(scheme).LegacyCodec(kubevirtv1.SchemeGroupVersion), vm)
			if err != nil {
				return fmt.Errorf("failed to encode VM object: %v", err)
			}

			// Update the template's object with the modified VM
			template.Objects[i].Raw = raw
		}
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

	// Create the TemplateInstance
	_, err = templateClient.TemplateV1().TemplateInstances(namespace).Create(context.TODO(), templateInstance, meta_v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create TemplateInstance: %v", err)
	}

	// Wait for the VM creation if requested
	if waitForCreation {
		fmt.Printf("Waiting for the VM %s to be created...\n", vmName)
		for {
			ti, err := templateClient.TemplateV1().TemplateInstances(namespace).Get(context.TODO(), vmName, meta_v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get TemplateInstance status: %v", err)
			}

			// Check if the TemplateInstance has been processed
			processed := false
			for _, condition := range ti.Status.Conditions {
				if condition.Type == templatev1.TemplateInstanceReady && condition.Status == "True" {
					processed = true
					break
				}
			}
			if processed {
				fmt.Printf("VM %s has been created successfully.\n", vmName)
				break
			}

			// Sleep before checking again
			time.Sleep(30 * time.Second)
		}
	}

	return nil
}

