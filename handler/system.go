package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	"github.com/MmadF14/vwireguard/model"
	"github.com/MmadF14/vwireguard/store"
	"github.com/MmadF14/vwireguard/util"
	"github.com/MmadF14/vwireguard/zip"
)

type SystemMetrics struct {
	CPU struct {
		Usage float64 `json:"usage"`
		Cores int     `json:"cores"`
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
		TotalOut      uint64  `json:"totalOut"`
		TotalIn       uint64  `json:"totalIn"`
	} `json:"network"`

	SystemLoad string `json:"systemLoad"`
	Uptime     string `json:"uptime"`
}

var (
	lastNetStats     []net.IOCountersStat
	lastNetStatsTime time.Time
)

func GetSystemMetrics() echo.HandlerFunc {
	return func(c echo.Context) error {
		metrics := SystemMetrics{}

		cpuPercent, err := cpu.Percent(0, false)
		if err == nil && len(cpuPercent) > 0 {
			metrics.CPU.Usage = cpuPercent[0]
		}
		metrics.CPU.Cores = runtime.NumCPU()

		if vmstat, err := mem.VirtualMemory(); err == nil {
			metrics.RAM.Total = vmstat.Total
			metrics.RAM.Used = vmstat.Used
			metrics.RAM.Usage = vmstat.UsedPercent
		}

		if swapstat, err := mem.SwapMemory(); err == nil {
			metrics.Swap.Total = swapstat.Total
			metrics.Swap.Used = swapstat.Used
			metrics.Swap.Usage = swapstat.UsedPercent
		}

		if diskstat, err := disk.Usage("/"); err == nil {
			metrics.Disk.Total = diskstat.Total
			metrics.Disk.Used = diskstat.Used
			metrics.Disk.Usage = diskstat.UsedPercent
		}

		if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
			currentTime := time.Now()
			if !lastNetStatsTime.IsZero() && len(lastNetStats) > 0 {
				timeDiff := currentTime.Sub(lastNetStatsTime).Seconds()
				bytesSentDiff := float64(netStats[0].BytesSent - lastNetStats[0].BytesSent)
				bytesRecvDiff := float64(netStats[0].BytesRecv - lastNetStats[0].BytesRecv)

				if timeDiff > 0 {
					metrics.Network.UploadSpeed = bytesSentDiff / timeDiff / 1024
					metrics.Network.DownloadSpeed = bytesRecvDiff / timeDiff / 1024
				}
			}
			metrics.Network.TotalOut = netStats[0].BytesSent / 1024 / 1024
			metrics.Network.TotalIn = netStats[0].BytesRecv / 1024 / 1024

			lastNetStats = netStats
			lastNetStatsTime = currentTime
		}

		if loadavg, err := load.Avg(); err == nil {
			metrics.SystemLoad = fmt.Sprintf("%.2f | %.2f | %.2f",
				loadavg.Load1, loadavg.Load5, loadavg.Load15)
		}

		if uptimeSec, err := host.Uptime(); err == nil {
			hours := uptimeSec / 3600
			minutes := (uptimeSec % 3600) / 60
			metrics.Uptime = fmt.Sprintf("%dh %dm", hours, minutes)
		}

		return c.JSON(http.StatusOK, metrics)
	}
}

func SystemMonitorPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		data := map[string]interface{}{
			"baseData": model.BaseData{
				Active:      "system-monitor",
				CurrentUser: currentUser(c),
				Admin:       isAdmin(c),
				BasePath:    util.BasePath,
			},
		}

		if err := c.Render(http.StatusOK, "system_monitor.html", data); err != nil {
			c.Logger().Error("Error rendering system_monitor.html:", err)
			return c.String(http.StatusInternalServerError, "Error rendering system_monitor.html")
		}
		return nil
	}
}

