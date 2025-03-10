package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// runCommand executes a command and returns its output and error
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command '%s %s' failed: %v, output: %s", name, strings.Join(args, " "), err, string(output))
	}
	return string(output), nil
}

// InstallWARP installs Cloudflare WARP if not already installed
func InstallWARP() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Windows is not supported for WARP installation")
	}

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
		log.Printf("Warning: failed to start warp-svc: %v", err)
	}

	// Wait for service to be ready
	log.Println("Waiting for WARP service to be ready...")
	time.Sleep(10 * time.Second)

	return nil
}

// getWarpPath returns the path to warp-cli based on the operating system
func getWarpPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramFiles"), "Cloudflare", "Cloudflare WARP", "warp-cli.exe")
	}
	return "/usr/bin/warp-cli"
}

// ConfigureWARP configures WARP with the specified domains
func ConfigureWARP(enabled bool, domains []string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Windows is not supported for WARP configuration")
	}

	log.Printf("Configuring WARP (enabled=%v, domains=%v)...", enabled, domains)

	// Check if service is running
	if _, err := runCommand("systemctl", "is-active", "warp-svc"); err != nil {
		log.Println("WARP service is not active, attempting to start...")
		if _, err := runCommand("sudo", "systemctl", "start", "warp-svc"); err != nil {
			return fmt.Errorf("failed to start WARP service: %v", err)
		}
		time.Sleep(10 * time.Second)
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

	// Enable WARP mode
	log.Println("Setting WARP mode...")
	if _, err := runCommand("warp-cli", "--accept-tos", "mode", "warp"); err != nil {
		return fmt.Errorf("failed to set WARP mode: %v", err)
	}

	// Configure split tunnel for domains
	log.Println("Configuring split tunnel for domains...")
	for _, domain := range domains {
		// First try to exclude the domain
		if _, err := runCommand("warp-cli", "--accept-tos", "add-excluded-domain", domain); err != nil {
			// If that fails, try the older split-tunnel command
			if _, err := runCommand("warp-cli", "--accept-tos", "split-tunnel", "add", domain); err != nil {
				log.Printf("Warning: failed to add domain %s to split tunnel: %v", domain, err)
				continue
			}
		}
		log.Printf("Added domain: %s", domain)
	}

	// Connect WARP
	log.Println("Connecting WARP...")
	if _, err := runCommand("warp-cli", "--accept-tos", "connect"); err != nil {
		return fmt.Errorf("failed to connect WARP: %v", err)
	}

	// Enable always-on mode using the correct command
	log.Println("Enabling always-on mode...")
	if _, err := runCommand("warp-cli", "--accept-tos", "always-on", "on"); err != nil {
		log.Printf("Warning: failed to enable always-on mode: %v", err)
	}

	log.Println("WARP configuration completed successfully")
	return nil
}

// GetWARPStatus returns the current WARP connection status
func GetWARPStatus() (bool, error) {
	if runtime.GOOS == "windows" {
		return false, fmt.Errorf("Windows is not supported for WARP status check")
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
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("Windows is not supported for WARP configuration")
	}

	// Try the newer command first
	output, err := runCommand("warp-cli", "--accept-tos", "show-excluded-domains")
	if err != nil {
		// If that fails, try the older split-tunnel command
		output, err = runCommand("warp-cli", "--accept-tos", "split-tunnel", "list")
		if err != nil {
			return nil, fmt.Errorf("failed to get excluded domains: %v", err)
		}
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
