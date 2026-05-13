package Handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"telegram-message-sync-bot/internal/Entity"
	"telegram-message-sync-bot/internal/service/adminqueryservice"
	"telegram-message-sync-bot/internal/service/syncservice"
)

const adminPageSize = 5

func AdminHome(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config) {
	overview, err := adminqueryservice.LoadOverview(config)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载管理首页失败: %v", err))
		return
	}

	respondAdminPage(ctx, b, update, renderOverview(overview), buildHomeKeyboard())
}

func AdminCallback(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config) {
	if update == nil || update.CallbackQuery == nil {
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	page, err := parseAdminCallback(update.CallbackQuery.Data)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("无效操作: %v", err))
		return
	}

	switch page.Kind {
	case "home":
		AdminHome(ctx, b, update, config)
	case "sources":
		renderSourcesPage(ctx, b, update, config, page.Scope, page.Offset)
	case "source":
		renderSourceMessagesPage(ctx, b, update, config, page.Scope, page.SourceIndex, page.Offset)
	case "failed":
		renderFailedMessagesPage(ctx, b, update, config, page.Offset)
	case "detail-source":
		renderMessageDetailFromSource(ctx, b, update, config, page.Scope, page.SourceIndex, page.Offset, page.ItemIndex)
	case "detail-failed":
		renderMessageDetailFromFailed(ctx, b, update, config, page.Offset, page.ItemIndex)
	case "resync-entry-source":
		renderResyncEntryFromSource(ctx, b, update, config, page.Scope, page.SourceIndex, page.Offset, page.ItemIndex)
	case "resync-entry-failed":
		renderResyncEntryFromFailed(ctx, b, update, config, page.Offset, page.ItemIndex)
	case "resync-entry-id":
		renderResyncEntryByID(ctx, b, update, page.ArchivedMessageID)
	case "detail-id":
		renderMessageDetailByID(ctx, b, update, page.ArchivedMessageID)
	case "resync-run":
		renderManualResyncResult(ctx, b, update, config, page.ArchivedMessageID, page.Platform)
	default:
		respondAdminError(ctx, b, update, "暂不支持的管理操作")
	}
}

type adminPage struct {
	Kind              string
	Scope             string
	Offset            int
	SourceIndex       int
	ItemIndex         int
	ArchivedMessageID int64
	Platform          string
}

