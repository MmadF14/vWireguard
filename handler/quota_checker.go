package handler

import (
	"fmt"
	"io/fs"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/util"
	"golang.zx2c4.com/wireguard/wgctrl"
)

var (
	configMutex sync.Mutex
	// نگهداری زمان آخرین غیرفعال‌سازی برای هر کلاینت
	lastDisableTime     = make(map[string]time.Time)
	lastDisableMutex    sync.RWMutex
	quotaCheckerTmplDir fs.FS
)

// cooldownPeriod مدت زمان انتظار بین غیرفعال‌سازی‌های متوالی
const cooldownPeriod = 5 * time.Minute

// StartQuotaChecker starts a goroutine that periodically checks client quotas and expiration dates
func StartQuotaChecker(db store.IStore, tmplDir fs.FS) {
	quotaCheckerTmplDir = tmplDir
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in quota checker: %v", r)
				time.Sleep(10 * time.Second)
				StartQuotaChecker(db, tmplDir)
			}
		}()

		// اولین بررسی را با تاخیر انجام می‌دهیم تا سیستم کاملاً بالا بیاید
		time.Sleep(30 * time.Second)

		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic in check cycle: %v", r)
					}
				}()
				checkQuotasAndExpiration(db)
			}()

			// از Server Interface بخوانیم
			server, err := db.GetServer()
			if err != nil {
				log.Printf("Error retrieving server config for check interval: %v", err)
				// پیشفرض 5 دقیقه
				time.Sleep(5 * time.Minute)
				continue
			}

			interval := server.Interface.CheckInterval
			// اگر خارج از 1..5 بود، 5 بگذاریم
			if interval < 1 || interval > 5 {
				interval = 5
			}

			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()
}

// isInCooldown checks if a client is in cooldown period
func isInCooldown(clientID string) bool {
	lastDisableMutex.RLock()
	defer lastDisableMutex.RUnlock()

	if lastTime, exists := lastDisableTime[clientID]; exists {
		return time.Since(lastTime) < cooldownPeriod
	}
	return false
}

