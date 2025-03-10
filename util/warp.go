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

// InstallWARP installs Cloudflare WARP if not already installed
func InstallWARP() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Windows is not supported for WARP installation")
	}

	// Check if warp-cli is installed
	_, err := exec.LookPath("warp-svc")
	if err == nil {
		return nil // Already installed
	}

	// Add Cloudflare GPG key
	cmd := exec.Command("curl", "https://pkg.cloudflareclient.com/pubkey.gpg", "--output", "/tmp/cloudflare.gpg")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download Cloudflare GPG key: %v", err)
	}

	// Add key to apt
	cmd = exec.Command("sudo", "apt-key", "add", "/tmp/cloudflare.gpg")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add Cloudflare GPG key: %v", err)
	}

	// Add Cloudflare repository
	cmd = exec.Command("sh", "-c", "echo 'deb [arch=amd64] http://pkg.cloudflareclient.com/ focal main' | sudo tee /etc/apt/sources.list.d/cloudflare-client.list")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add Cloudflare repository: %v", err)
	}

	// Update package list
	cmd = exec.Command("sudo", "apt", "update")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update package list: %v", err)
	}

	// Install cloudflare-warp
	cmd = exec.Command("sudo", "apt", "install", "cloudflare-warp", "-y")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install cloudflare-warp: %v", err)
	}

	// Start WARP service
	cmd = exec.Command("sudo", "systemctl", "start", "warp-svc")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start WARP service: %v", err)
	}

	// Wait for service to be ready
	time.Sleep(5 * time.Second)

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

	// Check if service is running
	cmd := exec.Command("systemctl", "is-active", "warp-svc")
	if err := cmd.Run(); err != nil {
		// Try to start the service
		startCmd := exec.Command("sudo", "systemctl", "start", "warp-svc")
		if err := startCmd.Run(); err != nil {
			return fmt.Errorf("failed to start WARP service: %v", err)
		}
		// Wait for service to be ready
		time.Sleep(5 * time.Second)
	}

	if !enabled {
		// Disable WARP
		cmd := exec.Command("warp-cli", "disconnect")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disconnect WARP: %v", err)
		}
		return nil
	}

	// Register WARP (this will be skipped if already registered)
	cmd = exec.Command("warp-cli", "register")
	cmd.Run() // Ignore error as it might already be registered

	// Set mode to WARP
	cmd = exec.Command("warp-cli", "set-mode", "warp")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set WARP mode: %v", err)
	}

	// Clear existing split tunnel rules
	cmd = exec.Command("warp-cli", "delete", "rules")
	cmd.Run() // Ignore error as rules might not exist

	// Add domains to split tunnel
	for _, domain := range domains {
		cmd = exec.Command("warp-cli", "add-excluded-route", domain)
		if err := cmd.Run(); err != nil {
			log.Printf("Warning: failed to add domain %s to excluded routes: %v", domain, err)
			continue
		}
	}

	// Connect WARP
	cmd = exec.Command("warp-cli", "connect")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to connect WARP: %v", err)
	}

	// Enable always-on mode
	cmd = exec.Command("warp-cli", "enable-always-on")
	cmd.Run() // Ignore error as it's optional

	return nil
}

// GetWARPStatus returns the current WARP connection status
func GetWARPStatus() (bool, error) {
	if runtime.GOOS == "windows" {
		return false, fmt.Errorf("Windows is not supported for WARP status check")
	}

	cmd := exec.Command("warp-cli", "status")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get WARP status: %v", err)
	}

	outputStr := strings.ToLower(string(output))
	return strings.Contains(outputStr, "connected"), nil
}
