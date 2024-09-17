
# KubeVirt VM Manager

A Go-based application to manage the creation and deployment of Virtual Machines (VMs) using KubeVirt and OpenShift templates. This project simplifies the process of creating VMs using templates, including resource configurations and cloud-init scripts.  

## Table of Contents
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)

## Overview

This project allows you to:
- Create and manage VMs on KubeVirt using predefined templates.
- Automatically configure VM resources (CPU, memory) and cloud-init scripts.
- Automatically start the VMs after creation.
- Optionally log all operations with timestamps to both a file and console.

## Prerequisites

Ensure you have the following prerequisites installed:
- Go (v1.16+)
- Kubernetes cluster with KubeVirt and OpenShift installed.
- Access to a KubeVirt-enabled cluster with the required permissions.
- Git for version control.

### Environment Requirements:
- `kubeconfig`: Ensure that your kubeconfig file is accessible for authentication.

## Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/dror1212/kubevirt-vm-manager.git
   cd kubevirt-vm-manager
   ```

2. **Install dependencies**:
   Ensure that the Go modules are installed:
   ```bash
   go mod tidy
   ```

3. **Set up KubeConfig**:
   - Ensure that your `kubeconfig` is set up and that you have the necessary access to interact with the Kubernetes API.
   - You can specify your `KUBECONFIG` environment variable if needed:
     ```bash
     export KUBECONFIG=/path/to/your/kubeconfig
     ```

## Usage

To run the application:

```bash
go run main.go
```

This will:
- Authenticate with the Kubernetes cluster.
- Create a new VM using the specified template.
- Start the VM and log the progress.

### Example Commands:
- **VM Creation**: 
  Customize your VM by editing the parameters in `main.go` or passing them programmatically.
  The default configuration uses a `print_os_info.sh` script for cloud-init to gather OS information.

### Files Overview:

- **`main.go`**: The entry point of the application.
- **`util/`**: Contains utility files like VM creation logic (`vm.go`).
- **`consts/`**: Holds constant variables such as default memory and namespace settings.
- **`print_os_info.sh`**: A shell script used by cloud-init to print OS information into a file on the VM.
- **`go.mod`**: Go modules dependencies file.

## Configuration

You can configure the default settings like CPU, memory, and namespaces in the `consts/constants.go` file.

```go
package consts

const (
    DefaultMemory    = "2Gi"
    DefaultCPUReq    = "500m"
    DefaultCPULimit  = "1000m"
    DefaultNamespace = "core"
    TemplateNamespace = "template-ns"
    DefaultVMName    = "test-vm-2"
)
```

To adjust the script used for cloud-init, modify the `print_os_info.sh` file or create a new shell script.