package handler

import (
    "log"
    "time"
    "github.com/MmadF14/wireguard-ui/store"
    "golang.zx2c4.com/wireguard/wgctrl"
    "fmt"
    "github.com/MmadF14/wireguard-ui/util"
    "strings"
    "os/exec"
)

// StartQuotaChecker starts a goroutine that periodically checks client quotas and expiration dates
func StartQuotaChecker(db store.IStore) {
    go func() {
        for {
            checkQuotasAndExpiration(db)
            time.Sleep(5 * time.Minute) // هر 5 دقیقه چک می‌کنیم
        }
    }()
}

// checkQuotasAndExpiration checks all clients for quota limits and expiration dates
func checkQuotasAndExpiration(db store.IStore) {
    // دریافت لیست تمام کلاینت‌ها
    clients, err := db.GetClients(false)
    if err != nil {
        log.Printf("Error getting clients for quota check: %v", err)
        return
    }

    // دریافت آمار ترافیک از WireGuard
    usageMap, err := getWireGuardUsage()
    if err != nil {
        log.Printf("Error getting WireGuard usage: %v", err)
        return
    }

    configChanged := false

    for _, cData := range clients {
        client := cData.Client
        wasEnabled := client.Enabled

        // بررسی Expiration - اگر تاریخ انقضا تنظیم نشده باشد (zero time)، به معنی unlimited است
        if !client.Expiration.IsZero() {  // فقط اگر تاریخ انقضا تنظیم شده باشد، چک می‌کنیم
            if time.Now().After(client.Expiration) {
                if client.Enabled {
                    client.Enabled = false
                    if err := db.SaveClient(*client); err != nil {
                        log.Printf("Error saving client %s after expiration: %v", client.Name, err)
                        continue
                    }
                    log.Printf("Client %s disabled due to expiration", client.Name)
                    configChanged = true
                }
                continue
            }
        }

        // بررسی Quota
        if client.Quota > 0 {
            if usage, ok := usageMap[client.PublicKey]; ok {
                total := usage[0] + usage[1] // جمع ارسال و دریافت به صورت uint64
                // چون client.Quota از نوع int64 است، اینجا آن را به int64 تبدیل می‌کنیم
                if int64(total) > client.Quota && client.Enabled {
                    client.Enabled = false
                    if err := db.SaveClient(*client); err != nil {
                        log.Printf("Error saving client %s after quota exceeded: %v", client.Name, err)
                        continue
                    }
                    log.Printf("Client %s disabled due to quota limit", client.Name)
                    configChanged = true
                }
            }
        }

        // اگر وضعیت کلاینت تغییر کرده، نیاز به اعمال تغییرات داریم
        if wasEnabled != client.Enabled {
            configChanged = true
        }
    }

    // اگر تغییری در وضعیت کلاینت‌ها داشتیم، پیکربندی WireGuard را به‌روز می‌کنیم
    if configChanged {
        if err := applyWireGuardConfig(db); err != nil {
            log.Printf("Error applying WireGuard config: %v", err)
        }
    }
}

// applyWireGuardConfig applies the current configuration to WireGuard
func applyWireGuardConfig(db store.IStore) error {
    server, err := db.GetServer()
    if err != nil {
        return fmt.Errorf("cannot get server config: %v", err)
    }

    clients, err := db.GetClients(false)
    if err != nil {
        return fmt.Errorf("cannot get client config: %v", err)
    }

    users, err := db.GetUsers()
    if err != nil {
        return fmt.Errorf("cannot get users config: %v", err)
    }

    settings, err := db.GetGlobalSettings()
    if err != nil {
        return fmt.Errorf("cannot get global settings: %v", err)
    }

    // Write config file
    if err := util.WriteWireGuardServerConfig(nil, server, clients, users, settings); err != nil {
        return fmt.Errorf("cannot write config: %v", err)
    }

    // Get interface name from config file path
    interfaceName := "wg0"
    if settings.ConfigFilePath != "" {
        parts := strings.Split(settings.ConfigFilePath, "/")
        if len(parts) > 0 {
            baseName := parts[len(parts)-1]
            interfaceName = strings.TrimSuffix(baseName, ".conf")
        }
    }

    // Reload WireGuard service using systemctl
    serviceName := fmt.Sprintf("wg-quick@%s.service", interfaceName)
    
    // Reload the service configuration
    cmd := exec.Command("systemctl", "reload", serviceName)
    if err := cmd.Run(); err != nil {
        // If reload fails, try restart as fallback
        cmd = exec.Command("systemctl", "restart", serviceName)
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("error restarting WireGuard service: %v", err)
        }
    }

    return nil
}

// getWireGuardUsage returns a map of public keys to their traffic usage [received, sent]
func getWireGuardUsage() (map[string][2]uint64, error) {
    usageMap := make(map[string][2]uint64)
    
    wgClient, err := wgctrl.New()
    if err != nil {
        return nil, err
    }
    defer wgClient.Close()

    devices, err := wgClient.Devices()
    if err != nil {
        return nil, err
    }

    for _, dev := range devices {
        for _, peer := range dev.Peers {
            usageMap[peer.PublicKey.String()] = [2]uint64{
                uint64(peer.ReceiveBytes),
                uint64(peer.TransmitBytes),
            }
        }
    }
    
    return usageMap, nil
}
