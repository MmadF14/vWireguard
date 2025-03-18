package model

// Metrics represents system metrics
type Metrics struct {
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
	NetworkIn   int64
	NetworkOut  int64
	Uptime      string
} 