func parseAdminCallback(data string) (adminPage, error) {
	parts := strings.Split(data, ":")
	if len(parts) < 2 || parts[0] != "admin" {
		return adminPage{}, fmt.Errorf("callback 前缀不合法")
	}

	switch parts[1] {
	case "home":
		return adminPage{Kind: "home"}, nil
	case "sources":
		if len(parts) != 4 {
			return adminPage{}, fmt.Errorf("sources callback 参数不合法")
		}
		offset, err := strconv.Atoi(parts[3])
		if err != nil {
			return adminPage{}, err
		}
		return adminPage{Kind: "sources", Scope: parts[2], Offset: offset}, nil
	case "source":
		if len(parts) != 5 {
			return adminPage{}, fmt.Errorf("source callback 参数不合法")
		}
		sourceIndex, err := strconv.Atoi(parts[3])
		if err != nil {
			return adminPage{}, err
		}
		offset, err := strconv.Atoi(parts[4])
		if err != nil {
			return adminPage{}, err
		}
		return adminPage{Kind: "source", Scope: parts[2], SourceIndex: sourceIndex, Offset: offset}, nil
	case "failed":
		if len(parts) != 3 {
			return adminPage{}, fmt.Errorf("failed callback 参数不合法")
		}
		offset, err := strconv.Atoi(parts[2])
		if err != nil {
			return adminPage{}, err
		}
		return adminPage{Kind: "failed", Offset: offset}, nil
	case "detail":
		if len(parts) < 5 {
			return adminPage{}, fmt.Errorf("detail callback 参数不合法")
		}
		if parts[2] == "source" && len(parts) == 7 {
			sourceIndex, err := strconv.Atoi(parts[4])
			if err != nil {
				return adminPage{}, err
			}
			offset, err := strconv.Atoi(parts[5])
			if err != nil {
				return adminPage{}, err
			}
			itemIndex, err := strconv.Atoi(parts[6])
			if err != nil {
				return adminPage{}, err
			}
			return adminPage{Kind: "detail-source", Scope: parts[3], SourceIndex: sourceIndex, Offset: offset, ItemIndex: itemIndex}, nil
		}
		if parts[2] == "failed" && len(parts) == 5 {
			offset, err := strconv.Atoi(parts[3])
			if err != nil {
				return adminPage{}, err
			}
			itemIndex, err := strconv.Atoi(parts[4])
			if err != nil {
				return adminPage{}, err
			}
			return adminPage{Kind: "detail-failed", Offset: offset, ItemIndex: itemIndex}, nil
		}
		return adminPage{}, fmt.Errorf("detail callback 参数不合法")
	case "resync-entry":
		if len(parts) < 5 {
			return adminPage{}, fmt.Errorf("resync-entry callback 参数不合法")
		}
		if parts[2] == "source" && len(parts) == 7 {
			sourceIndex, err := strconv.Atoi(parts[4])
			if err != nil {
				return adminPage{}, err
			}
			offset, err := strconv.Atoi(parts[5])
			if err != nil {
				return adminPage{}, err
			}
			itemIndex, err := strconv.Atoi(parts[6])
			if err != nil {
				return adminPage{}, err
			}
			return adminPage{Kind: "resync-entry-source", Scope: parts[3], SourceIndex: sourceIndex, Offset: offset, ItemIndex: itemIndex}, nil
		}
		if parts[2] == "failed" && len(parts) == 5 {
			offset, err := strconv.Atoi(parts[3])
			if err != nil {
				return adminPage{}, err
			}
			itemIndex, err := strconv.Atoi(parts[4])
			if err != nil {
				return adminPage{}, err
			}
			return adminPage{Kind: "resync-entry-failed", Offset: offset, ItemIndex: itemIndex}, nil
		}
		return adminPage{}, fmt.Errorf("resync-entry callback 参数不合法")
	case "detail-id":
		if len(parts) != 3 {
			return adminPage{}, fmt.Errorf("detail-id callback 参数不合法")
		}
		archivedMessageID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return adminPage{}, err
		}
		return adminPage{Kind: "detail-id", ArchivedMessageID: archivedMessageID}, nil
	case "resync-entry-id":
		if len(parts) != 3 {
			return adminPage{}, fmt.Errorf("resync-entry-id callback 参数不合法")
		}
		archivedMessageID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return adminPage{}, err
		}
		return adminPage{Kind: "resync-entry-id", ArchivedMessageID: archivedMessageID}, nil
	case "resync-run":
		if len(parts) != 4 {
			return adminPage{}, fmt.Errorf("resync-run callback 参数不合法")
		}
		archivedMessageID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return adminPage{}, err
		}
		return adminPage{Kind: "resync-run", ArchivedMessageID: archivedMessageID, Platform: parts[3]}, nil
	default:
		return adminPage{}, fmt.Errorf("未知 callback 动作")
	}
}

func renderSourcesPage(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config, scope string, offset int) {
	syncTargetsOnly := scope == "targets"
	sources, err := adminqueryservice.ListSources(config, syncTargetsOnly)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载来源列表失败: %v", err))
		return
	}

	text, markup := buildSourcesPage(scope, offset, sources)
	respondAdminPage(ctx, b, update, text, markup)
}

func renderSourceMessagesPage(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config, scope string, sourceIndex int, offset int) {
	sources, err := adminqueryservice.ListSources(config, scope == "targets")
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载来源列表失败: %v", err))
		return
	}

	if sourceIndex < 0 || sourceIndex >= len(sources) {
		respondAdminError(ctx, b, update, "来源已不存在")
		return
	}

	messages, err := adminqueryservice.ListMessagesBySource(sources[sourceIndex].SourceID)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载消息列表失败: %v", err))
		return
	}

	text, markup := buildSourceMessagesPage(scope, sourceIndex, offset, sources[sourceIndex], messages)
	respondAdminPage(ctx, b, update, text, markup)
}

