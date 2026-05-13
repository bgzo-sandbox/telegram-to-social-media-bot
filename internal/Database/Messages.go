package Database

import (
	"strings"
	"telegram-message-sync-bot/internal/Entity"
)

type SourceMessageCount struct {
	SourceID      string
	ArchivedCount int64
}

// 保存消息及附件
func SaveMessage(msg *Entity.Message) (int64, error) {
	err := DB.Create(msg).Error
	if err != nil {
		return 0, err
	}
	return msg.ID, nil
}

func IsDuplicateMessageError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return strings.Contains(errMsg, "UNIQUE constraint failed") || strings.Contains(errMsg, "duplicated key")
}

// ListMessages 返回全部消息（含附件），用于归档补齐与核对。
func ListMessages() ([]Entity.Message, error) {
	var msgs []Entity.Message
	err := DB.Preload("Attachments").Order("id ASC").Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// 按ID查找消息（含附件）
func GetMessageByID(id int64) (*Entity.Message, error) {
	var msg Entity.Message
	err := DB.Preload("Attachments").First(&msg, id).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func CountMessages() (int64, error) {
	var count int64
	err := DB.Model(&Entity.Message{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func CountDistinctSources() (int64, error) {
	var count int64
	err := DB.Model(&Entity.Message{}).Distinct("username").Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func ListSourceMessageCounts() ([]SourceMessageCount, error) {
	var rows []SourceMessageCount
	err := DB.Model(&Entity.Message{}).
		Select("username AS source_id, COUNT(*) AS archived_count").
		Group("username").
		Order("archived_count DESC, source_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func GetMessageBySource(messageID int64, username string) (*Entity.Message, error) {
	var msg Entity.Message
	err := DB.Preload("Attachments").Where("message_id = ? AND username = ?", messageID, username).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func ListMessagesBySourceID(sourceID string) ([]Entity.Message, error) {
	var msgs []Entity.Message
	err := DB.Preload("Attachments").Where("username = ?", sourceID).Order("message_date DESC, id DESC").Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// 按用户查找消息（含附件）
func GetMessagesByUser(userID string, limit int) ([]Entity.Message, error) {
	var msgs []Entity.Message
	err := DB.Preload("Attachments").
		Where("sender_id = ? OR receiver_id = ?", userID, userID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// 更新消息内容
func UpdateMessage(msg *Entity.Message) error {
	return DB.Save(msg).Error
}

// 删除消息及附件
func DeleteMessage(id int64) error {
	// 先删除附件
	DB.Where("message_id = ?", id).Delete(&Entity.Attachment{})
	return DB.Delete(&Entity.Message{}, id).Error
}