// setLastDisableTime updates the last disable time for a client
func setLastDisableTime(clientID string) {
	lastDisableMutex.Lock()
	defer lastDisableMutex.Unlock()
	lastDisableTime[clientID] = time.Now()
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

	for _, cData := range clients {
		client := cData.Client
		if client == nil {
			continue
		}

		// فقط کلاینت‌های فعال را بررسی می‌کنیم
		if !client.Enabled {
			continue
		}

		// اگر کلاینت در دوره cooldown است، آن را بررسی نمی‌کنیم
		if isInCooldown(client.ID) {
			log.Printf("Client %s (%s) is in cooldown period, skipping check", client.Name, client.ID)
			continue
		}

		log.Printf("Checking client: %s", client.Name)

		// بروزرسانی مصرف کلاینت
		if usage, ok := usageMap[client.PublicKey]; ok {
			total := usage.Rx + usage.Tx // جمع ارسال و دریافت
			client.UsedQuota = int64(total)
			if client.FirstConnectedAt.IsZero() && !usage.LastHandshake.IsZero() {
				client.FirstConnectedAt = usage.LastHandshake
				if client.ExpirationDays > 0 {
					client.Expiration = client.FirstConnectedAt.Add(time.Duration(client.ExpirationDays) * 24 * time.Hour)
				}
			}
			if err := db.SaveClient(*client); err != nil {
				log.Printf("Error saving client %s usage data: %v", client.Name, err)
				continue
			}
			log.Printf("Client %s usage updated: %d bytes", client.Name, total)
		}

		shouldDisable := false
		disableReason := ""

		// بررسی Expiration - اگر تاریخ انقضا تنظیم نشده باشد (zero time)، به معنی unlimited است
		if client.ExpirationDays > 0 && !client.FirstConnectedAt.IsZero() && client.Expiration.IsZero() {
			client.Expiration = client.FirstConnectedAt.Add(time.Duration(client.ExpirationDays) * 24 * time.Hour)
			if err := db.SaveClient(*client); err != nil {
				log.Printf("Error saving expiration for client %s: %v", client.Name, err)
			}
		}

		if !client.Expiration.IsZero() && time.Now().After(client.Expiration) {
			shouldDisable = true
			disableReason = "expiration"
		}

		// بررسی Quota
		if client.Quota > 0 {
			if usage, ok := usageMap[client.PublicKey]; ok {
				total := usage.Rx + usage.Tx
				if int64(total) > client.Quota {
					shouldDisable = true
					disableReason = "quota"
				}
			}
		}

		// اگر نیاز به غیرفعال کردن کلاینت باشد
		if shouldDisable {
			// غیرفعال‌سازی مستقیم کلاینت
			client.Enabled = false
			if err := db.SaveClient(*client); err != nil {
				log.Printf("Error saving disabled state for client %s: %v", client.Name, err)
				continue
			}

			// ثبت زمان غیرفعال‌سازی
			setLastDisableTime(client.ID)
			log.Printf("Client %s disabled due to %s", client.Name, disableReason)

			// اعمال مستقیم کانفیگ
			if err := applyWireGuardConfig(db); err != nil {
				log.Printf("Error applying WireGuard config after disabling client %s: %v", client.Name, err)
				// ادامه می‌دهیم چون کلاینت در هر صورت غیرفعال شده است
			} else {
				log.Printf("WireGuard config applied after disabling client %s", client.Name)
			}
		}
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
		log.Printf("Error getting clients: %v", err)
		return fmt.Errorf("cannot get clients: %v", err)
	}
	log.Printf("Successfully got clients")

	users, err := db.GetUsers()
	if err != nil {
		log.Printf("Error getting users: %v", err)
		return fmt.Errorf("cannot get users: %v", err)
	}
	log.Printf("Successfully got users")

	settings, err := db.GetGlobalSettings()
	if err != nil {
		log.Printf("Error getting global settings: %v", err)
		return fmt.Errorf("cannot get global settings: %v", err)
	}
	log.Printf("Successfully got global settings")

	// Write config file
	err = util.WriteWireGuardServerConfig(quotaCheckerTmplDir, server, clients, users, settings)
	if err != nil {
		log.Printf("Error writing WireGuard config: %v", err)
		return fmt.Errorf("cannot write WireGuard config: %v", err)
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

	// Check if interface is active
	if !util.IsInterfaceActive(interfaceName) {
		log.Printf("Interface %s is not active, starting it first", interfaceName)
		// Try to start the interface if it's not active
		startCmd := exec.Command("sudo", "wg-quick", "up", interfaceName)
		if output, err := startCmd.CombinedOutput(); err != nil {
			log.Printf("Failed to start interface %s: %v, output: %s", interfaceName, err, string(output))
			return fmt.Errorf("interface %s is not active and cannot be started: %v", interfaceName, err)
		}
	}

	// Apply configuration changes using pure runtime commands with zero disruption
	err = util.ApplyConfigChanges(interfaceName, settings.ConfigFilePath, clients, settings)
	if err != nil {
		log.Printf("Pure runtime configuration failed: %v, falling back to service restart", err)

		// Fallback to service restart only if absolutely necessary
		serviceName := fmt.Sprintf("wg-quick@%s", interfaceName)
		cmd := exec.Command("sudo", "systemctl", "restart", serviceName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Error restarting WireGuard service: %v, Output: %s", err, string(output))
			return fmt.Errorf("error restarting WireGuard service: %v, Output: %s", err, string(output))
		}
		log.Printf("Configuration applied via service restart (fallback method)")

		// Verify service is active after restart
		checkCmd := exec.Command("sudo", "systemctl", "is-active", serviceName)
		status, err := checkCmd.CombinedOutput()
		if err != nil || strings.TrimSpace(string(status)) != "active" {
			log.Printf("WireGuard service is not active after restart. Status: %s", string(status))
			return fmt.Errorf("WireGuard service is not active after restart. Status: %s", string(status))
		}
		log.Printf("WireGuard service is active after restart")
	} else {
		log.Printf("Configuration applied successfully using pure runtime commands - zero disruption to existing connections")
	}

	return nil
}

// getWireGuardUsage returns a map of public keys to their traffic usage [received, sent]
type peerUsage struct {
	Rx            uint64
	Tx            uint64
	LastHandshake time.Time
}

func getWireGuardUsage() (map[string]peerUsage, error) {
	log.Printf("Starting to get WireGuard usage")
	usageMap := make(map[string]peerUsage)

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
			usageMap[peer.PublicKey.String()] = peerUsage{
				Rx:            uint64(peer.ReceiveBytes),
				Tx:            uint64(peer.TransmitBytes),
				LastHandshake: peer.LastHandshakeTime,
			}
		}
	}

	return usageMap, nil
}
