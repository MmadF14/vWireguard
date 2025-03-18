package model

// Quota represents a client's data usage quota
type Quota struct {
	ClientID    int
	TotalQuota  int64
	UsedQuota   int64
	Expiration  string
} 