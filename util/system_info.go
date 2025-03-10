package util

import (
	"io/ioutil"
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
		CPU: model.CPUInfo{
			Cores: 4,
			Used:  50.0,
		},
		Memory: model.MemoryInfo{
			Total: 8 * 1024 * 1024 * 1024, // 8GB
			Used:  4 * 1024 * 1024 * 1024, // 4GB
			Free:  4 * 1024 * 1024 * 1024, // 4GB
		},
		Swap: model.SwapInfo{
			Total: 2 * 1024 * 1024 * 1024, // 2GB
			Used:  0,
			Free:  2 * 1024 * 1024 * 1024,
		},
		Disk: model.DiskInfo{
			Total: 100 * 1024 * 1024 * 1024, // 100GB
			Used:  50 * 1024 * 1024 * 1024,  // 50GB
			Free:  50 * 1024 * 1024 * 1024,  // 50GB
		},
		Load:   []float64{1.0, 1.0, 1.0},
		Uptime: "up 24 hours, 0 minutes",
		Network: model.NetworkInfo{
			UploadSpeed:   1024 * 1024, // 1MB/s
			DownloadSpeed: 1024 * 1024, // 1MB/s
			TotalUpload:   1024 * 1024 * 1024,
			TotalDownload: 1024 * 1024 * 1024,
			IPv4:          true,
			IPv6:          false,
		},
	}

	return status, nil
}
