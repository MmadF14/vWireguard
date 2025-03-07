package handler

import (
    "log"
    "sync"
    "time"
    "github.com/MmadF14/wireguard-ui/store"
    "golang.zx2c4.com/wireguard/wgctrl"
    "fmt"
    "github.com/MmadF14/wireguard-ui/util"
    "strings"
    "os/exec"
)

var (
    configMutex sync.Mutex
)

// StartQuotaChecker starts a goroutine that periodically checks client quotas and expiration dates
func StartQuotaChecker(db store.IStore) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Recovered from panic in quota checker: %v", r)
                // Restart the goroutine after a short delay
                time.Sleep(10 * time.Second)
                StartQuotaChecker(db)
            }
        }()

        for {
            func() {
                defer func() {
                    if r := recover(); r != nil {
                        log.Printf("Recovered from panic in check cycle: %v", r)
                    }
                }()
                checkQuotasAndExpiration(db)
            }()
            time.Sleep(5 * time.Minute)
        }
    }()
}

// checkQuotasAndExpiration checks all clients for quota limits and expiration dates
func checkQuotasAndExpiration(db store.IStore) {
    log.Printf("Starting quota and expiration check")
    // دریافت لیست تمام کلاینت‌ها
    clients, err := db.GetClients(false)
    if err != nil {
        log.Printf("Error getting clients for quota check: %v", err)
        return
    }
    log.Printf("Successfully retrieved %d clients", len(clients))

    // دریافت آمار ترافیک از WireGuard
    usageMap, err := getWireGuardUsage()
    if err != nil {
        log.Printf("Error getting WireGuard usage: %v", err)
        return
    }
    log.Printf("Successfully retrieved WireGuard usage stats")

    // Get settings for interface name
    settings, err := db.GetGlobalSettings()
    if err != nil {
        log.Printf("Error getting global settings: %v", err)
        return
    }

    // Get interface name
    interfaceName := "wg0"
    if settings.ConfigFilePath != "" {
        parts := strings.Split(settings.ConfigFilePath, "/")
        if len(parts) > 0 {
            baseName := parts[len(parts)-1]
            interfaceName = strings.TrimSuffix(baseName, ".conf")
        }
    }

    configChanged := false

    for _, cData := range clients {
        client := cData.Client
        if client == nil {
            continue
        }

        // فقط کلاینت‌های فعال را بررسی می‌کنیم
        if !client.Enabled {
            continue
        }

        log.Printf("Checking client: %s", client.Name)
        wasEnabled := client.Enabled

        // بروزرسانی مصرف کلاینت
        if usage, ok := usageMap[client.PublicKey]; ok {
            total := usage[0] + usage[1] // جمع ارسال و دریافت
            client.UsedQuota = int64(total)
            if err := db.SaveClient(*client); err != nil {
                log.Printf("Error saving client %s usage data: %v", client.Name, err)
                continue
            }
            log.Printf("Client %s usage updated: %d bytes", client.Name, total)
        }

        shouldDisable := false
        disableReason := ""

        // بررسی Expiration - اگر تاریخ انقضا تنظیم نشده باشد (zero time)، به معنی unlimited است
        if !client.Expiration.IsZero() && time.Now().After(client.Expiration) {
            shouldDisable = true
            disableReason = "expiration"
        }

        // بررسی Quota
        if client.Quota > 0 {
            if usage, ok := usageMap[client.PublicKey]; ok {
                total := usage[0] + usage[1]
                if int64(total) > client.Quota {
                    shouldDisable = true
                    disableReason = "quota"
                }
            }
        }

        // اگر نیاز به غیرفعال کردن کلاینت باشد
        if shouldDisable {
            client.Enabled = false
            if err := db.SaveClient(*client); err != nil {
                log.Printf("Error saving client %s after %s check: %v", client.Name, disableReason, err)
                continue
            }

            // غیرفعال‌سازی خودکار کلاینت
            cmd := exec.Command("sudo", "wg", "set", interfaceName, "peer", client.PublicKey, "remove")
            if err := cmd.Run(); err != nil {
                log.Printf("Error removing peer %s: %v", client.Name, err)
            } else {
                log.Printf("Successfully removed peer %s", client.Name)
            }

            log.Printf("Client %s disabled due to %s", client.Name, disableReason)
            configChanged = true
        }

        // اگر وضعیت کلاینت تغییر کرده، نیاز به اعمال تغییرات داریم
        if wasEnabled != client.Enabled {
            configChanged = true
        }
    }

    // اگر تغییری در وضعیت کلاینت‌ها داشتیم، فقط فایل کانفیگ را به‌روز می‌کنیم
    if configChanged {
        server, err := db.GetServer()
        if err != nil {
            log.Printf("Error getting server config: %v", err)
            return
        }

        users, err := db.GetUsers()
        if err != nil {
            log.Printf("Error getting users config: %v", err)
            return
        }

        // Write config file
        if err := util.WriteWireGuardServerConfig(nil, server, clients, users, settings); err != nil {
            log.Printf("Error writing WireGuard config: %v", err)
            return
        }
        log.Printf("Successfully updated WireGuard config file")
    }
}

