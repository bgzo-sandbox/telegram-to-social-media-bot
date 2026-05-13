package adminqueryservice

import (
	"sort"
	"telegram-message-sync-bot/internal/Database"
	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/internal/service/syncservice"
	"time"

	"gorm.io/gorm"
)

type Overview struct {
	MessageCount          int64
	SourceCount           int64
	SyncTargetSourceCount int
}

type SourceSummary struct {
	SourceID      string
	ArchivedCount int64
	SyncTarget    bool
}

type MessageSummary struct {
	ArchivedMessageID int64
	SourceID          string
	TelegramMessageID int64
	SourceLink        string
	ArchivedAt        time.Time
	LatestStatuses    []PlatformStatus
	HasSyncFailure    bool
}

type PlatformStatus struct {
	Platform       string
	Status         string
	RemoteID       string
	RemoteURL      string
	ErrorMessage   string
	Trigger        string
	ImageRequested bool
	UsedImage      bool
	AttemptNo      int
	UpdatedAt      time.Time
}

type MessageDetail struct {
	ArchivedMessageID int64
	SourceID          string
	TelegramMessageID int64
	SourceLink        string
	Content           string
	ArchivedAt        time.Time
	LatestStatuses    []PlatformStatus
}

func LoadOverview(config Entity.Config) (Overview, error) {
	messageCount, err := Database.CountMessages()
	if err != nil {
		return Overview{}, err
	}

	sourceCount, err := Database.CountDistinctSources()
	if err != nil {
		return Overview{}, err
	}

	return Overview{
		MessageCount:          messageCount,
		SourceCount:           sourceCount,
		SyncTargetSourceCount: len(config.SocialMediaSync.TargetChannel),
	}, nil
}

func ListSources(config Entity.Config, syncTargetsOnly bool) ([]SourceSummary, error) {
	rows, err := Database.ListSourceMessageCounts()
	if err != nil {
		return nil, err
	}

	result := make([]SourceSummary, 0, len(rows))
	for _, row := range rows {
		isSyncTarget := syncservice.ContainsExactTarget(config.SocialMediaSync.TargetChannel, row.SourceID)
		if syncTargetsOnly && !isSyncTarget {
			continue
		}
		result = append(result, SourceSummary{
			SourceID:      row.SourceID,
			ArchivedCount: row.ArchivedCount,
			SyncTarget:    isSyncTarget,
		})
	}
	return result, nil
}

func ListMessagesBySource(sourceID string) ([]MessageSummary, error) {
	msgs, err := Database.ListMessagesBySourceID(sourceID)
	if err != nil {
		return nil, err
	}

	return buildMessageSummaries(msgs)
}

func ListFailedSyncMessages(config Entity.Config) ([]MessageSummary, error) {
	msgs, err := Database.ListMessages()
	if err != nil {
		return nil, err
	}

	filtered := make([]Entity.Message, 0)
	for _, msg := range msgs {
		if !syncservice.ContainsExactTarget(config.SocialMediaSync.TargetChannel, msg.Username) {
			continue
		}

		statuses, err := latestStatusesByMessage(msg.ID)
		if err != nil {
			return nil, err
		}
		if hasFailedStatus(statuses) {
			filtered = append(filtered, msg)
		}
	}

	return buildMessageSummaries(filtered)
}

func GetMessageDetail(archivedMessageID int64) (MessageDetail, error) {
	msg, err := Database.GetMessageByID(archivedMessageID)
	if err != nil {
		return MessageDetail{}, err
	}

	statuses, err := latestStatusesByMessage(msg.ID)
	if err != nil {
		return MessageDetail{}, err
	}

	return MessageDetail{
		ArchivedMessageID: msg.ID,
		SourceID:          msg.Username,
		TelegramMessageID: msg.MessageID,
		SourceLink:        msg.MessageUrl,
		Content:           msg.Content,
		ArchivedAt:        msg.CreatedTime,
		LatestStatuses:    statuses,
	}, nil
}

func buildMessageSummaries(msgs []Entity.Message) ([]MessageSummary, error) {
	result := make([]MessageSummary, 0, len(msgs))
	for _, msg := range msgs {
		statuses, err := latestStatusesByMessage(msg.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, MessageSummary{
			ArchivedMessageID: msg.ID,
			SourceID:          msg.Username,
			TelegramMessageID: msg.MessageID,
			SourceLink:        msg.MessageUrl,
			ArchivedAt:        msg.CreatedTime,
			LatestStatuses:    statuses,
			HasSyncFailure:    hasFailedStatus(statuses),
		})
	}
	return result, nil
}

func latestStatusesByMessage(archivedMessageID int64) ([]PlatformStatus, error) {
	records, err := Database.ListSyncRecordsByMessage(archivedMessageID)
	if err != nil {
		return nil, err
	}

	byPlatform := make(map[string]Entity.SyncRecord)
	for _, record := range records {
		current, ok := byPlatform[record.Platform]
		if !ok || record.AttemptNo > current.AttemptNo || (record.AttemptNo == current.AttemptNo && record.ID > current.ID) {
			byPlatform[record.Platform] = record
		}
	}

	statuses := make([]PlatformStatus, 0, len(byPlatform))
	for _, record := range byPlatform {
		statuses = append(statuses, PlatformStatus{
			Platform:       record.Platform,
			Status:         string(record.Status),
			RemoteID:       record.RemoteID,
			RemoteURL:      record.RemoteURL,
			ErrorMessage:   record.ErrorMessage,
			Trigger:        string(record.Trigger),
			ImageRequested: record.ImageRequested,
			UsedImage:      record.UsedImage,
			AttemptNo:      record.AttemptNo,
			UpdatedAt:      record.CreatedTime,
		})
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Platform < statuses[j].Platform
	})

	return statuses, nil
}

func hasFailedStatus(statuses []PlatformStatus) bool {
	for _, status := range statuses {
		if status.Status == string(Entity.SyncStatusFailed) {
			return true
		}
	}
	return false
}

func IsNotFound(err error) bool {
	return err != nil && err == gorm.ErrRecordNotFound
}
