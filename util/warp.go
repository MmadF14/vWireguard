package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MmadF14/vwireguard/model"
)

const warpConfigTemplate = `[Interface]
PrivateKey = {{ .PrivateKey }}
Address = {{ .Address }}
DNS = {{ .DNS }}

[Peer]
PublicKey = {{ .PublicKey }}
AllowedIPs = {{ .AllowedIPs }}
Endpoint = {{ .Endpoint }}
`

// GenerateWARPConfig generates the WARP configuration file
func GenerateWARPConfig(settings *model.GlobalSetting) error {
	if !settings.WARPEnabled || len(settings.WARPDomains) == 0 {
		return nil
	}

	// Create WARP config directory if it doesn't exist
	warpDir := "/etc/wireguard/warp"
	if err := os.MkdirAll(warpDir, 0755); err != nil {
		return fmt.Errorf("failed to create WARP directory: %v", err)
	}

	// Generate WARP rules
	var rules []string
	for _, domain := range settings.WARPDomains {
		if domain = strings.TrimSpace(domain); domain != "" {
			rules = append(rules, fmt.Sprintf("*.%s", domain))
			rules = append(rules, domain)
		}
	}

	// Generate exclude rules
	var excludes []string
	for _, domain := range settings.WARPExclude {
		if domain = strings.TrimSpace(domain); domain != "" {
			excludes = append(excludes, fmt.Sprintf("*.%s", domain))
			excludes = append(excludes, domain)
		}
	}

	// Create warp-rules.conf
	rulesPath := filepath.Join(warpDir, "warp-rules.conf")
	rulesContent := strings.Join(rules, "\n")
	if err := os.WriteFile(rulesPath, []byte(rulesContent), 0644); err != nil {
		return fmt.Errorf("failed to write WARP rules: %v", err)
	}

	// Create warp-excludes.conf if there are exclude rules
	if len(excludes) > 0 {
		excludesPath := filepath.Join(warpDir, "warp-excludes.conf")
		excludesContent := strings.Join(excludes, "\n")
		if err := os.WriteFile(excludesPath, []byte(excludesContent), 0644); err != nil {
			return fmt.Errorf("failed to write WARP excludes: %v", err)
		}
	}

	return nil
}

// UpdateWARPConfig updates the WARP configuration when settings change
func UpdateWARPConfig(settings *model.GlobalSetting) error {
	if err := GenerateWARPConfig(settings); err != nil {
		return err
	}

	// Restart WARP service if it exists
	if _, err := os.Stat("/etc/systemd/system/warp.service"); err == nil {
		if err := RestartService("warp"); err != nil {
			return fmt.Errorf("failed to restart WARP service: %v", err)
		}
	}

	return nil
}