func BackupSystem() echo.HandlerFunc {
	return func(c echo.Context) error {
		tempDir, err := os.MkdirTemp("", "wg-backup-*")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error creating temporary directory: " + err.Error(),
			})
		}
		defer os.RemoveAll(tempDir)

		zipPath := filepath.Join(tempDir, "backup.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error creating zip file: " + err.Error(),
			})
		}
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		wgConfig, err := os.ReadFile("/etc/wireguard/wg0.conf")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error reading WireGuard configuration: " + err.Error(),
			})
		}

		wgWriter, err := zipWriter.Create("wg0.conf")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error adding configuration file to zip: " + err.Error(),
			})
		}
		if _, err := wgWriter.Write(wgConfig); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error writing configuration file: " + err.Error(),
			})
		}

		dbPath := "./db"
		err = filepath.Walk(dbPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(dbPath, path)
			if err != nil {
				return err
			}

			zipEntry, err := zipWriter.Create(filepath.Join("db", relPath))
			if err != nil {
				return err
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			_, err = zipEntry.Write(content)
			return err
		})

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error adding database files to zip: " + err.Error(),
			})
		}

		zipWriter.Close()

		zipContent, err := os.ReadFile(zipPath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error reading zip file: " + err.Error(),
			})
		}

		c.Response().Header().Set("Content-Type", "application/zip")
		c.Response().Header().Set("Content-Disposition", "attachment; filename=wireguard-backup.zip")

		return c.Blob(http.StatusOK, "application/zip", zipContent)
	}
}

func RestoreSystem(db store.IStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		file, err := c.FormFile("backup")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Backup file not found: " + err.Error(),
			})
		}

		tempDir, err := os.MkdirTemp("", "wg-restore-*")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error creating temporary directory: " + err.Error(),
			})
		}
		defer os.RemoveAll(tempDir)

		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error opening backup file: " + err.Error(),
			})
		}
		defer src.Close()

		zipPath := filepath.Join(tempDir, "backup.zip")
		dst, err := os.Create(zipPath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error saving backup file: " + err.Error(),
			})
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error copying backup file: " + err.Error(),
			})
		}

		reader, err := zip.OpenReader(zipPath)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Error opening zip file: " + err.Error(),
			})
		}
		defer reader.Close()

		hasWgConfig := false
		hasDbFiles := false
		for _, file := range reader.File {
			if file.Name == "wg0.conf" {
				hasWgConfig = true
			}
			if strings.HasPrefix(file.Name, "db/") {
				hasDbFiles = true
			}
		}

		if !hasWgConfig || !hasDbFiles {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid backup file",
			})
		}

		tempDbDir := filepath.Join(tempDir, "db")
		if err := os.MkdirAll(tempDbDir, 0755); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error creating temporary database directory: " + err.Error(),
			})
		}

		for _, file := range reader.File {
			rc, err := file.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Error opening file from zip: " + err.Error(),
				})
			}

			if file.Name == "wg0.conf" {
				wgConfig, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Error reading configuration file: " + err.Error(),
					})
				}

				if err := os.WriteFile("/etc/wireguard/wg0.conf", wgConfig, 0600); err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Error writing configuration file: " + err.Error(),
					})
				}
			} else if strings.HasPrefix(file.Name, "db/") {
				targetPath := filepath.Join(tempDbDir, filepath.Base(file.Name))
				outFile, err := os.Create(targetPath)
				if err != nil {
					rc.Close()
					return c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Error creating database file: " + err.Error(),
					})
				}

				if _, err := io.Copy(outFile, rc); err != nil {
					rc.Close()
					outFile.Close()
					return c.JSON(http.StatusInternalServerError, map[string]string{
						"error": "Error writing database file: " + err.Error(),
					})
				}
				outFile.Close()
			}
			rc.Close()
		}

		if err := os.RemoveAll("./db"); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error removing old database: " + err.Error(),
			})
		}

		if err := os.Rename(tempDbDir, "./db"); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error replacing database: " + err.Error(),
			})
		}

		if err := exec.Command("wg-quick", "down", "wg0").Run(); err != nil {
			c.Logger().Error("Error bringing down WireGuard interface:", err)
		}
		if err := exec.Command("wg-quick", "up", "wg0").Run(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Error restarting WireGuard: " + err.Error(),
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Restore completed successfully",
		})
	}
}
