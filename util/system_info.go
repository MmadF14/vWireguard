package util

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/MmadF14/vwireguard/model"
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
	status := &model.SystemStatus{}

	// CPU Info
	if out, err := exec.Command("sh", "-c", "grep -c processor /proc/cpuinfo").Output(); err == nil {
		if cores, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			status.CPU.Cores = cores
		}
	}
	if out, err := exec.Command("sh", "-c", "top -bn1 | grep 'Cpu(s)' | awk '{print $2}'").Output(); err == nil {
		if used, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil {
			status.CPU.Used = used
		}
	}

	// Memory Info
	if out, err := exec.Command("sh", "-c", "free -b | grep Mem | awk '{print $2,$3,$4}'").Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) == 3 {
			if total, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
				status.Memory.Total = total
			}
			if used, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				status.Memory.Used = used
			}
			if free, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
				status.Memory.Free = free
			}
		}
	}

	// Swap Info
	if out, err := exec.Command("sh", "-c", "free -b | grep Swap | awk '{print $2,$3,$4}'").Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) == 3 {
			if total, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
				status.Swap.Total = total
			}
			if used, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				status.Swap.Used = used
			}
			if free, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
				status.Swap.Free = free
			}
		}
	}

	// Disk Info
	if out, err := exec.Command("sh", "-c", "df -B1 / | tail -1 | awk '{print $2,$3,$4}'").Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) == 3 {
			if total, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
				status.Disk.Total = total
			}
			if used, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				status.Disk.Used = used
			}
			if free, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
				status.Disk.Free = free
			}
		}
	}

	// System Load
	if out, err := exec.Command("sh", "-c", "cat /proc/loadavg | awk '{print $1,$2,$3}'").Output(); err == nil {
		fields := strings.Fields(string(out))
		status.Load = make([]float64, 3)
		for i := 0; i < 3 && i < len(fields); i++ {
			if val, err := strconv.ParseFloat(fields[i], 64); err == nil {
				status.Load[i] = val
			}
		}
	}

	// Uptime
	if out, err := exec.Command("sh", "-c", "uptime -p").Output(); err == nil {
		status.Uptime = strings.TrimSpace(string(out))
	} else {
		status.Uptime = "Unknown"
	}

	// Network Info
	status.Network = model.NetworkInfo{
		UploadSpeed:   0,
		DownloadSpeed: 0,
		TotalUpload:   0,
		TotalDownload: 0,
		IPv4:          true, // Default values
		IPv6:          false,
	}

	return status, nil
}
