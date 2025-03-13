package handler

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	// توجه کنید مسیر زیر را مطابق با مسیر درست پکیج model در پروژه‌تان تغییر دهید:
	"github.com/MmadF14/vwireguard/model"
)

// ساختار پاسخ JSON برای ارسال اطلاعات سیستمی به فرانت‌اند
type SystemMetrics struct {
	CPU struct {
		Usage float64 `json:"usage"` // درصد مصرف CPU
		Cores int     `json:"cores"` // تعداد هسته‌های CPU
	} `json:"cpu"`

	RAM struct {
		Total uint64  `json:"total"` // کل حافظه RAM (بایت)
		Used  uint64  `json:"used"`  // میزان استفاده شده (بایت)
		Usage float64 `json:"usage"` // درصد مصرف RAM
	} `json:"ram"`

	Swap struct {
		Total uint64  `json:"total"`
		Used  uint64  `json:"used"`
		Usage float64 `json:"usage"`
	} `json:"swap"`

	Disk struct {
		Total uint64  `json:"total"`
		Used  uint64  `json:"used"`
		Usage float64 `json:"usage"`
	} `json:"disk"`

	Network struct {
		UploadSpeed   float64 `json:"uploadSpeed"`   // سرعت آپلود بر حسب KB/s
		DownloadSpeed float64 `json:"downloadSpeed"` // سرعت دانلود بر حسب KB/s
		TotalOut      uint64  `json:"totalOut"`      // حجم کل ارسال‌شده (MB)
		TotalIn       uint64  `json:"totalIn"`       // حجم کل دریافت‌شده (MB)
	} `json:"network"`

	SystemLoad string `json:"systemLoad"` // مثلاً "0.12 | 0.25 | 0.30"
	Uptime     string `json:"uptime"`     // مثلاً "5h 32m"
}

// برای نگهداری آمار شبکه در فراخوانی‌های متوالی
var (
	lastNetStats     []net.IOCountersStat
	lastNetStatsTime time.Time
)

// GetSystemMetrics یک هندلر Echo است که اطلاعات سیستمی را در قالب JSON برمی‌گرداند
func GetSystemMetrics() echo.HandlerFunc {
	return func(c echo.Context) error {
		metrics := SystemMetrics{}

		// CPU
		cpuPercent, err := cpu.Percent(0, false)
		if err == nil && len(cpuPercent) > 0 {
			metrics.CPU.Usage = cpuPercent[0]
		}
		metrics.CPU.Cores = runtime.NumCPU()

		// RAM
		if vmstat, err := mem.VirtualMemory(); err == nil {
			metrics.RAM.Total = vmstat.Total
			metrics.RAM.Used = vmstat.Used
			metrics.RAM.Usage = vmstat.UsedPercent
		}

		// Swap
		if swapstat, err := mem.SwapMemory(); err == nil {
			metrics.Swap.Total = swapstat.Total
			metrics.Swap.Used = swapstat.Used
			metrics.Swap.Usage = swapstat.UsedPercent
		}

		// Disk
		if diskstat, err := disk.Usage("/"); err == nil {
			metrics.Disk.Total = diskstat.Total
			metrics.Disk.Used = diskstat.Used
			metrics.Disk.Usage = diskstat.UsedPercent
		}

		// Network
		if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
			currentTime := time.Now()
			// اگر قبلاً اطلاعاتی ذخیره شده باشد، سرعت آپلود/دانلود را حساب می‌کنیم
			if !lastNetStatsTime.IsZero() && len(lastNetStats) > 0 {
				timeDiff := currentTime.Sub(lastNetStatsTime).Seconds()
				bytesSentDiff := float64(netStats[0].BytesSent - lastNetStats[0].BytesSent)
				bytesRecvDiff := float64(netStats[0].BytesRecv - lastNetStats[0].BytesRecv)

				if timeDiff > 0 {
					// تبدیل بایت بر ثانیه به کیلوبایت بر ثانیه
					metrics.Network.UploadSpeed = bytesSentDiff / timeDiff / 1024
					metrics.Network.DownloadSpeed = bytesRecvDiff / timeDiff / 1024
				}
			}
			// تبدیل بایت به مگابایت
			metrics.Network.TotalOut = netStats[0].BytesSent / 1024 / 1024
			metrics.Network.TotalIn = netStats[0].BytesRecv / 1024 / 1024

			lastNetStats = netStats
			lastNetStatsTime = currentTime
		}

		// System Load
		if loadavg, err := load.Avg(); err == nil {
			metrics.SystemLoad = fmt.Sprintf("%.2f | %.2f | %.2f",
				loadavg.Load1, loadavg.Load5, loadavg.Load15)
		}

		// Uptime
		if uptimeSec, err := host.Uptime(); err == nil {
			hours := uptimeSec / 3600
			minutes := (uptimeSec % 3600) / 60
			metrics.Uptime = fmt.Sprintf("%dh %dm", hours, minutes)
		}

		return c.JSON(http.StatusOK, metrics)
	}
}

// SystemMonitorPage یک هندلر برای رندر صفحهٔ مانیتورینگ است
func SystemMonitorPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		// توجه: اگر از BaseData استفاده می‌کنید، ساختار دلخواهتان را پر کنید
		return c.Render(http.StatusOK, "system_monitor.html", map[string]interface{}{
			"baseData": model.BaseData{
				Active:      "system-monitor",
				CurrentUser: currentUser(c), // بسته به پیاده‌سازی‌تان
				Admin:       isAdmin(c),     // بسته به پیاده‌سازی‌تان
			},
		})
	}
}
