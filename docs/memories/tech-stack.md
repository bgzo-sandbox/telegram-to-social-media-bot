---
title: 技术栈选择与规范
created: 2026-04-30T00:00:00
modified: 2026-05-04T14:30:00
description: 保证项目使用最简单、最健壮的技术栈，实施过程中可以依据实际情况引入更多元的技术栈，但必须保证最终文件落地到项目中，保证日后可追溯 (严格禁止在 /tmp 目录黑箱操作)。
tags:
  - ai-notes
---

## 通用编程原则

1. 严格禁止使用 Emoji
2. 注重模块化（多文件）和禁止单体巨文件（monolith），代码行数 < 1000；
3. 任何代码必须包含单元测试，且测试覆盖率尽量达到 100%；
4. 任何大段的代码必须包含文档注释，且注释内容须易于理解、准确，最好使用简体中文；

## 当前技术栈

- 语言：Go。
- 数据库：SQLite。
- ORM：Gorm。
- Telegram：Telebot。
- CLI：Cobra。
- 社交平台适配：Twitter、Mastodon、BlueSky。

## 当前社媒同步能力

- Mastodon：支持纯文字和单张图片。
- BlueSky：支持纯文字和单张图片。
- Twitter：支持纯文字和单张图片。
- 当前统一策略是：优先图文发送，失败时降级为纯文字。

## R2 / S3 适配

- `pkg/S3Utils`：Cloudflare R2（S3 兼容）上传能力，aws-sdk-go-v2 + 静态凭证，path-style 寻址。
- `internal/service/attachmentmigrationservice`：历史图片附件批量上传回填 S3Url 并重写 Markdown。
- `Attachment.S3Url` 非空时 Markdown 导出优先使用 R2 公开 URL，否则回落本地路径。

## 配置入口

- 唯一配置文件是 `config/config.yaml`。
- 运行命令保持 `./tg sync -c ./config/config.yaml`。
- `pipeline.executionMode` 只允许 `serial` 或 `async_experimental`。
- 未知执行模式必须自动回退到 `serial`。

## 关键配置项

- `token`：Telegram bot 凭据。
- `output.json` 和 `output.json_dir`：是否输出原始 JSON 及其目录。
- `output.person_dir` 和 `output.channel_dir`：归档目录。
- `log.dir`：日志目录，也是 SQLite 基准目录。
- `template.dir`：Markdown 模板路径。
- `template.file_name_template`：归档文件名模板，支持 `{{.id}}` 和 `{{.title-filename-truncated}}`。
- `targetUserList`：通知目标用户。
- `socialMediaSync.enable`：同步总开关。
- `socialMediaSync.targetChannel`：允许同步的频道名单。
- `socialMediaSync.mastodon.*`、`twitter.*`、`bluesky.*`：平台配置。
- `r2.*`：Cloudflare R2 上传配置（`enable`、`server_address`、`bucket`、`access_key_id`、`secret_access_key`、`path`、`public_address`）。
- `notification.*`：历史遗留，视为废弃。

## 工程规则

- 可读性优先，不追求花哨实现。
- 保持模块化，避免把归档、同步、通知重新揉回 `main.go`。
- 公共实体统一放在 `internal/Entity`。
- 去重优先依赖数据库唯一约束。
- 归档模板字段与文件名模板字段都由 `archiveservice` 统一提供，不允许在调用方重复拼装。
- 新平台只能走适配层扩展。
- 默认串行，异步只能渐进演进，不能直接替换现有语义。
- 有行为变化时必须补测试。
- 图片同步当前只做单图，不扩到多图和视频。

## 文档规则

- memories 只保留稳定事实。
- 实施计划、迁移步骤、发布记录放在 `docs/implementation-plans/` 或其他 docs，不放在 memories。
- 改动如果影响目标、边界或约束，必须同步更新这 3 份核心文档。

## 最小验证清单

1. 配置可以正常加载。
2. 串行模式可以正常启动。
3. 实验模式与串行模式行为一致。
4. 非目标频道不会触发同步。
5. 核心服务测试保持通过。

