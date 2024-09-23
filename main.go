package main

import (
	"log"
	"myproject/util"
    corev1 "k8s.io/api/core/v1"
)

func main() {
	// Define parameters
	scriptPath := "./print_os_info.sh"
	namespace := "core"
	templateName := "rhel8-4-az-a"
	vmName := "test-vm-2"
	cpuRequests := "500m"
	cpuLimits := "1000m"
	memoryRequests := "2Gi"
	memoryLimits := "2Gi"
	waitForCreation := true // Set this to 'true' to wait for the VM creation
    serviceName := "test-vm-2-service"
    labels := map[string]string{
        "app": vmName,
    }
    ports := []corev1.ServicePort{
        util.GeneratePort("ssh", 22, 22, "TCP"),    // Open port 22 for SSH
    }

	// Authenticate using in-cluster config or kubeconfig
	clientset, config, err := util.Authenticate()
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	log.Println("Successfully authenticated with Kubernetes.")

	// Verify that the connection to the cluster is working
	err = util.VerifyConnection(clientset)
	if err != nil {
		log.Fatalf("Failed to verify connection: %v", err)
	}

	log.Println("Kubernetes connection verified successfully.")

	// Generate the resource requirements
	resourceRequirements := util.GenerateResourceRequirements(cpuRequests, cpuLimits, memoryRequests, memoryLimits)
    convertedResourceRequirements := util.ConvertCoreV1ToKubeVirtResourceRequirements(resourceRequirements)

	// Call the CreateVM function, passing the resource requirements
	err = util.CreateVM(config, namespace, templateName, vmName, &convertedResourceRequirements, nil, waitForCreation, scriptPath)
	if err != nil {
		log.Fatalf("Error creating VM: %v", err)
	}

	log.Println("VM creation process completed successfully.")

    // Create a LoadBalancer service for the VM
    service, err := util.CreateService(clientset, namespace, serviceName, corev1.ServiceTypeLoadBalancer, ports, labels)
    if err != nil {
        log.Fatalf("Failed to create LoadBalancer service: %v", err)
    }

    log.Printf("LoadBalancer service %s created successfully", service.Name)

}
