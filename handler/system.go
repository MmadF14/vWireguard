package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type SystemMetrics struct {
	CPU struct {
		Usage  float64 `json:"usage"`
		Cores  int     `json:"cores"`
	} `json:"cpu"`
	RAM struct {
		Total uint64  `json:"total"`
		Used  uint64  `json:"used"`
		Usage float64 `json:"usage"`
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
		UploadSpeed   float64 `json:"uploadSpeed"`
		DownloadSpeed float64 `json:"downloadSpeed"`
		TotalOut     uint64  `json:"totalOut"`
		TotalIn      uint64  `json:"totalIn"`
	} `json:"network"`
	SystemLoad string    `json:"systemLoad"`
	Uptime     string    `json:"uptime"`
}

var (
	lastNetStats    []net.IOCountersStat
	lastNetStatsTime time.Time
)

// GetSystemMetrics returns current system metrics
func GetSystemMetrics() echo.HandlerFunc {
	return func(c echo.Context) error {
		metrics := SystemMetrics{}

		// CPU
		cpuPercent, err := cpu.Percent(0, false)
		if err == nil && len(cpuPercent) > 0 {
			metrics.CPU.Usage = cpuPercent[0]
		}
		metrics.CPU.Cores = runtime.NumCPU()

		// Memory
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
			if !lastNetStatsTime.IsZero() {
				timeDiff := currentTime.Sub(lastNetStatsTime).Seconds()
				bytesSentDiff := float64(netStats[0].BytesSent - lastNetStats[0].BytesSent)
				bytesRecvDiff := float64(netStats[0].BytesRecv - lastNetStats[0].BytesRecv)

				metrics.Network.UploadSpeed = bytesSentDiff / timeDiff / 1024 // KB/s
				metrics.Network.DownloadSpeed = bytesRecvDiff / timeDiff / 1024 // KB/s
			}
			metrics.Network.TotalOut = netStats[0].BytesSent / 1024 / 1024 // MB
			metrics.Network.TotalIn = netStats[0].BytesRecv / 1024 / 1024 / 1024 // GB
			lastNetStats = netStats
			lastNetStatsTime = currentTime
		}

		// System Load
		if loadavg, err := host.LoadAvg(); err == nil {
			metrics.SystemLoad = fmt.Sprintf("%.2f | %.2f | %.2f", loadavg.Load1, loadavg.Load5, loadavg.Load15)
		}

		// Uptime
		if uptime, err := host.Uptime(); err == nil {
			hours := uptime / 3600
			minutes := (uptime % 3600) / 60
			metrics.Uptime = fmt.Sprintf("%dh %dm", hours, minutes)
		}

		return c.JSON(http.StatusOK, metrics)
	}
}

// SystemMonitorPage handler for system monitoring page
func SystemMonitorPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "system_monitor.html", map[string]interface{}{
			"baseData": model.BaseData{
				Active: "system-monitor",
				CurrentUser: currentUser(c),
				Admin: isAdmin(c),
			},
		})
	}
} 