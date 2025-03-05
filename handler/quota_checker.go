package handler

import (
    "time"

    "github.com/MmadF14/wireguard-ui/model"
    "github.com/MmadF14/wireguard-ui/store"
    // پکیج util حذف شد چون در این کد استفاده‌ای از آن نداریم
    // "github.com/MmadF14/wireguard-ui/util"

    "golang.zx2c4.com/wireguard/wgctrl"
)

func StartQuotaAndExpirationChecker(db store.IStore) {
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()

        for range ticker.C {
            checkQuotaAndExpiration(db)
        }
    }()
}

func checkQuotaAndExpiration(db store.IStore) {
    clients, err := db.GetClients(false)
    if err != nil {
        return
    }

    wgClient, err := wgctrl.New()
    if err != nil {
        // لاگ خطا
        return
    }
    defer wgClient.Close()

    devices, err := wgClient.Devices()
    if err != nil {
        // لاگ خطا
        return
    }

    // map: publicKey => (receivedBytes, transmitBytes)
    // چون ReceiveBytes و TransmitBytes از نوع int64 هستند، آن‌ها را به uint64 تبدیل می‌کنیم
    usageMap := make(map[string][2]uint64)

    for _, dev := range devices {
        for _, peer := range dev.Peers {
            usageMap[peer.PublicKey.String()] = [2]uint64{
                uint64(peer.ReceiveBytes),
                uint64(peer.TransmitBytes),
            }
        }
    }

    for _, cData := range clients {
        client := cData.Client

        // بررسی Expiration
        if !client.Expiration.IsZero() && time.Now().After(client.Expiration) {
            client.Enabled = false
            db.SaveClient(*client)
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
                }
            }
        }
    }
}
