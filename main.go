package main

import (
    "context"
	"fmt"
	"log"
	"time"
	"myproject/util"
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	sshPublicKeyPath := "./test.pub" // Path to your public key
	privateKeyPath := "./test"       // Path to your private key for SSH access
	labels := map[string]string{
		"app": vmName,
	}
	ports := []corev1.ServicePort{
		util.GeneratePort("ssh", 22, 22, "TCP"), // Open port 22 for SSH
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
	vm, err := util.CreateVM(config, namespace, templateName, vmName, &convertedResourceRequirements, nil, waitForCreation, scriptPath, sshPublicKeyPath)
	if err != nil {
		log.Fatalf("Error creating VM: %v", err)
	}

	// Check if vm is nil before trying to access it
	if vm == nil {
		log.Fatalf("VM creation returned nil object.")
	}

	log.Printf("VM %s created successfully.", vm.ObjectMeta.Name)

	// Create a LoadBalancer service for the VM
	service, err := util.CreateService(clientset, namespace, serviceName, corev1.ServiceTypeLoadBalancer, ports, labels)
	if err != nil {
		log.Fatalf("Failed to create LoadBalancer service: %v", err)
	}

	log.Printf("LoadBalancer service %s created successfully", service.Name)

	// Wait for the service to get an external IP
	var externalIP string
	for i := 0; i < 10; i++ { // Poll for the external IP for a maximum of 10 times (adjust as needed)
		svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, meta_v1.GetOptions{})
		if err != nil {
			log.Fatalf("Failed to get service: %v", err)
		}

		// Check for an external IP in the LoadBalancer status
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			externalIP = svc.Status.LoadBalancer.Ingress[0].IP
			break
		}

		log.Println("Waiting for the LoadBalancer service to get an external IP...")
		time.Sleep(10 * time.Second) // Wait before retrying
	}

	if externalIP == "" {
		log.Fatalf("LoadBalancer service did not get an external IP.")
	}

	log.Printf("External IP of the service is: %s", externalIP)

	// SSH into the VM and read the /tmp/os_info.txt file
	sshConfig := &util.SSHConfig{
		User:       "cloud-user",      // Replace with the correct username for your VM
		Host:       externalIP,        // Use the external IP address of the service
		Port:       22,                // Default SSH port
		PrivateKey: privateKeyPath,    // Path to the private key
	}

	sshClient, err := util.PollSSHConnection(sshConfig, 5*time.Second, 2*time.Minute)
	if err != nil {
		log.Fatalf("Failed to create SSH client: %v", err)
	}
	defer sshClient.Close()

	// Read the content of the file /tmp/os_info.txt
	fileContent, err := sshClient.ReadFileContent("/tmp/os_info.txt")
	if err != nil {
		log.Fatalf("Failed to read file from VM: %v", err)
	}

	// Print the content of the file
	fmt.Printf("File content of /tmp/os_info.txt: \n%s\n", fileContent)

	// Check if a specific word exists
	word := "x86_64"
	wordFound := util.CheckWordPresence(fileContent, word)
	log.Printf("Word '%s' found: %v", word, wordFound)
}
