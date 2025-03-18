package model

// Telegram represents Telegram bot settings
type Telegram struct {
	Token            string
	AllowConfRequest bool
	FloodWait        int
} 