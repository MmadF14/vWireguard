package model

import "time"

// GlobalSetting represents global application settings
type GlobalSetting struct {
	DNSServers      []string  `json:"dns_servers"`
	ConfigFilePath  string    `json:"config_file_path"`
	GitHubLink      string    `json:"github_link"`
	TelegramLink    string    `json:"telegram_link"`
	UpdatedAt       time.Time `json:"updated_at"`
} 