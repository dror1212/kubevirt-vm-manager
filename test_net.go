package main

import (
    "context"
	"time"
	"myproject/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	// Define parameters
	namespace := "core"
	serviceName := "test-vm-2-service"
	privateKeyPath := "./test"       // Path to your private key for SSH access

	// Authenticate using in-cluster config or kubeconfig
	clientset, _, err := util.Authenticate()
	if err != nil {
		util.LogError("Failed to authenticate: %v", err)
		return
	}

	util.LogInfo("Successfully authenticated with Kubernetes.")

	// Verify that the connection to the cluster is working
	err = util.VerifyConnection(clientset)
	if err != nil {
		util.LogError("Failed to verify connection: %v", err)
		return
	}

	util.LogInfo("Kubernetes connection verified successfully.")

	// Wait for the service to get an external IP
	var externalIP string
	for i := 0; i < 10; i++ { // Poll for the external IP for a maximum of 10 times (adjust as needed)
		svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, meta_v1.GetOptions{})
		if err != nil {
			util.LogError("Failed to get service: %v", err)
			return
		}

		// Check for an external IP in the LoadBalancer status
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			externalIP = svc.Status.LoadBalancer.Ingress[0].IP
			break
		}

		util.LogInfo("Waiting for the LoadBalancer service to get an external IP...")
		time.Sleep(10 * time.Second) // Wait before retrying
	}

	if externalIP == "" {
		util.LogError("LoadBalancer service did not get an external IP.")
		return
	}

	util.LogInfo("External IP of the service is: %s", externalIP)

	// SSH into the VM and read the /tmp/os_info.txt file
	sshConfig := &util.SSHConfig{
		User:       "cloud-user",      // Replace with the correct username for your VM
		Host:       externalIP,        // Use the external IP address of the service
		Port:       22,                // Default SSH port
		PrivateKey: privateKeyPath,    // Path to the private key
	}

	sshClient, err := util.PollSSHConnection(sshConfig, 5*time.Second, 2*time.Minute)
	if err != nil {
		util.LogError("Failed to create SSH client: %v", err)
		return
	}
	defer sshClient.Close()

	// Read the content of the file /tmp/os_info.txt
	fileContent, err := sshClient.ReadFileContent("/tmp/os_info.txt")
	if err != nil {
		util.LogError("Failed to read file from VM: %v", err)
		return
	}

	// Print the content of the file
	util.LogInfo("File content of /tmp/os_info.txt: \n%s\n", fileContent)
}
