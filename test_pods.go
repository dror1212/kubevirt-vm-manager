package main

import (
	"flag"
	"fmt"
	"os"
	"myproject/util"
)

func main() {
	// Define parameters
	namespace := "core"
	podName := "test-pod"
	image := "quay.med.one:8443/openshift/ubi8/ubi"
	cpuRequests := "500m"
	cpuLimits := "2000m"
	memoryRequests := "1Gi"
	memoryLimits := "1Gi"
	labels := map[string]string{
		"app": podName,
	}

	// Define a command-line flag for the log level
	logLevel := flag.String("loglevel", "info", "Log level (debug, info, warn, error, fatal)")
	flag.Parse() // Parse the flags

	// Map the input string to a custom LogLevel and set it
	err := util.SetLogLevel(*logLevel)
	if err != nil {
		fmt.Println(err)
		os.Exit(1) // Exit if invalid log level is provided
	}

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

	// Define the container configurations
	containers := []util.ContainerConfig{
		{
			Name:  "test-container",
			Image: image,
			Resources: util.GenerateResourceRequirements(cpuRequests, cpuLimits, memoryRequests, memoryLimits),
		},
	}

	// Create the pod
	pod, err := util.CreatePod(config, namespace, podName, containers, labels, true)
	if err != nil {
		util.LogError("Failed to create pod: %v", err)
		return
	}

	util.LogInfo("Pod %s created successfully.", pod.Name)
}
