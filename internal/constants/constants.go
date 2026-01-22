package constants

import "time"

// Time format strings
const (
	// DateTimeFormat is the standard datetime format used throughout the application
	DateTimeFormat = "2006-01-02 15:04:05"

	// DateFormat is the date-only format
	DateFormat = "2006-01-02"

	// TimestampFormat is used for generating filenames
	TimestampFormat = "20060102-150405"
)

// Display limits for CLI output
const (
	// MaxFilesToShowByDefault is the maximum number of files to show in listings
	MaxFilesToShowByDefault = 20

	// MaxConflictsToShow is the maximum number of conflict warnings to show
	MaxConflictsToShow = 10

	// MaxPatternsToShow is the maximum patterns to show before truncating
	MaxPatternsToShow = 5

	// MaxNameLength is the maximum file name length before truncating in display
	MaxNameLength = 20

	// MaxNameLengthExtended is the extended max name length for verbose output
	MaxNameLengthExtended = 50
)

// File size thresholds
const (
	// SmallFileThreshold is the size below which a file is considered "small" (1KB)
	SmallFileThreshold = 1024

	// MediumFileThreshold is the size below which a file is considered "medium" (10KB)
	MediumFileThreshold = 10 * 1024
)

// Time thresholds
const (
	// RecentFileAge is the threshold for considering a file "recent" (30 days)
	RecentFileAge = 30 * 24 * time.Hour
)

// Archive file extension
const (
	// ArchiveExtension is the file extension for encrypted archives
	ArchiveExtension = ".enc"
)
