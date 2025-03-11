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

// checkWARPService checks if WARP service is installed and running
func checkWARPService() error {
	// Check if warp-svc exists
	if _, err := exec.LookPath("warp-svc"); err != nil {
		return fmt.Errorf("WARP service is not installed")
	}

	// Check if service is running
	output, err := runCommand("systemctl", "is-active", "warp-svc")
	if err != nil || !strings.Contains(strings.ToLower(output), "active") {
		return fmt.Errorf("WARP service is not running")
	}

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

	// First try to install using apt directly
	if _, err := runCommand("sudo", "apt", "update"); err != nil {
		log.Printf("Warning: apt update failed: %v", err)
	}

	log.Println("Installing cloudflare-warp package...")
	if output, err := runCommand("sudo", "apt", "install", "-y", "cloudflare-warp"); err != nil {
		// If direct installation fails, try adding the repository first
		log.Printf("Direct installation failed: %v\nTrying with repository setup...", err)

		// Download and add GPG key
		log.Println("Adding Cloudflare GPG key...")
		if _, err := runCommand("curl", "-fsSL", "https://pkg.cloudflareclient.com/pubkey.gpg", "--output", "/tmp/cloudflare.gpg"); err != nil {
			return fmt.Errorf("failed to download GPG key: %v", err)
		}

		if _, err := runCommand("sudo", "apt-key", "add", "/tmp/cloudflare.gpg"); err != nil {
			return fmt.Errorf("failed to add GPG key: %v", err)
		}

		// Add repository
		log.Println("Adding Cloudflare repository...")
		repoCmd := `echo "deb [arch=amd64] https://pkg.cloudflareclient.com/ $(lsb_release -sc) main" | sudo tee /etc/apt/sources.list.d/cloudflare-client.list`
		if _, err := runCommand("bash", "-c", repoCmd); err != nil {
			return fmt.Errorf("failed to add repository: %v", err)
		}

		// Update and install
		log.Println("Updating package list and installing WARP...")
		if _, err := runCommand("sudo", "apt", "update"); err != nil {
			return fmt.Errorf("failed to update package list: %v", err)
		}

		if _, err := runCommand("sudo", "apt", "install", "-y", "cloudflare-warp"); err != nil {
			return fmt.Errorf("failed to install WARP: %v", err)
		}
	} else {
		log.Printf("Installation output: %s", output)
	}

	// Ensure the service is running
	log.Println("Starting WARP service...")
	if _, err := runCommand("sudo", "systemctl", "start", "warp-svc"); err != nil {
		return fmt.Errorf("failed to start WARP service: %v", err)
	}

	// Wait for service to be ready with timeout
	log.Println("Waiting for WARP service to be ready...")
	if err := waitForWARPService(30 * time.Second); err != nil {
		return fmt.Errorf("WARP service failed to become ready: %v", err)
	}

	return nil
}

// ConfigureWARP configures WARP with the specified domains
func ConfigureWARP(enabled bool, domains []string) error {
	log.Printf("Configuring WARP (enabled=%v, domains=%v)...", enabled, domains)

	// Check WARP service status
	if err := checkWARPService(); err != nil {
		return fmt.Errorf("WARP service check failed: %v", err)
	}

	if !enabled {
		log.Println("Disabling WARP...")
		if _, err := runCommand("warp-cli", "--accept-tos", "disconnect"); err != nil {
			return fmt.Errorf("failed to disconnect WARP: %v", err)
		}
		return nil
	}

	// Delete old registration if exists
	log.Println("Checking for existing registration...")
	if _, err := runCommand("warp-cli", "--accept-tos", "registration", "delete"); err != nil {
		log.Printf("Warning: failed to delete old registration: %v", err)
	}

	// Initialize WARP
	log.Println("Initializing WARP...")
	if _, err := runCommand("warp-cli", "--accept-tos", "registration", "new"); err != nil {
		return fmt.Errorf("failed to initialize WARP: %v", err)
	}

	// Set mode to proxy
	log.Println("Setting WARP mode to proxy...")
	if _, err := runCommand("warp-cli", "--accept-tos", "mode", "proxy"); err != nil {
		return fmt.Errorf("failed to set WARP mode: %v", err)
	}

	// Enable split tunnel mode
	log.Println("Enabling split tunnel mode...")
	if _, err := runCommand("warp-cli", "--accept-tos", "disable-dns-log"); err != nil {
		log.Printf("Warning: failed to disable DNS logging: %v", err)
	}

	// Clear any existing rules
	log.Println("Clearing existing proxy rules...")
	if _, err := runCommand("warp-cli", "--accept-tos", "proxy-port", "40000"); err != nil {
		return fmt.Errorf("failed to set proxy port: %v", err)
	}

	// Add domains to proxy rules
	log.Println("Adding domains to proxy rules...")
	for _, domain := range domains {
		if _, err := runCommand("warp-cli", "--accept-tos", "add-proxy-domain", domain); err != nil {
			log.Printf("Warning: failed to add domain %s to proxy rules: %v", domain, err)
			continue
		}
		log.Printf("Added domain to proxy: %s", domain)
	}

	// Connect WARP
	log.Println("Connecting WARP...")
	if _, err := runCommand("warp-cli", "--accept-tos", "connect"); err != nil {
		return fmt.Errorf("failed to connect WARP: %v", err)
	}

	// Wait for connection to be established
	if err := waitForWARPService(30 * time.Second); err != nil {
		return fmt.Errorf("WARP failed to connect: %v", err)
	}

	log.Println("WARP configuration completed successfully")
	return nil
}

// GetWARPStatus returns the current WARP connection status
func GetWARPStatus() (bool, error) {
	// Check service status first
	if err := checkWARPService(); err != nil {
		return false, err
	}

	output, err := runCommand("warp-cli", "--accept-tos", "status")
	if err != nil {
		return false, fmt.Errorf("failed to get WARP status: %v", err)
	}

	outputStr := strings.ToLower(output)
	return strings.Contains(outputStr, "connected"), nil
}

// GetExcludedDomains returns the list of domains currently excluded from WARP
func GetExcludedDomains() ([]string, error) {
	// Check service status first
	if err := checkWARPService(); err != nil {
		return nil, err
	}

	// Try the newer command first
	output, err := runCommand("warp-cli", "--accept-tos", "proxy-list")
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy domains: %v", err)
	}

	// Parse the output to extract domains
	domains := []string{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Domains") && !strings.HasPrefix(line, "-") {
			domains = append(domains, line)
		}
	}

	return domains, nil
}