func renderFailedMessagesPage(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config, offset int) {
	messages, err := adminqueryservice.ListFailedSyncMessages(config)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载异常同步消息失败: %v", err))
		return
	}

	text, markup := buildFailedMessagesPage(offset, messages)
	respondAdminPage(ctx, b, update, text, markup)
}

func renderMessageDetailFromSource(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config, scope string, sourceIndex int, offset int, itemIndex int) {
	message, err := resolveSourceMessage(config, scope, sourceIndex, itemIndex)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载消息详情失败: %v", err))
		return
	}

	detail, err := adminqueryservice.GetMessageDetail(message.ArchivedMessageID)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载消息详情失败: %v", err))
		return
	}

	text, markup := buildDetailPage(detail, fmt.Sprintf("admin:source:%s:%d:%d", scope, sourceIndex, offset), fmt.Sprintf("admin:resync-entry:source:%s:%d:%d:%d", scope, sourceIndex, offset, itemIndex))
	respondAdminPage(ctx, b, update, text, markup)
}

func renderMessageDetailFromFailed(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config, offset int, itemIndex int) {
	message, err := resolveFailedMessage(config, itemIndex)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载异常消息详情失败: %v", err))
		return
	}

	detail, err := adminqueryservice.GetMessageDetail(message.ArchivedMessageID)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载异常消息详情失败: %v", err))
		return
	}

	text, markup := buildDetailPage(detail, fmt.Sprintf("admin:failed:%d", offset), fmt.Sprintf("admin:resync-entry:failed:%d:%d", offset, itemIndex))
	respondAdminPage(ctx, b, update, text, markup)
}

func renderMessageDetailByID(ctx context.Context, b *bot.Bot, update *models.Update, archivedMessageID int64) {
	detail, err := adminqueryservice.GetMessageDetail(archivedMessageID)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载消息详情失败: %v", err))
		return
	}

	text, markup := buildDetailPage(detail, "admin:home", fmt.Sprintf("admin:resync-entry-id:%d", archivedMessageID))
	respondAdminPage(ctx, b, update, text, markup)
}

func renderResyncEntryFromSource(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config, scope string, sourceIndex int, offset int, itemIndex int) {
	message, err := resolveSourceMessage(config, scope, sourceIndex, itemIndex)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载重同步入口失败: %v", err))
		return
	}

	detail, err := adminqueryservice.GetMessageDetail(message.ArchivedMessageID)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载重同步入口失败: %v", err))
		return
	}

	text, markup := buildResyncEntryPage(detail, fmt.Sprintf("admin:detail:source:%s:%d:%d:%d", scope, sourceIndex, offset, itemIndex))
	respondAdminPage(ctx, b, update, text, markup)
}

func renderResyncEntryFromFailed(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config, offset int, itemIndex int) {
	message, err := resolveFailedMessage(config, itemIndex)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载重同步入口失败: %v", err))
		return
	}

	detail, err := adminqueryservice.GetMessageDetail(message.ArchivedMessageID)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载重同步入口失败: %v", err))
		return
	}

	text, markup := buildResyncEntryPage(detail, fmt.Sprintf("admin:detail:failed:%d:%d", offset, itemIndex))
	respondAdminPage(ctx, b, update, text, markup)
}

func renderResyncEntryByID(ctx context.Context, b *bot.Bot, update *models.Update, archivedMessageID int64) {
	detail, err := adminqueryservice.GetMessageDetail(archivedMessageID)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("加载重同步入口失败: %v", err))
		return
	}

	text, markup := buildResyncEntryPage(detail, fmt.Sprintf("admin:detail-id:%d", archivedMessageID))
	respondAdminPage(ctx, b, update, text, markup)
}

func renderManualResyncResult(ctx context.Context, b *bot.Bot, update *models.Update, config Entity.Config, archivedMessageID int64, platform string) {
	result, err := syncservice.ManualResync(config, archivedMessageID, platform)
	if err != nil {
		respondAdminError(ctx, b, update, fmt.Sprintf("执行重同步失败: %v", err))
		return
	}

	text, markup := buildManualResyncResultPage(archivedMessageID, platform, result)
	respondAdminPage(ctx, b, update, text, markup)
}

