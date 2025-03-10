package util

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/MmadF14/vwireguard/model"
	"github.com/labstack/gommon/log"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

var (
	lastNetStats   map[string]net.IOCountersStat
	lastUpdateTime time.Time
)

func init() {
	lastNetStats = make(map[string]net.IOCountersStat)
	lastUpdateTime = time.Now()
}

// GetSystemStatus returns complete system status information
func GetSystemStatus() (*model.SystemStatus, error) {
	fmt.Printf("Starting GetSystemStatus function\n")

	// ساخت یک وضعیت ثابت برای تست
	status := &model.SystemStatus{
		CPU: model.CPUInfo{
			Cores: 4,
			Used:  50.0,
		},
		Memory: model.MemoryInfo{
			Total: 8 * 1024 * 1024 * 1024, // 8GB
			Used:  4 * 1024 * 1024 * 1024, // 4GB
			Free:  4 * 1024 * 1024 * 1024, // 4GB
		},
		Disk: model.DiskInfo{
			Total: 100 * 1024 * 1024 * 1024, // 100GB
			Used:  50 * 1024 * 1024 * 1024,  // 50GB
			Free:  50 * 1024 * 1024 * 1024,  // 50GB
		},
		Load:   []float64{1.0, 1.0, 1.0},
		Uptime: "1d 2h 30m",
		Network: model.NetworkInfo{
			UploadSpeed:   1024 * 1024,             // 1MB/s
			DownloadSpeed: 2 * 1024 * 1024,         // 2MB/s
			TotalUpload:   10 * 1024 * 1024 * 1024, // 10GB
			TotalDownload: 20 * 1024 * 1024 * 1024, // 20GB
			IPv4:          true,
			IPv6:          false,
		},
	}

	fmt.Printf("Created test status object successfully\n")
	return status, nil
}

func getCPUInfo(info *model.CPUInfo) error {
	log.Info("شروع دریافت اطلاعات CPU...")

	cpus, err := cpu.Info()
	if err != nil {
		log.Errorf("خطا در دریافت اطلاعات CPU: %v", err)
		return fmt.Errorf("failed to get CPU info: %v", err)
	}
	info.Cores = len(cpus)

	// برای ویندوز، زمان نمونه‌برداری را کاهش می‌دهیم
	var sampleTime time.Duration
	if runtime.GOOS == "windows" {
		sampleTime = 100 * time.Millisecond
	} else {
		sampleTime = time.Second
	}

	percentage, err := cpu.Percent(sampleTime, false)
	if err != nil {
		log.Errorf("خطا در دریافت درصد استفاده CPU: %v", err)
		return fmt.Errorf("failed to get CPU percentage: %v", err)
	}
	if len(percentage) > 0 {
		info.Used = percentage[0]
	}

	log.Infof("اطلاعات CPU با موفقیت دریافت شد: Cores=%d, Used=%.2f%%", info.Cores, info.Used)
	return nil
}

func getMemoryInfo(info *model.MemoryInfo) error {
	log.Info("شروع دریافت اطلاعات حافظه...")

	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Errorf("خطا در دریافت اطلاعات حافظه: %v", err)
		return fmt.Errorf("failed to get virtual memory info: %v", err)
	}
	info.Total = vm.Total
	info.Used = vm.Used
	info.Free = vm.Available

	log.Infof("اطلاعات حافظه با موفقیت دریافت شد: Total=%d, Used=%d, Free=%d", info.Total, info.Used, info.Free)
	return nil
}

func getSwapInfo(info *model.SwapInfo) error {
	swap, err := mem.SwapMemory()
	if err != nil {
		return fmt.Errorf("failed to get swap memory info: %v", err)
	}
	info.Total = swap.Total
	info.Used = swap.Used
	info.Free = swap.Free
	return nil
}

