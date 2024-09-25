package util

import (
    "fmt"
    "golang.org/x/crypto/ssh"
    "io/ioutil"
	"io"
    "os"
	"time"
)

// SSHConfig holds the SSH configuration details
type SSHConfig struct {
    User        string
    Host        string
    Port        int
    PrivateKey  string // Path to private key
}

// SSHClient holds the SSH client connection
type SSHClient struct {
    config *SSHConfig
    client *ssh.Client
}

// PollSSHConnection tries to establish an SSH connection until it succeeds or times out.
func PollSSHConnection(sshConfig *SSHConfig, interval, timeout time.Duration) (*SSHClient, error) {
	// Use the WaitFor utility to attempt SSH connection
	err := WaitFor(func() (bool, error) {
		client, err := NewSSHClient(sshConfig)
		if err == nil {
			fmt.Println("SSH connection established successfully.")
			sshClient := &SSHClient{config: sshConfig, client: client}
			return true, nil // Return true if the connection is successful
		}
		fmt.Println("SSH not ready, retrying...")
		return false, nil // Return false to indicate retry
	}, interval, timeout)

	if err != nil {
		return nil, fmt.Errorf("timed out waiting for SSH connection to become available: %v", err)
	}

	// Return the successful SSH client
	client, _ := NewSSHClient(sshConfig)
	return &SSHClient{config: sshConfig, client: client}, nil
}

// NewSSHClient is your existing function to create an SSH client.
func NewSSHClient(config *SSHConfig) (*ssh.Client, error) {
	// Read the private key file
	key, err := ioutil.ReadFile(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}

	// Parse the private key for use in authentication
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	// Prepare the SSH configuration with the private key and user
	sshConfig := &ssh.ClientConfig{
		User: config.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Use a more secure HostKeyCallback in production
		Timeout:         10 * time.Second,
	}

	// Format the address as "host:port"
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	fmt.Printf("Attempting to connect to SSH at %s...\n", address)

	// Dial the SSH connection
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %v", err)
	}

	return client, nil
}

// RunCommand runs a command on the remote VM and returns the output
func (s *SSHClient) RunCommand(cmd string) (string, error) {
    session, err := s.client.NewSession()
    if err != nil {
        return "", fmt.Errorf("failed to create session: %v", err)
    }
    defer session.Close()

    // Run the command
    output, err := session.CombinedOutput(cmd)
    if err != nil {
        return "", fmt.Errorf("failed to run command: %v", err)
    }

    return string(output), nil
}

// CopyFileToVM copies a local file to the remote VM
func (s *SSHClient) CopyFileToVM(localFilePath, remoteFilePath string) error {
    // Open the local file
    localFile, err := os.Open(localFilePath)
    if err != nil {
        return fmt.Errorf("failed to open local file: %v", err)
    }
    defer localFile.Close()

    // Create a new session for SCP
    session, err := s.client.NewSession()
    if err != nil {
        return fmt.Errorf("failed to create session: %v", err)
    }
    defer session.Close()

    // Set up SCP
    remoteFileCommand := fmt.Sprintf("scp -t %s", remoteFilePath)
    pipe, err := session.StdinPipe()
    if err != nil {
        return fmt.Errorf("failed to create stdin pipe: %v", err)
    }

    // Start SCP session
    err = session.Start(remoteFileCommand)
    if err != nil {
        return fmt.Errorf("failed to start SCP session: %v", err)
    }

    // Copy file contents
    fileInfo, _ := localFile.Stat()
    fmt.Fprintf(pipe, "C0644 %d %s\n", fileInfo.Size(), remoteFilePath)
    io.Copy(pipe, localFile)
    fmt.Fprint(pipe, "\x00")

    // Wait for the session to complete
    err = session.Wait()
    if err != nil {
        return fmt.Errorf("failed to copy file: %v", err)
    }

    return nil
}

// ReadFileContent reads the content of a remote file from the VM
func (s *SSHClient) ReadFileContent(remoteFilePath string) (string, error) {
    // Run the cat command to read the file content
    return s.RunCommand(fmt.Sprintf("cat %s", remoteFilePath))
}

// Close closes the SSH connection
func (s *SSHClient) Close() {
    s.client.Close()
}
