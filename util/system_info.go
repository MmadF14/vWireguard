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
	if out, err := exec.Command("/usr/bin/grep", "-c", "processor", "/proc/cpuinfo").Output(); err == nil {
		if cores, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			status.CPU.Cores = cores
		}
	}
	if out, err := exec.Command("/usr/bin/top", "-bn1").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "Cpu(s)") {
				fields := strings.Fields(line)
				for i, field := range fields {
					if field == "id," && i > 0 {
						if idle, err := strconv.ParseFloat(fields[i-1], 64); err == nil {
							status.CPU.Used = 100.0 - idle
							break
						}
					}
				}
				break
			}
		}
	}

	// Memory Info
	if out, err := exec.Command("/usr/bin/free", "-b").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Mem:") {
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					if total, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						status.Memory.Total = total
					}
					if used, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
						status.Memory.Used = used
					}
					if free, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
						status.Memory.Free = free
					}
				}
				break
			}
		}
	}

	// Swap Info
	if out, err := exec.Command("/usr/bin/free", "-b").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Swap:") {
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					if total, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						status.Swap.Total = total
					}
					if used, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
						status.Swap.Used = used
					}
					if free, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
						status.Swap.Free = free
					}
				}
				break
			}
		}
	}

	// Disk Info
	if out, err := exec.Command("/usr/bin/df", "-B1", "/").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				if total, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					status.Disk.Total = total
				}
				if used, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
					status.Disk.Used = used
				}
				if free, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
					status.Disk.Free = free
				}
			}
		}
	}

	// System Load
	if out, err := exec.Command("/usr/bin/cat", "/proc/loadavg").Output(); err == nil {
		fields := strings.Fields(string(out))
		status.Load = make([]float64, 3)
		for i := 0; i < 3 && i < len(fields); i++ {
			if val, err := strconv.ParseFloat(fields[i], 64); err == nil {
				status.Load[i] = val
			}
		}
	}

	// Uptime
	if out, err := exec.Command("/usr/bin/uptime", "-p").Output(); err == nil {
		status.Uptime = strings.TrimSpace(string(out))
	} else {
		status.Uptime = "Unknown"
	}

	// Network Info (simplified)
	status.Network = model.NetworkInfo{
		UploadSpeed:   0,
		DownloadSpeed: 0,
		TotalUpload:   0,
		TotalDownload: 0,
		IPv4:          true,
		IPv6:          false,
	}

	return status, nil
}
