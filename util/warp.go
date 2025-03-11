package util

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// runCommand executes a command with timeout and returns its output and error
func runCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("command timed out after 30 seconds")
	}
	if err != nil {
		return string(output), fmt.Errorf("command '%s %s' failed: %v, output: %s", name, strings.Join(args, " "), err, string(output))
	}
	return string(output), nil
}

// checkWARPService checks if WARP is installed and running
func checkWARPService() error {
	// Check if warp-cli exists
	warpCliPath, err := exec.LookPath("warp-cli")
	if err != nil {
		log.Printf("Warning: warp-cli not found in PATH: %v", err)
		return fmt.Errorf("WARP CLI is not installed or not in PATH")
	}

	log.Printf("Found warp-cli at: %s", warpCliPath)

	// Try to check status using warp-cli
	_, err = runCommand("warp-cli", "--accept-tos", "status")
	if err != nil {
		log.Printf("Warning: warp-cli status check failed: %v", err)

		// Try to check service with systemctl
		output, err := runCommand("systemctl", "is-active", "warp-svc")
		if err != nil || !strings.Contains(strings.ToLower(output), "active") {
			log.Printf("Warning: WARP service is not active: %v", err)

			// Try to start the service
			log.Println("Attempting to start WARP service...")
			_, err = runCommand("sudo", "systemctl", "start", "warp-svc")
			if err != nil {
				log.Printf("Warning: Failed to start WARP service: %v", err)
			} else {
				// Wait for service to start
				time.Sleep(5 * time.Second)
			}
		}
	}

	// Continue anyway - we won't block configuration just because service status check failed
	return nil
}