// applyWireGuardConfig applies the current configuration to WireGuard
func applyWireGuardConfig(db store.IStore) error {
    configMutex.Lock()
    defer configMutex.Unlock()
    
    log.Printf("Starting to apply WireGuard config")
    server, err := db.GetServer()
    if err != nil {
        log.Printf("Error getting server config: %v", err)
        return fmt.Errorf("cannot get server config: %v", err)
    }
    log.Printf("Successfully got server config")

    clients, err := db.GetClients(false)
    if err != nil {
        log.Printf("Error getting client config: %v", err)
        return fmt.Errorf("cannot get client config: %v", err)
    }
    log.Printf("Successfully got %d clients", len(clients))

    users, err := db.GetUsers()
    if err != nil {
        log.Printf("Error getting users: %v", err)
        return fmt.Errorf("cannot get users config: %v", err)
    }
    log.Printf("Successfully got users")

    settings, err := db.GetGlobalSettings()
    if err != nil {
        log.Printf("Error getting global settings: %v", err)
        return fmt.Errorf("cannot get global settings: %v", err)
    }
    log.Printf("Successfully got global settings")

    // Write config file
    if err := util.WriteWireGuardServerConfig(nil, server, clients, users, settings); err != nil {
        log.Printf("Error writing WireGuard config: %v", err)
        return fmt.Errorf("cannot write config: %v", err)
    }
    log.Printf("Successfully wrote WireGuard config")

    // Get interface name from config file path
    interfaceName := "wg0"
    if settings.ConfigFilePath != "" {
        parts := strings.Split(settings.ConfigFilePath, "/")
        if len(parts) > 0 {
            baseName := parts[len(parts)-1]
            interfaceName = strings.TrimSuffix(baseName, ".conf")
        }
    }

    // Restart WireGuard service using systemctl
    serviceName := fmt.Sprintf("wg-quick@%s", interfaceName)
    
    // Execute systemctl restart with output capture
    cmd := exec.Command("sudo", "systemctl", "restart", serviceName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("Error restarting WireGuard service: %v, Output: %s", err, string(output))
        return fmt.Errorf("error restarting WireGuard service: %v, Output: %s", err, string(output))
    }

    // Verify service is active
    checkCmd := exec.Command("sudo", "systemctl", "is-active", serviceName)
    status, err := checkCmd.CombinedOutput()
    if err != nil || strings.TrimSpace(string(status)) != "active" {
        log.Printf("WireGuard service is not active after restart. Status: %s", string(status))
        return fmt.Errorf("WireGuard service is not active after restart")
    }

    log.Printf("Successfully restarted WireGuard service and verified it is active")
    return nil
}

// getWireGuardUsage returns a map of public keys to their traffic usage [received, sent]
func getWireGuardUsage() (map[string][2]uint64, error) {
    log.Printf("Starting to get WireGuard usage")
    usageMap := make(map[string][2]uint64)
    
    wgClient, err := wgctrl.New()
    if err != nil {
        log.Printf("Error creating WireGuard client: %v", err)
        return nil, err
    }
    defer wgClient.Close()
    log.Printf("Successfully created WireGuard client")

    devices, err := wgClient.Devices()
    if err != nil {
        log.Printf("Error getting WireGuard devices: %v", err)
        return nil, err
    }
    log.Printf("Found %d WireGuard devices", len(devices))

    for _, dev := range devices {
        log.Printf("Processing device: %s", dev.Name)
        for _, peer := range dev.Peers {
            usageMap[peer.PublicKey.String()] = [2]uint64{
                uint64(peer.ReceiveBytes),
                uint64(peer.TransmitBytes),
            }
        }
    }
    
    return usageMap, nil
}
