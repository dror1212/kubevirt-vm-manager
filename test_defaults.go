package main

import (
	"log"
	"myproject/util"
)

func main() {
	// Define parameters
	namespace := "core"
	templateName := "rhel8-4-az-a"

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

	// Pass nil for resourceRequirements to use default resources
	vm, err := util.CreateVM(config, namespace, templateName, "", nil, nil, false, "", "")
	if err != nil {
		log.Fatalf("Error creating VM: %v", err)
	}

	// Check if vm is nil before trying to access it
	if vm == nil {
		log.Fatalf("VM creation returned nil object.")
	}

	log.Printf("VM %s created successfully.", vm.ObjectMeta.Name)
}
