package logger

import (
	"fmt"
	"log"
	"os"

	"github.com/coreos/go-systemd/v22/journal"
)

// Logger provides structured logging with systemd journal integration
type Logger struct {
	foreground bool
}

// New creates a new logger instance
func New(foreground bool) *Logger {
	return &Logger{
		foreground: foreground,
	}
}

// Info logs an informational message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(journal.PriInfo, "INFO", msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(journal.PriWarning, "WARN", msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(journal.PriErr, "ERROR", msg, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(journal.PriDebug, "DEBUG", msg, args...)
}

// Audit logs an audit message with structured fields
func (l *Logger) Audit(msg string, fields map[string]string) {
	if l.foreground {
		// In foreground mode, log to stderr with structured format
		fieldStr := ""
		for k, v := range fields {
			fieldStr += fmt.Sprintf(" %s=%s", k, v)
		}
		log.Printf("[AUDIT] %s%s", msg, fieldStr)
	} else {
		// In daemon mode, use systemd journal with structured fields
		journalFields := make(map[string]string)
		journalFields["MESSAGE"] = msg
		journalFields["PRIORITY"] = fmt.Sprintf("%d", journal.PriInfo)
		journalFields["SYSLOG_IDENTIFIER"] = "host-manager"
		journalFields["LOG_TYPE"] = "AUDIT"

		// Add custom fields
		for k, v := range fields {
			journalFields[k] = v
		}

		if err := journal.Send(msg, journal.PriInfo, journalFields); err != nil {
			// Fallback to stderr if journal fails
			log.Printf("[AUDIT] %s (journal error: %v)", msg, err)
		}
	}
}

// log handles the actual logging logic
func (l *Logger) log(priority journal.Priority, level, msg string, args ...interface{}) {
	// Build structured message
	message := msg
	fields := make(map[string]string)

	// Parse key-value pairs from args
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key := fmt.Sprintf("%v", args[i])
			value := fmt.Sprintf("%v", args[i+1])
			fields[key] = value
			message += fmt.Sprintf(" %s=%s", key, value)
		}
	}

	if l.foreground {
		// In foreground mode, log to stderr
		log.Printf("[%s] %s", level, message)
	} else {
		// In daemon mode, use systemd journal
		journalFields := make(map[string]string)
		journalFields["SYSLOG_IDENTIFIER"] = "host-manager"

		// Add custom fields
		for k, v := range fields {
			journalFields[k] = v
		}

		if err := journal.Send(msg, priority, journalFields); err != nil {
			// Fallback to stderr if journal fails
			log.Printf("[%s] %s (journal error: %v)", level, msg, err)
		}
	}
}

// IsJournalAvailable checks if systemd journal is available
func IsJournalAvailable() bool {
	return os.Getenv("JOURNAL_STREAM") != "" || journal.Enabled()
}