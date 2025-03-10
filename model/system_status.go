package model

// SystemStatus represents the complete system status information
type SystemStatus struct {
	CPU     CPUInfo     `json:"cpu"`
	Memory  MemoryInfo  `json:"memory"`
	Swap    SwapInfo    `json:"swap"`
	Disk    DiskInfo    `json:"disk"`
	Load    []float64   `json:"load"`
	Uptime  string      `json:"uptime"`
	Network NetworkInfo `json:"network"`
}

// CPUInfo represents CPU usage information
type CPUInfo struct {
	Cores int     `json:"cores"`
	Used  float64 `json:"used"`
}

// MemoryInfo represents memory usage information
type MemoryInfo struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

// SwapInfo represents swap usage information
type SwapInfo struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

// DiskInfo represents disk usage information
type DiskInfo struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

// NetworkInfo represents network status information
type NetworkInfo struct {
	UploadSpeed   uint64 `json:"uploadSpeed"`
	DownloadSpeed uint64 `json:"downloadSpeed"`
	TotalUpload   uint64 `json:"totalUpload"`
	TotalDownload uint64 `json:"totalDownload"`
	IPv4          bool   `json:"ipv4"`
	IPv6          bool   `json:"ipv6"`
	TCPPort       int    `json:"tcpPort"`
	UDPPort       int    `json:"udpPort"`
}
