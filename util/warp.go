package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallWARP installs Cloudflare WARP if not already installed
func InstallWARP() error {
	// Check if warp-cli is installed
	warpPath := getWarpPath()
	if _, err := os.Stat(warpPath); err == nil {
		return nil // Already installed
	}

	if runtime.GOOS == "windows" {
		// Download Windows installer
		downloadURL := "https://1111-releases.cloudflareclient.com/windows/Cloudflare_WARP_Release-x64.msi"
		installerPath := filepath.Join(os.TempDir(), "warp_installer.msi")

		// Download installer
		cmd := exec.Command("powershell", "-Command",
			fmt.Sprintf("Invoke-WebRequest -Uri '%s' -OutFile '%s'", downloadURL, installerPath))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to download WARP installer: %v", err)
		}

		// Install silently
		cmd = exec.Command("msiexec", "/i", installerPath, "/quiet", "/norestart")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install WARP: %v", err)
		}

		// Clean up installer
		os.Remove(installerPath)
		return nil
	} else {
		// Linux installation
		// Add Cloudflare GPG key
		cmd := exec.Command("curl", "https://pkg.cloudflareclient.com/pubkey.gpg", "-o", "/tmp/cloudflare.gpg")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to download Cloudflare GPG key: %v", err)
		}

		// Add key to apt
		cmd = exec.Command("sudo", "apt-key", "add", "/tmp/cloudflare.gpg")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add Cloudflare GPG key: %v", err)
		}

		// Add Cloudflare repository
		repoContent := "deb http://pkg.cloudflareclient.com/ focal main"
		err := os.WriteFile("/etc/apt/sources.list.d/cloudflare-client.list", []byte(repoContent), 0644)
		if err != nil {
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

		return nil
	}
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
	warpPath := getWarpPath()

	if !enabled {
		// Disable WARP
		cmd := exec.Command(warpPath, "disconnect")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disconnect WARP: %v", err)
		}
		return nil
	}

	// Register WARP if not already registered
	cmd := exec.Command(warpPath, "register")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to register WARP: %v", err)
	}

	// Connect WARP
	cmd = exec.Command(warpPath, "connect")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to connect WARP: %v", err)
	}

	// Configure split tunnel mode
	cmd = exec.Command(warpPath, "enable-always-on")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable always-on mode: %v", err)
	}

	// Add domains to split tunnel
	for _, domain := range domains {
		cmd = exec.Command(warpPath, "add-split-tunnel", domain)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add domain %s to split tunnel: %v", domain, err)
		}
	}

	return nil
}

// GetWARPStatus returns the current WARP connection status
func GetWARPStatus() (bool, error) {
	warpPath := getWarpPath()
	cmd := exec.Command(warpPath, "status")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get WARP status: %v", err)
	}

	return strings.Contains(string(output), "Connected"), nil
}
