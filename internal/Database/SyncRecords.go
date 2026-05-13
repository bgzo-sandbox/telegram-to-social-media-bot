package Database

import (
	"errors"

	"gorm.io/gorm"

	"telegram-message-sync-bot/internal/Entity"
)

func SaveSyncRecord(record *Entity.SyncRecord) (int64, error) {
	if record.AttemptNo <= 0 {
		attemptNo, err := nextSyncAttemptNo(record.ArchivedMessageID, record.Platform)
		if err != nil {
			return 0, err
		}
		record.AttemptNo = attemptNo
	}

	err := DB.Create(record).Error
	if err != nil {
		return 0, err
	}
	return record.ID, nil
}

func ListSyncRecordsByMessage(archivedMessageID int64) ([]Entity.SyncRecord, error) {
	var records []Entity.SyncRecord
	err := DB.Where("archived_message_id = ?", archivedMessageID).
		Order("platform ASC, attempt_no ASC, id ASC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

func GetLatestSyncRecord(archivedMessageID int64, platform string) (*Entity.SyncRecord, error) {
	var record Entity.SyncRecord
	err := DB.Where("archived_message_id = ? AND platform = ?", archivedMessageID, platform).
		Order("attempt_no DESC, id DESC").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func nextSyncAttemptNo(archivedMessageID int64, platform string) (int, error) {
	var record Entity.SyncRecord
	err := DB.Where("archived_message_id = ? AND platform = ?", archivedMessageID, platform).
		Order("attempt_no DESC, id DESC").
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return record.AttemptNo + 1, nil
}
