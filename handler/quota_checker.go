package handler

import (
    "time"
    "github.com/ngoduykhanh/wireguard-ui/store"
    "github.com/ngoduykhanh/wireguard-ui/model"
    "github.com/ngoduykhanh/wireguard-ui/util"
    "golang.zx2c4.com/wireguard/wgctrl"
)

func StartQuotaAndExpirationChecker(db store.IStore) {
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()

        for {
            <-ticker.C
            checkQuotaAndExpiration(db)
        }
    }()
}

func checkQuotaAndExpiration(db store.IStore) {
    clients, err := db.GetClients(false)
    if err != nil {
        return
    }

    // از wgctrl یا هر روش دیگری برای گرفتن میزان ترافیک مصرفی استفاده کنید
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
    usageMap := make(map[string][2]uint64)

    for _, dev := range devices {
        for _, peer := range dev.Peers {
            usageMap[peer.PublicKey.String()] = [2]uint64{
                peer.ReceiveBytes,
                peer.TransmitBytes,
            }
        }
    }

    for _, cData := range clients {
        client := cData.Client

        // اگر Expiration تنظیم شده و زمانش گذشته است
        if !client.Expiration.IsZero() && time.Now().After(client.Expiration) {
            client.Enabled = false
            db.SaveClient(*client)
            // می‌توانید لاگ بگیرید یا به کاربر ایمیل بزنید
            continue
        }

        // اگر Quota تنظیم شده و > 0 باشد
        if client.Quota > 0 {
            // گرفتن میزان مصرف از usageMap
            if usage, ok := usageMap[client.PublicKey]; ok {
                total := usage[0] + usage[1] // جمع ارسال و دریافت
                // اگر از مقدار Quota گذشت
                if int64(total) > client.Quota {
                    client.Enabled = false
                    db.SaveClient(*client)
                    // اطلاع‌رسانی یا لاگ
                }
            }
        }
    }
}
