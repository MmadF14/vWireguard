package util

import (
	"bufio"
	"fmt"
	"io/ioutil"
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

func readFileContent(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetSystemStatus returns complete system status information
func GetSystemStatus() (*model.SystemStatus, error) {
	status := &model.SystemStatus{
		CPU:     model.CPUInfo{},
		Memory:  model.MemoryInfo{},
		Swap:    model.SwapInfo{},
		Disk:    model.DiskInfo{},
		Load:    make([]float64, 3),
		Network: model.NetworkInfo{},
	}

	// CPU Info
	if content, err := readFileContent("/proc/cpuinfo"); err == nil {
		cores := 0
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "processor") {
				cores++
			}
		}
		status.CPU.Cores = cores
	}

	// CPU Usage
	if content, err := readFileContent("/proc/stat"); err == nil {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "cpu ") {
				fields := strings.Fields(line)
				if len(fields) >= 8 {
					user, _ := strconv.ParseUint(fields[1], 10, 64)
					nice, _ := strconv.ParseUint(fields[2], 10, 64)
					system, _ := strconv.ParseUint(fields[3], 10, 64)
					idle, _ := strconv.ParseUint(fields[4], 10, 64)
					total := user + nice + system + idle
					if total > 0 {
						status.CPU.Used = float64(user+nice+system) * 100.0 / float64(total)
					}
				}
				break
			}
		}
	}

	// Memory Info
	if content, err := readFileContent("/proc/meminfo"); err == nil {
		var total, free, available uint64
		scanner := bufio.NewScanner(strings.NewReader(content))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					total, _ = strconv.ParseUint(fields[1], 10, 64)
					total *= 1024 // Convert from KB to bytes
				}
			} else if strings.HasPrefix(line, "MemFree:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					free, _ = strconv.ParseUint(fields[1], 10, 64)
					free *= 1024
				}
			} else if strings.HasPrefix(line, "MemAvailable:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					available, _ = strconv.ParseUint(fields[1], 10, 64)
					available *= 1024
				}
			}
		}
		status.Memory.Total = total
		status.Memory.Free = available
		status.Memory.Used = total - available
	}

	// Load Average
	if content, err := readFileContent("/proc/loadavg"); err == nil {
		fields := strings.Fields(content)
		for i := 0; i < 3 && i < len(fields); i++ {
			status.Load[i], _ = strconv.ParseFloat(fields[i], 64)
		}
	}

	// Uptime
	if content, err := readFileContent("/proc/uptime"); err == nil {
		fields := strings.Fields(content)
		if len(fields) > 0 {
			if uptime, err := strconv.ParseFloat(fields[0], 64); err == nil {
				hours := int(uptime / 3600)
				minutes := int((uptime - float64(hours)*3600) / 60)
				status.Uptime = fmt.Sprintf("up %d hours, %d minutes", hours, minutes)
			}
		}
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
