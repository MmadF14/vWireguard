package util

import (
	"fmt"
	"os/exec"
	"strings"
)

// RestartService restarts a systemd service
func RestartService(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "restart", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %v (output: %s)", serviceName, err, string(output))
	}

	// Verify service is active
	checkCmd := exec.Command("sudo", "systemctl", "is-active", serviceName)
	status, err := checkCmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(status)) != "active" {
		return fmt.Errorf("service %s is not active after restart (status: %s)", serviceName, string(status))
	}

	return nil
}