func resolveSourceMessage(config Entity.Config, scope string, sourceIndex int, itemIndex int) (adminqueryservice.MessageSummary, error) {
	sources, err := adminqueryservice.ListSources(config, scope == "targets")
	if err != nil {
		return adminqueryservice.MessageSummary{}, err
	}
	if sourceIndex < 0 || sourceIndex >= len(sources) {
		return adminqueryservice.MessageSummary{}, fmt.Errorf("来源下标越界")
	}
	messages, err := adminqueryservice.ListMessagesBySource(sources[sourceIndex].SourceID)
	if err != nil {
		return adminqueryservice.MessageSummary{}, err
	}
	if itemIndex < 0 || itemIndex >= len(messages) {
		return adminqueryservice.MessageSummary{}, fmt.Errorf("消息下标越界")
	}
	return messages[itemIndex], nil
}

func resolveFailedMessage(config Entity.Config, itemIndex int) (adminqueryservice.MessageSummary, error) {
	messages, err := adminqueryservice.ListFailedSyncMessages(config)
	if err != nil {
		return adminqueryservice.MessageSummary{}, err
	}
	if itemIndex < 0 || itemIndex >= len(messages) {
		return adminqueryservice.MessageSummary{}, fmt.Errorf("异常消息下标越界")
	}
	return messages[itemIndex], nil
}

func renderOverview(overview adminqueryservice.Overview) string {
	return fmt.Sprintf(
		"管理首页\n\n消息总量: %d\n来源总数: %d\n可同步频道数: %d\n\n请选择要查看的管理视图。",
		overview.MessageCount,
		overview.SourceCount,
		overview.SyncTargetSourceCount,
	)
}

func buildHomeKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "概览", CallbackData: "admin:home"}, {Text: "来源", CallbackData: "admin:sources:all:0"}},
		{{Text: "异常同步", CallbackData: "admin:failed:0"}, {Text: "可同步频道", CallbackData: "admin:sources:targets:0"}},
	}}
}

