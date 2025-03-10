package util

import (
	"fmt"
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
	lastNetStats   map[string]uint64
	lastUpdateTime time.Time
)

func init() {
	lastNetStats = make(map[string]uint64)
	lastUpdateTime = time.Now()
}

// GetSystemStatus returns complete system status information
func GetSystemStatus() (*model.SystemStatus, error) {
	status := &model.SystemStatus{
		CPU:     model.CPUInfo{},
		Memory:  model.MemoryInfo{},
		Swap:    model.SwapInfo{},
		Disk:    model.DiskInfo{},
		Network: model.NetworkInfo{},
		Load:    []float64{0, 0, 0},
		Uptime:  "Unknown",
	}

	// CPU Info
	cpuCores, err := cpu.Counts(true)
	if err != nil {
		log.Warnf("Failed to get CPU cores: %v", err)
		status.CPU.Cores = 0
	} else {
		status.CPU.Cores = cpuCores
	}

	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		log.Warnf("Failed to get CPU percent: %v", err)
		status.CPU.Used = 0
	} else if len(cpuPercent) > 0 {
		status.CPU.Used = cpuPercent[0]
	}

	// Memory Info
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Warnf("Failed to get memory info: %v", err)
	} else {
		status.Memory.Total = memInfo.Total
		status.Memory.Used = memInfo.Used
		status.Memory.Free = memInfo.Free
	}

	// Swap Info
	swapInfo, err := mem.SwapMemory()
	if err != nil {
		log.Warnf("Failed to get swap info: %v", err)
	} else {
		status.Swap.Total = swapInfo.Total
		status.Swap.Used = swapInfo.Used
		status.Swap.Free = swapInfo.Free
	}

	// Disk Info
	diskInfo, err := disk.Usage("/")
	if err != nil {
		log.Warnf("Failed to get disk info: %v", err)
	} else {
		status.Disk.Total = diskInfo.Total
		status.Disk.Used = diskInfo.Used
		status.Disk.Free = diskInfo.Free
	}

	// Load Average
	loadInfo, err := load.Avg()
	if err != nil {
		log.Warnf("Failed to get load average: %v", err)
		status.Load = []float64{0, 0, 0}
	} else {
		status.Load = []float64{loadInfo.Load1, loadInfo.Load5, loadInfo.Load15}
	}

	// Uptime
	uptime, err := host.Uptime()
	if err != nil {
		log.Warnf("Failed to get uptime: %v", err)
		status.Uptime = "Unknown"
	} else {
		hours := uptime / 3600
		minutes := (uptime % 3600) / 60
		status.Uptime = fmt.Sprintf("up %d hours, %d minutes", hours, minutes)
	}

	// Network Info
	netStats, err := net.IOCounters(false)
	if err != nil {
		log.Warnf("Failed to get network stats: %v", err)
	} else if len(netStats) > 0 {
		currentTime := time.Now()
		timeDiff := currentTime.Sub(lastUpdateTime).Seconds()

		if timeDiff > 0 && len(lastNetStats) > 0 {
			bytesRecv := netStats[0].BytesRecv
			bytesSent := netStats[0].BytesSent

			status.Network.DownloadSpeed = uint64(float64(bytesRecv-lastNetStats["recv"]) / timeDiff)
			status.Network.UploadSpeed = uint64(float64(bytesSent-lastNetStats["sent"]) / timeDiff)
			status.Network.TotalDownload = bytesRecv
			status.Network.TotalUpload = bytesSent
		}

		lastNetStats["recv"] = netStats[0].BytesRecv
		lastNetStats["sent"] = netStats[0].BytesSent
		lastUpdateTime = currentTime
	}

	// Network Capabilities
	status.Network.IPv4 = true  // Assuming IPv4 is always available
	status.Network.IPv6 = false // Would need more complex detection

	return status, nil
}