// waitForWARPService waits for WARP service to be ready
func waitForWARPService(timeout time.Duration) error {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for WARP service")
		}

		output, err := runCommand("warp-cli", "--accept-tos", "status")
		if err == nil && strings.Contains(strings.ToLower(output), "connected") {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

// InstallWARP installs Cloudflare WARP if not already installed
func InstallWARP() error {
	log.Println("Checking if WARP is already installed...")

	// Check if warp-cli is already installed
	if warpCliPath, err := exec.LookPath("warp-cli"); err == nil {
		log.Printf("warp-cli already installed at: %s", warpCliPath)
		return nil
	}

	log.Println("WARP CLI not found, attempting to install...")

	// Try to install in different ways
	installMethods := []func() error{
		// Method 1: Direct apt install
		func() error {
			log.Println("Trying direct apt install...")
			if _, err := runCommand("sudo", "apt", "update"); err != nil {
				log.Printf("Warning: apt update failed: %v", err)
			}
			_, err := runCommand("sudo", "apt", "install", "-y", "cloudflare-warp")
			return err
		},
		// Method 2: Add repository first
		func() error {
			log.Println("Trying with repository setup...")
			// Download and add GPG key
			if _, err := runCommand("curl", "-fsSL", "https://pkg.cloudflareclient.com/pubkey.gpg", "--output", "/tmp/cloudflare.gpg"); err != nil {
				return fmt.Errorf("failed to download GPG key: %v", err)
			}

			if _, err := runCommand("sudo", "apt-key", "add", "/tmp/cloudflare.gpg"); err != nil {
				return fmt.Errorf("failed to add GPG key: %v", err)
			}

			// Add repository
			repoCmd := `echo "deb [arch=amd64] https://pkg.cloudflareclient.com/ $(lsb_release -sc) main" | sudo tee /etc/apt/sources.list.d/cloudflare-client.list`
			if _, err := runCommand("bash", "-c", repoCmd); err != nil {
				return fmt.Errorf("failed to add repository: %v", err)
			}

			if _, err := runCommand("sudo", "apt", "update"); err != nil {
				log.Printf("Warning: apt update failed: %v", err)
			}

			_, err := runCommand("sudo", "apt", "install", "-y", "cloudflare-warp")
			return err
		},
	}

	// Try each installation method
	var lastErr error
	for i, method := range installMethods {
		log.Printf("Trying installation method %d...", i+1)
		if err := method(); err != nil {
			log.Printf("Installation method %d failed: %v", i+1, err)
			lastErr = err
		} else {
			log.Printf("Installation method %d succeeded", i+1)
			lastErr = nil
			break
		}
	}

	// Check if warp-cli was installed
	if _, err := exec.LookPath("warp-cli"); err != nil {
		if lastErr != nil {
			log.Printf("All installation methods failed, last error: %v", lastErr)
			return fmt.Errorf("failed to install WARP: %v", lastErr)
		}
	}

	// Try to start the service
	log.Println("Starting WARP service...")
	if _, err := runCommand("sudo", "systemctl", "start", "warp-svc"); err != nil {
		log.Printf("Warning: Failed to start WARP service: %v", err)
	}

	// Wait for service to be ready
	log.Println("Waiting for WARP service to be ready...")
	time.Sleep(5 * time.Second)

	return nil
}

// ConfigureWARP configures WARP with the specified domains
func ConfigureWARP(enabled bool, domains []string) error {
	log.Printf("Configuring WARP (enabled=%v, domains=%v)...", enabled, domains)

	// Check WARP service status
	if err := checkWARPService(); err != nil {
		log.Printf("Warning: WARP service check failed: %v, continuing anyway", err)
	}

	if !enabled {
		log.Println("Disabling WARP...")
		if _, err := runCommand("warp-cli", "--accept-tos", "disconnect"); err != nil {
			log.Printf("Warning: failed to disconnect WARP: %v", err)
		}
		return nil
	}

	// Try to delete old registration - this might fail but that's okay
	log.Println("Checking for existing registration...")
	if _, err := runCommand("warp-cli", "--accept-tos", "registration", "delete"); err != nil {
		log.Printf("Warning: failed to delete old registration (this is normal for first setup): %v", err)
	}

	// Try different initialization commands (different warp-cli versions use different commands)
	registrationSuccess := false
	registrationAttempts := [][]string{
		{"warp-cli", "--accept-tos", "registration", "new"},
		{"warp-cli", "--accept-tos", "register"},
		{"warp-cli", "--accept-tos", "init"},
	}

	for _, cmdArgs := range registrationAttempts {
		log.Printf("Trying to initialize WARP with command: %v", cmdArgs)
		if _, err := runCommand(cmdArgs[0], cmdArgs[1:]...); err == nil {
			registrationSuccess = true
			log.Println("WARP initialization successful")
			break
		} else {
			log.Printf("Initialization attempt failed: %v", err)
		}
	}

	if !registrationSuccess {
		log.Println("Warning: All WARP initialization attempts failed, but continuing anyway")
	}

	// Try different mode commands
	modeSuccess := false
	modeAttempts := [][]string{
		{"warp-cli", "--accept-tos", "mode", "proxy"},
		{"warp-cli", "--accept-tos", "set-mode", "proxy"},
		{"warp-cli", "--accept-tos", "set-mode", "warp+doh"},
	}

	for _, cmdArgs := range modeAttempts {
		log.Printf("Trying to set WARP mode with command: %v", cmdArgs)
		if _, err := runCommand(cmdArgs[0], cmdArgs[1:]...); err == nil {
			modeSuccess = true
			log.Println("WARP mode set successfully")
			break
		} else {
			log.Printf("Mode setting attempt failed: %v", err)
		}
	}

	if !modeSuccess {
		log.Println("Warning: All WARP mode setting attempts failed, but continuing anyway")
	}

	// Try to clear existing proxy settings
	log.Println("Setting up proxy...")
	if _, err := runCommand("warp-cli", "--accept-tos", "proxy-port", "40000"); err != nil {
		log.Printf("Warning: failed to set proxy port: %v", err)
	}

	// Try different commands to add domains
	for _, domain := range domains {
		domainSuccess := false
		domainAttempts := [][]string{
			{"warp-cli", "--accept-tos", "add-proxy-domain", domain},
			{"warp-cli", "--accept-tos", "add-excluded-route", domain},
			{"warp-cli", "--accept-tos", "routing", "add", domain},
		}

		for _, cmdArgs := range domainAttempts {
			log.Printf("Trying to add domain %s with command: %v", domain, cmdArgs)
			if _, err := runCommand(cmdArgs[0], cmdArgs[1:]...); err == nil {
				domainSuccess = true
				log.Printf("Added domain to proxy: %s", domain)
				break
			} else {
				log.Printf("Domain addition attempt failed: %v", err)
			}
		}

		if !domainSuccess {
			log.Printf("Warning: All attempts to add domain %s failed, but continuing", domain)
		}
	}

	// Try to connect WARP
	log.Println("Connecting WARP...")
	if _, err := runCommand("warp-cli", "--accept-tos", "connect"); err != nil {
		log.Printf("Warning: failed to connect WARP: %v", err)
		// Not failing here, we'll check status to confirm
	}

	// Check if WARP is connected
	log.Println("Checking WARP connection status...")
	status, err := GetWARPStatus()
	if err != nil || !status {
		log.Printf("Warning: WARP connection check failed or not connected: %v, status: %v", err, status)
	} else {
		log.Println("WARP connected successfully")
	}

	log.Println("WARP configuration completed")
	return nil
}

// GetWARPStatus returns the current WARP connection status
func GetWARPStatus() (bool, error) {
	// First try to run status command without service check
	output, err := runCommand("warp-cli", "--accept-tos", "status")
	if err == nil {
		outputStr := strings.ToLower(output)
		return strings.Contains(outputStr, "connected"), nil
	}

	log.Printf("Initial WARP status check failed: %v", err)

	// If that failed, try to check service and then status again
	if err := checkWARPService(); err != nil {
		log.Printf("WARP service check failed: %v", err)
		// Don't return error, try once more after service check
	}

	output, err = runCommand("warp-cli", "--accept-tos", "status")
	if err != nil {
		log.Printf("WARP status check failed after service check: %v", err)
		// Instead of error, return false as status
		return false, nil
	}

	outputStr := strings.ToLower(output)
	return strings.Contains(outputStr, "connected"), nil
}

// GetExcludedDomains returns the list of domains currently excluded from WARP
func GetExcludedDomains() ([]string, error) {
	domains := []string{}

	// Try different commands to get domain list
	domainCommandAttempts := [][]string{
		{"warp-cli", "--accept-tos", "proxy-list"},
		{"warp-cli", "--accept-tos", "excluded-routes", "list"},
		{"warp-cli", "--accept-tos", "routing", "list"},
	}

	for _, cmdArgs := range domainCommandAttempts {
		log.Printf("Trying to get domain list with command: %v", cmdArgs)
		output, err := runCommand(cmdArgs[0], cmdArgs[1:]...)
		if err != nil {
			log.Printf("Domain list attempt failed: %v", err)
			continue
		}

		// Parse the output to extract domains
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip header lines and empty lines
			if line != "" &&
				!strings.HasPrefix(line, "Domains") &&
				!strings.HasPrefix(line, "Route") &&
				!strings.HasPrefix(line, "Name") &&
				!strings.HasPrefix(line, "-") &&
				!strings.HasPrefix(line, "=") {
				// Extract domain name if it's in a table format
				fields := strings.Fields(line)
				if len(fields) > 0 {
					domainField := fields[0]
					// Remove any trailing commas or other punctuation
					domainField = strings.TrimRight(domainField, ",.:;")
					domains = append(domains, domainField)
				}
			}
		}

		// If we found domains, return them
		if len(domains) > 0 {
			return domains, nil
		}
	}

	// If we get here, we haven't found any domains with any command
	if len(domains) == 0 {
		log.Println("No domains found in WARP configuration")
	}

	return domains, nil
}
