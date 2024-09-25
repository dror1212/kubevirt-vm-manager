package util

import (
	"fmt"
	"time"
)

// WaitFor function accepts a checking function, interval, and timeout to wait for a resource to be in a ready state
func WaitFor(checkFunc func() (bool, error), interval time.Duration, timeout time.Duration) error {
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			errMsg := "Timeout reached while waiting"
			LogError(errMsg)
			return fmt.Errorf(errMsg)
		case <-ticker.C:
			// Check the condition function
			ready, err := checkFunc()
			if err != nil {
				errMsg := "Error during wait: %v"
				LogError(errMsg, err)
				return fmt.Errorf(errMsg, err)
			}
			if ready {
				return nil
			}
		}
	}
}
