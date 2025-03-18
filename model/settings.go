package model

type GlobalSetting struct {
    ConfigFilePath string   `json:"config_file_path"`
    DNSServers    []string `json:"dns_servers"`
    GitHubLink    string   `json:"github_link"`
    TelegramLink  string   `json:"telegram_link"`
    UpdatedAt     int64    `json:"updated_at"`
} 