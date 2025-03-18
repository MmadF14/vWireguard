package model

// Email represents email settings
type Email struct {
	FromAddress string
	FromName    string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	SMTPAuth    string
	SMTPEncrypt string
	SMTPHelo    string
} 