func buildSourcesPage(scope string, offset int, sources []adminqueryservice.SourceSummary) (string, *models.InlineKeyboardMarkup) {
	label := "全部来源"
	if scope == "targets" {
		label = "可同步频道"
	}

	start, end := pageBounds(offset, len(sources))
	var builder strings.Builder
	builder.WriteString(label)
	builder.WriteString("\n\n")
	if len(sources) == 0 {
		builder.WriteString("当前没有可展示的来源。")
	} else {
		for idx := start; idx < end; idx++ {
			source := sources[idx]
			builder.WriteString(fmt.Sprintf("%d. %s | 归档 %d | 同步目标: %t\n", idx+1, source.SourceID, source.ArchivedCount, source.SyncTarget))
		}
	}

	rows := make([][]models.InlineKeyboardButton, 0)
	for idx := start; idx < end; idx++ {
		source := sources[idx]
		rows = append(rows, []models.InlineKeyboardButton{{Text: fmt.Sprintf("查看 %s", source.SourceID), CallbackData: fmt.Sprintf("admin:source:%s:%d:0", scope, idx)}})
	}
	rows = append(rows, paginationRow(fmt.Sprintf("admin:sources:%s", scope), offset, len(sources))...)
	rows = append(rows, []models.InlineKeyboardButton{{Text: "返回首页", CallbackData: "admin:home"}})

	return builder.String(), &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func buildSourceMessagesPage(scope string, sourceIndex int, offset int, source adminqueryservice.SourceSummary, messages []adminqueryservice.MessageSummary) (string, *models.InlineKeyboardMarkup) {
	start, end := pageBounds(offset, len(messages))
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("来源消息\n\n来源: %s\n归档数量: %d\n\n", source.SourceID, source.ArchivedCount))
	if len(messages) == 0 {
		builder.WriteString("当前来源下没有可展示的消息。")
	} else {
		for idx := start; idx < end; idx++ {
			msg := messages[idx]
			builder.WriteString(fmt.Sprintf("%d. TG 消息 %d | 异常: %t\n", idx+1, msg.TelegramMessageID, msg.HasSyncFailure))
		}
	}

	rows := make([][]models.InlineKeyboardButton, 0)
	for idx := start; idx < end; idx++ {
		msg := messages[idx]
		rows = append(rows, []models.InlineKeyboardButton{{Text: fmt.Sprintf("查看消息 %d", msg.TelegramMessageID), CallbackData: fmt.Sprintf("admin:detail:source:%s:%d:%d:%d", scope, sourceIndex, offset, idx)}})
	}
	rows = append(rows, paginationRow(fmt.Sprintf("admin:source:%s:%d", scope, sourceIndex), offset, len(messages))...)
	rows = append(rows, []models.InlineKeyboardButton{{Text: "返回来源列表", CallbackData: fmt.Sprintf("admin:sources:%s:0", scope)}})

	return builder.String(), &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func buildFailedMessagesPage(offset int, messages []adminqueryservice.MessageSummary) (string, *models.InlineKeyboardMarkup) {
	start, end := pageBounds(offset, len(messages))
	var builder strings.Builder
	builder.WriteString("异常同步消息\n\n")
	if len(messages) == 0 {
		builder.WriteString("当前没有同步异常消息。")
	} else {
		for idx := start; idx < end; idx++ {
			msg := messages[idx]
			builder.WriteString(fmt.Sprintf("%d. %s / TG %d\n", idx+1, msg.SourceID, msg.TelegramMessageID))
		}
	}

	rows := make([][]models.InlineKeyboardButton, 0)
	for idx := start; idx < end; idx++ {
		msg := messages[idx]
		rows = append(rows, []models.InlineKeyboardButton{{Text: fmt.Sprintf("查看异常消息 %d", msg.TelegramMessageID), CallbackData: fmt.Sprintf("admin:detail:failed:%d:%d", offset, idx)}})
	}
	rows = append(rows, paginationRow("admin:failed", offset, len(messages))...)
	rows = append(rows, []models.InlineKeyboardButton{{Text: "返回首页", CallbackData: "admin:home"}})

	return builder.String(), &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func buildDetailPage(detail adminqueryservice.MessageDetail, backCallback string, resyncEntryCallback string) (string, *models.InlineKeyboardMarkup) {
	var builder strings.Builder
	builder.WriteString("消息详情\n\n")
	builder.WriteString(fmt.Sprintf("归档消息 ID: %d\n", detail.ArchivedMessageID))
	builder.WriteString(fmt.Sprintf("来源: %s\n", detail.SourceID))
	builder.WriteString(fmt.Sprintf("TG 消息 ID: %d\n", detail.TelegramMessageID))
	builder.WriteString(fmt.Sprintf("来源链接: %s\n", detail.SourceLink))
	builder.WriteString(fmt.Sprintf("归档时间: %s\n\n", detail.ArchivedAt.Format("2006-01-02 15:04:05")))
	builder.WriteString("同步状态:\n")
	if len(detail.LatestStatuses) == 0 {
		builder.WriteString("- 当前没有同步记录\n")
	} else {
		for _, status := range detail.LatestStatuses {
			builder.WriteString(fmt.Sprintf("- %s | %s | 链接: %s | 错误: %s\n", status.Platform, status.Status, status.RemoteURL, status.ErrorMessage))
		}
	}

	return builder.String(), &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "进入重同步入口", CallbackData: resyncEntryCallback}},
		{{Text: "返回上级", CallbackData: backCallback}, {Text: "返回首页", CallbackData: "admin:home"}},
	}}
}

