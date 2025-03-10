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
		if _, err := runCommand("warp-cli", "disconnect"); err != nil {
			return fmt.Errorf("failed to disconnect WARP: %v", err)
		}
		return nil
	}

	// Initialize WARP
	log.Println("Initializing WARP...")
	if _, err := runCommand("warp-cli", "--accept-tos", "init"); err != nil {
		log.Printf("Warning: WARP initialization failed: %v", err)
	}

	// Register WARP
	log.Println("Registering WARP...")
	if _, err := runCommand("warp-cli", "--accept-tos", "register"); err != nil {
		log.Printf("Warning: WARP registration failed: %v", err)
	}

	// Set mode to WARP
	log.Println("Setting WARP mode...")
	if _, err := runCommand("warp-cli", "set-mode", "warp"); err != nil {
		return fmt.Errorf("failed to set WARP mode: %v", err)
	}

	// Clear existing rules
	log.Println("Clearing existing rules...")
	if _, err := runCommand("warp-cli", "delete", "rules"); err != nil {
		log.Printf("Warning: failed to clear rules: %v", err)
	}

	// Add domains to split tunnel
	log.Println("Adding domains to split tunnel...")
	for _, domain := range domains {
		if _, err := runCommand("warp-cli", "add-excluded-route", domain); err != nil {
			log.Printf("Warning: failed to add domain %s to excluded routes: %v", domain, err)
			continue
		}
		log.Printf("Added domain: %s", domain)
	}

	// Connect WARP
	log.Println("Connecting WARP...")
	if _, err := runCommand("warp-cli", "connect"); err != nil {
		return fmt.Errorf("failed to connect WARP: %v", err)
	}

	// Enable always-on mode
	log.Println("Enabling always-on mode...")
	if _, err := runCommand("warp-cli", "enable-always-on"); err != nil {
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

	output, err := runCommand("warp-cli", "status")
	if err != nil {
		return false, fmt.Errorf("failed to get WARP status: %v", err)
	}

	outputStr := strings.ToLower(output)
	return strings.Contains(outputStr, "connected"), nil
}
