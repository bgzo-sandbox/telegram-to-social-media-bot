package Entity

import "time"

type SyncStatus string

const (
	SyncStatusSucceeded SyncStatus = "succeeded"
	SyncStatusFailed    SyncStatus = "failed"
)

type SyncTrigger string

const (
	SyncTriggerAutomatic SyncTrigger = "automatic"
	SyncTriggerManual    SyncTrigger = "manual"
)

type SyncRecord struct {
	ID int64

	ArchivedMessageID int64      `gorm:"not null;index:idx_sync_record_message_platform_attempt"`
	Platform          string     `gorm:"not null;index:idx_sync_record_message_platform_attempt"`
	Status            SyncStatus `gorm:"not null"`
	RemoteID          string
	RemoteURL         string
	ErrorMessage      string
	Trigger           SyncTrigger `gorm:"not null"`
	ImageRequested    bool
	UsedImage         bool
	AttemptNo         int `gorm:"not null;index:idx_sync_record_message_platform_attempt"`

	CreatedTime time.Time
}
