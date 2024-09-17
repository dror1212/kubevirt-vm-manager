package main

import (
    "log"
    "myproject/util"
)

func main() {
    scriptPath := "./print_os_info.sh"
    namespace := "core"
    templateName := "rhel8-4-az-a"
    vmName := "test-vm-2"
    memory := "2Gi"
    cpuRequest := "500m"
    cpuLimit := "1000m"
    waitForCreation := true // Set this to 'true' to wait for the VM creation

    // Authenticate using in-cluster config or kubeconfig
    _, config, err := util.Authenticate()
    if err != nil {
        log.Fatalf("Failed to authenticate: %v", err)
    }

    log.Println("Successfully authenticated with Kubernetes")

    // Call the CreateVM function, passing the clientset and config
    err = util.CreateVM(config, scriptPath, namespace, templateName, vmName, memory, cpuRequest, cpuLimit, waitForCreation)
    if err != nil {
        log.Fatalf("Error creating VM: %v", err)
    }

    log.Println("VM creation process completed successfully.")
}

