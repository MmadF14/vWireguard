package model

// Usage represents a client's data usage statistics
type Usage struct {
	ClientID    int
	BytesIn     int64
	BytesOut    int64
	LastUpdated string
} 