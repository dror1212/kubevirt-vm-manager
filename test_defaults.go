package main

import (
	"myproject/util"
)

func main() {
	// Define parameters
	namespace := "core"
	templateName := "rhel8-4-az-a"

	// Authenticate using in-cluster config or kubeconfig
	clientset, config, err := util.Authenticate()
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

	// Pass nil for resourceRequirements to use default resources
	vm, err := util.CreateVM(config, namespace, templateName, "", nil, nil, false, "", "")
	if err != nil {
		util.LogError("Error creating VM: %v", err)
		return
	}

	// Check if vm is nil before trying to access it
	if vm == nil {
		util.LogError("VM creation returned nil object.")
		return
	}

	util.LogInfo("VM %s created successfully.", vm.ObjectMeta.Name)
}
