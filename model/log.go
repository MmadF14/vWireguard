package model

// Log represents a system log entry
type Log struct {
	Level   string
	Message string
	Time    string
}

// LogLevel represents the level of a log entry
type LogLevel string

const (
	LogLevelError   LogLevel = "error"
	LogLevelWarning LogLevel = "warning"
	LogLevelInfo    LogLevel = "info"
	LogLevelDebug   LogLevel = "debug"
) 