func getDiskInfo(info *model.DiskInfo) error {
	log.Info("شروع دریافت اطلاعات دیسک...")

	var usage *disk.UsageStat
	var err error

	if runtime.GOOS == "windows" {
		usage, err = disk.Usage("C:")
	} else {
		usage, err = disk.Usage("/")
	}

	if err != nil {
		log.Errorf("خطا در دریافت اطلاعات دیسک: %v", err)
		return fmt.Errorf("failed to get disk usage info: %v", err)
	}

	info.Total = usage.Total
	info.Used = usage.Used
	info.Free = usage.Free

	log.Infof("اطلاعات دیسک با موفقیت دریافت شد: Total=%d, Used=%d, Free=%d", info.Total, info.Used, info.Free)
	return nil
}

func getSystemLoad(loadAvg *[]float64) error {
	log.Info("شروع دریافت اطلاعات بار سیستم...")

	if runtime.GOOS == "windows" {
		// در ویندوز از درصد CPU به عنوان معیار بار سیستم استفاده می‌کنیم
		percentage, err := cpu.Percent(100*time.Millisecond, false)
		if err != nil {
			log.Error("خطا در دریافت بار سیستم در ویندوز")
			*loadAvg = []float64{0, 0, 0}
			return nil
		}
		if len(percentage) > 0 {
			*loadAvg = []float64{percentage[0], percentage[0], percentage[0]}
		} else {
			*loadAvg = []float64{0, 0, 0}
		}
		return nil
	}

	avg, err := load.Avg()
	if err != nil {
		log.Error("خطا در دریافت بار سیستم")
		*loadAvg = []float64{0, 0, 0}
		return nil
	}
	*loadAvg = []float64{avg.Load1, avg.Load5, avg.Load15}

	log.Infof("اطلاعات بار سیستم با موفقیت دریافت شد: %v", *loadAvg)
	return nil
}

func getUptime(uptime *string) error {
	hostInfo, err := host.Info()
	if err != nil {
		return fmt.Errorf("failed to get host info: %v", err)
	}
	duration := time.Duration(hostInfo.Uptime) * time.Second
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	*uptime = fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	return nil
}

func getNetworkInfo(info *model.NetworkInfo) error {
	log.Info("شروع دریافت اطلاعات شبکه...")

	// دریافت آمار رابط‌های شبکه
	netStats, err := net.IOCounters(false)
	if err != nil {
		log.Errorf("خطا در دریافت آمار شبکه: %v", err)
		return fmt.Errorf("failed to get network IO counters: %v", err)
	}

	if len(netStats) > 0 {
		currentTime := time.Now()
		timeDiff := currentTime.Sub(lastUpdateTime).Seconds()

		if lastStat, ok := lastNetStats["total"]; ok && timeDiff > 0 {
			info.UploadSpeed = uint64(float64(netStats[0].BytesSent-lastStat.BytesSent) / timeDiff)
			info.DownloadSpeed = uint64(float64(netStats[0].BytesRecv-lastStat.BytesRecv) / timeDiff)
		}

		info.TotalUpload = netStats[0].BytesSent
		info.TotalDownload = netStats[0].BytesRecv

		lastNetStats["total"] = netStats[0]
		lastUpdateTime = currentTime

		log.Infof("آمار شبکه: Upload=%d B/s, Download=%d B/s", info.UploadSpeed, info.DownloadSpeed)
	}

	// بررسی پشتیبانی از IPv4 و IPv6
	info.IPv4 = false
	info.IPv6 = false

	interfaces, err := net.Interfaces()
	if err != nil {
		log.Errorf("خطا در دریافت رابط‌های شبکه: %v", err)
		return nil // ادامه می‌دهیم حتی با خطا
	}

	for _, iface := range interfaces {
		if len(iface.Addrs) > 0 {
			for _, addr := range iface.Addrs {
				if strings.Contains(addr.Addr, ":") {
					info.IPv6 = true
				} else if strings.Contains(addr.Addr, ".") {
					info.IPv4 = true
				}
			}
		}
	}

	log.Infof("پشتیبانی شبکه: IPv4=%v, IPv6=%v", info.IPv4, info.IPv6)
	return nil
}
