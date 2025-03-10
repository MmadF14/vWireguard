package util

import (
	"fmt"
	"time"

	"github.com/MmadF14/vwireguard/model"
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
	status := &model.SystemStatus{}
	var err error

	// Get CPU info
	if err = getCPUInfo(&status.CPU); err != nil {
		return nil, fmt.Errorf("error getting CPU info: %v", err)
	}

	// Get memory info
	if err = getMemoryInfo(&status.Memory); err != nil {
		return nil, fmt.Errorf("error getting memory info: %v", err)
	}

	// Get swap info
	if err = getSwapInfo(&status.Swap); err != nil {
		return nil, fmt.Errorf("error getting swap info: %v", err)
	}

	// Get disk info
	if err = getDiskInfo(&status.Disk); err != nil {
		return nil, fmt.Errorf("error getting disk info: %v", err)
	}

	// Get system load
	if err = getSystemLoad(&status.Load); err != nil {
		return nil, fmt.Errorf("error getting system load: %v", err)
	}

	// Get uptime
	if err = getUptime(&status.Uptime); err != nil {
		return nil, fmt.Errorf("error getting uptime: %v", err)
	}

	// Get network info
	if err = getNetworkInfo(&status.Network); err != nil {
		return nil, fmt.Errorf("error getting network info: %v", err)
	}

	return status, nil
}

func getCPUInfo(info *model.CPUInfo) error {
	cpus, err := cpu.Info()
	if err != nil {
		return err
	}
	info.Cores = len(cpus)

	percentage, err := cpu.Percent(0, false)
	if err != nil {
		return err
	}
	if len(percentage) > 0 {
		info.Used = percentage[0]
	}
	return nil
}

func getMemoryInfo(info *model.MemoryInfo) error {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	info.Total = vm.Total
	info.Used = vm.Used
	info.Free = vm.Free
	return nil
}

func getSwapInfo(info *model.SwapInfo) error {
	swap, err := mem.SwapMemory()
	if err != nil {
		return err
	}
	info.Total = swap.Total
	info.Used = swap.Used
	info.Free = swap.Free
	return nil
}

func getDiskInfo(info *model.DiskInfo) error {
	usage, err := disk.Usage("/")
	if err != nil {
		return err
	}
	info.Total = usage.Total
	info.Used = usage.Used
	info.Free = usage.Free
	return nil
}

func getSystemLoad(loadAvg *[]float64) error {
	avg, err := load.Avg()
	if err != nil {
		return err
	}
	*loadAvg = []float64{avg.Load1, avg.Load5, avg.Load15}
	return nil
}

func getUptime(uptime *string) error {
	hostInfo, err := host.Info()
	if err != nil {
		return err
	}
	duration := time.Duration(hostInfo.Uptime) * time.Second
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	*uptime = fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	return nil
}

func getNetworkInfo(info *model.NetworkInfo) error {
	// Get network interfaces statistics
	netStats, err := net.IOCounters(false)
	if err != nil {
		return err
	}

	if len(netStats) > 0 {
		currentTime := time.Now()
		timeDiff := currentTime.Sub(lastUpdateTime).Seconds()

		// Calculate speeds
		if lastStat, ok := lastNetStats["total"]; ok && timeDiff > 0 {
			info.UploadSpeed = uint64(float64(netStats[0].BytesSent-lastStat.BytesSent) / timeDiff)
			info.DownloadSpeed = uint64(float64(netStats[0].BytesRecv-lastStat.BytesRecv) / timeDiff)
		}

		// Update total values
		info.TotalUpload = netStats[0].BytesSent
		info.TotalDownload = netStats[0].BytesRecv

		// Store current values for next calculation
		lastNetStats["total"] = netStats[0]
		lastUpdateTime = currentTime
	}

	// Get network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	// Check for IPv4 and IPv6 support
	info.IPv4 = false
	info.IPv6 = false
	for _, iface := range interfaces {
		for _, addr := range iface.Addrs {
			addrStr := addr.String()
			if addrStr != "" {
				if addrStr[0] == '[' {
					info.IPv6 = true
				} else {
					info.IPv4 = true
				}
			}
		}
	}

	// Note: WireGuard ports will be set by the caller
	return nil
}
