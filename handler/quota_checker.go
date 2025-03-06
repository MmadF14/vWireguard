package handler

import (
    "log"
    "time"
    "github.com/ngoduykhanh/wireguard-ui/model"
    "github.com/ngoduykhanh/wireguard-ui/store"
    "golang.zx2c4.com/wireguard/wgctrl"
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

    for _, cData := range clients {
        client := cData.Client

        // بررسی Expiration
        if !client.Expiration.IsZero() && time.Now().After(client.Expiration) {
            client.Enabled = false
            db.SaveClient(*client)
            log.Printf("Client %s disabled due to expiration", client.Name)
            continue
        }

        // بررسی Quota
        if client.Quota > 0 {
            if usage, ok := usageMap[client.PublicKey]; ok {
                total := usage[0] + usage[1] // جمع ارسال و دریافت به صورت uint64
                // چون client.Quota از نوع int64 است، اینجا آن را به int64 تبدیل می‌کنیم
                if int64(total) > client.Quota {
                    client.Enabled = false
                    db.SaveClient(*client)
                    log.Printf("Client %s disabled due to quota limit", client.Name)
                }
            }
        }
    }
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