func buildResyncEntryPage(detail adminqueryservice.MessageDetail, backCallback string) (string, *models.InlineKeyboardMarkup) {
	var builder strings.Builder
	builder.WriteString("重同步入口\n\n")
	builder.WriteString(fmt.Sprintf("归档消息 ID: %d\n", detail.ArchivedMessageID))
	builder.WriteString(fmt.Sprintf("来源: %s\n", detail.SourceID))
	builder.WriteString("\n当前可进入的重同步范围:\n")
	builder.WriteString("- 全部平台\n")
	for _, status := range detail.LatestStatuses {
		builder.WriteString(fmt.Sprintf("- %s\n", status.Platform))
	}
	builder.WriteString("\n请选择要执行的重同步范围。")

	rows := [][]models.InlineKeyboardButton{{{Text: "重同步全部平台", CallbackData: fmt.Sprintf("admin:resync-run:%d:all", detail.ArchivedMessageID)}}}
	platformRow := make([]models.InlineKeyboardButton, 0, len(detail.LatestStatuses))
	for _, status := range detail.LatestStatuses {
		platformRow = append(platformRow, models.InlineKeyboardButton{Text: fmt.Sprintf("重同步 %s", status.Platform), CallbackData: fmt.Sprintf("admin:resync-run:%d:%s", detail.ArchivedMessageID, platformAlias(status.Platform))})
	}
	if len(platformRow) > 0 {
		rows = append(rows, platformRow)
	}
	rows = append(rows, []models.InlineKeyboardButton{{Text: "返回消息详情", CallbackData: backCallback}, {Text: "返回首页", CallbackData: "admin:home"}})
	return builder.String(), &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func buildManualResyncResultPage(archivedMessageID int64, platform string, result syncservice.ManualResyncResult) (string, *models.InlineKeyboardMarkup) {
	var builder strings.Builder
	builder.WriteString("重同步结果\n\n")
	builder.WriteString(fmt.Sprintf("归档消息 ID: %d\n", archivedMessageID))
	builder.WriteString(fmt.Sprintf("请求范围: %s\n\n", platformDisplay(platform)))
	if !result.Requested {
		builder.WriteString(fmt.Sprintf("未执行: %s", result.Reason))
	} else {
		for _, item := range result.Results {
			builder.WriteString(fmt.Sprintf("- %s | success=%t | 截断=%t | 链接=%s | 错误=%s\n", item.Platform, item.Success, item.Truncated, item.RemoteURL, item.ErrorMessage))
		}
	}

	return builder.String(), &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "刷新消息详情", CallbackData: fmt.Sprintf("admin:detail-id:%d", archivedMessageID)}},
		{{Text: "返回首页", CallbackData: "admin:home"}},
	}}
}

func platformAlias(platform string) string {
	switch strings.ToLower(platform) {
	case "bluesky":
		return "bs"
	case "mastodon":
		return "md"
	case "twitter":
		return "tw"
	default:
		return strings.ToLower(platform)
	}
}

func platformDisplay(platform string) string {
	switch syncservice.NormalizePlatform(platform) {
	case "all":
		return "全部平台"
	case "bluesky":
		return "BlueSky"
	case "mastodon":
		return "Mastodon"
	case "twitter":
		return "Twitter"
	default:
		return platform
	}
}

func paginationRow(base string, offset int, total int) [][]models.InlineKeyboardButton {
	row := make([]models.InlineKeyboardButton, 0, 2)
	if offset-adminPageSize >= 0 {
		row = append(row, models.InlineKeyboardButton{Text: "上一页", CallbackData: fmt.Sprintf("%s:%d", base, offset-adminPageSize)})
	}
	if offset+adminPageSize < total {
		row = append(row, models.InlineKeyboardButton{Text: "下一页", CallbackData: fmt.Sprintf("%s:%d", base, offset+adminPageSize)})
	}
	if len(row) == 0 {
		return nil
	}
	return [][]models.InlineKeyboardButton{row}
}

func pageBounds(offset int, total int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + adminPageSize
	if end > total {
		end = total
	}
	return offset, end
}

func respondAdminPage(ctx context.Context, b *bot.Bot, update *models.Update, text string, markup *models.InlineKeyboardMarkup) {
	if update != nil && update.CallbackQuery != nil && update.CallbackQuery.Message.Message != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
			MessageID:   update.CallbackQuery.Message.Message.ID,
			Text:        text,
			ReplyMarkup: markup,
		})
		return
	}

	if update != nil && update.Message != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        text,
			ReplyMarkup: markup,
		})
	}
}

func respondAdminError(ctx context.Context, b *bot.Bot, update *models.Update, text string) {
	respondAdminPage(ctx, b, update, text, &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{Text: "返回首页", CallbackData: "admin:home"}}}})
}
