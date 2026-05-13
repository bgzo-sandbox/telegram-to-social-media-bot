---
title: 项目架构
created: 2026-04-30T00:00:00
modified: 2026-04-30T17:00:00
description: 架构需要对项目的整体结构进行说明，明确每个文件夹和文件的作用，以及它们之间的关系。需要列出每个文件的作用，以及它们之间的调用关系。如果有数据库，包含完整数据库结构。
tags:
  - ai-notes
---

# 项目架构

## 运行链路

1. `main.go` 启动程序并装配依赖。
2. `bootstrapservice` 加载配置、初始化日志和数据库。
3. `pipelineservice` 统一编排处理顺序。
4. `archiveservice` 负责归档和入库。
5. `syncservice` 负责判定是否同步并调用平台适配器。
6. `notifyservice` 负责生成通知消息。
7. Telegram bot 把通知发回目标聊天。

## 模块边界

### 入口层

- `main.go`：CLI 启动、依赖装配、bot 启动。
- 入口层不写业务规则。
- 当前 bot 入口已注册 `/admin` 命令与 `admin:` callback 路由。

### 服务层

- `internal/service/bootstrapservice`：运行前初始化。
- `internal/service/pipelineservice`：主流程编排。
- `internal/service/archiveservice`：消息归档、模板渲染、文件落盘、数据库写入。
- `internal/service/syncservice`：同步判定、平台分发。
- `internal/service/notifyservice`：通知目标解析、通知内容拼装。
- `internal/service/adminqueryservice`：管理首页、来源列表、消息列表、消息详情等只读查询编排。


### 数据层

- `internal/Entity`：配置和核心数据结构。
- `internal/Database`：SQLite 连接和持久化。
- 数据库唯一约束是重复消息的第一防线。
- 社媒同步结果已独立建模为同步记录表，按消息、平台、尝试次数保留历史。

### 适配层

- `pkg/SocialMediaUtils`：Twitter、Mastodon、BlueSky 适配。
- `pkg/TgUtils`：Telegram 辅助逻辑。
- `pkg/StrUtils`：文本处理辅助逻辑。
- 工具层尽量保持纯逻辑，不混入流程编排。

## 目录职责

- `config/`：唯一配置文件和模板文件。
- `archives/`：Markdown 归档输出。
- `json/`：原始 JSON 输出。
- `docs/memories/`：只保留给 LLM 的稳定上下文。
- `docs/implementation-plans/`：实施计划，不属于长期上下文。
- `scripts/`：迁移或辅助脚本。

## 数据流

### 归档流

- Telegram 消息进入 `archiveservice`。
- 服务解析 `bucket`、`source_id`、`message_id`。
- 先写数据库，再生成或更新归档文件。
- 成功或失败结果交给通知层。

### 同步流

- `syncservice` 先判断总开关。
- 再判断频道是否精确命中目标列表。
- 命中后构建“文本 + 可选单图”载荷，再按平台适配器分发。
- 平台适配器优先尝试图文发送；图失败时回退到纯文字发送。
- 每个平台的结果都返回给通知层。
- 默认同步阶段会在平台分发后立即写入同步记录。
- 同步记录当前固定保存：平台状态、远端链接或远端标识、错误信息、图片使用情况、触发方式、尝试次数。
- 平台分发前当前会先执行统一文本预处理，并按平台上限截断。
- 平台分发结果当前统一返回结构化对象，供通知层和持久化层消费。

### 管理交互流

- Telegram bot 第一版管理交互固定采用按钮回调流。
- 入口层负责接收按钮事件并路由到管理查询或单次操作。
- 管理查询流固定为：首页统计 -> 来源列表 -> 消息列表 -> 消息详情。
- 重同步动作只允许从消息详情触发，不在首页、来源列表、消息列表直接执行。
- 管理交互不改变主处理链路，仍与 `archive -> sync -> notify` 并存。
- 查询服务当前已固定提供：首页统计、来源列表、按来源查看消息、消息详情、同步异常筛选。
- 当前页面跳转主要通过编辑同一条 bot 消息完成。

### 手动重同步流

- 手动重同步当前由 Telegram 管理页按钮触发。
- 入口层只负责接收按钮事件并调用 `syncservice` 的手动重同步能力。
- 执行前先校验消息来源是否仍命中可同步频道。
- 执行中对“同一消息 + 同一范围”应用最小防重入保护。
- 执行后结果立即追加写入同步记录，并回显到 Telegram 管理页。
- 最终回归口径当前以 `go test ./...` 为准，核心管理链路已纳入自动测试覆盖。

## 当前稳定约束

- 主流程顺序固定为 `archive -> sync -> notify`。
- 默认执行模式是 `serial`。
- `async_experimental` 只能在 pipeline 内部切换，不能改变外部接口。
- 新平台必须通过 provider 或 adapter 扩展，不能把平台细节写回主流程。
- 当前图片同步范围固定为单张图片。
- 第一版 bot 管理入口固定为 4 层视图，不引入命令参数驱动跳转。
- 第一版重同步粒度固定为“单个平台”或“当前消息全部平台”。
- 同步记录历史保留策略固定为追加写入，不做覆盖更新。

## 已知限制

- 历史 `notification` 配置仍可能存在于旧配置里，但当前实现不消费。
- 社交平台错误处理仍偏粗，只到成功或失败级别。
- memories 只记录稳定事实，不再记录阶段进度和迁移过程。
   - 读取：先新后旧；
   - 迁移：以 DB 为主，一次性全量补齐本地；
   - 删除：旧单文件必须在 zip 备份完成后再删除。
3. 迁移期错误传播：`source_id` 约束失败等错误必须可被通知阶段消费并展示具体原因。
4. 双向核对口径已冻结为 `(chat_id, message_id)`。

## 本轮新增架构洞察（2026-02-18：代码阶段步骤1）

1. `internal/service/archiveservice/service.go`
   - 变化：仅切换归档写入路径到 `source_id/message_id.md`；
   - 变化：`SourceMeta` 新增 `ArchiveRoot`，用于保持媒体资源仍写在来源分桶根路径；
   - 边界：本轮未实现读兼容（先新后旧）与迁移逻辑。

2. `internal/service/archiveservice/service_test.go`
   - 变化：补充路径断言，确保私聊与频道场景都落在 `source_id/message_id.md`；
   - 意义：先锁定步骤1行为，后续步骤在此基础上增量推进。

## 本轮新增架构洞察（2026-02-19：Front Matter 约束冻结）

1. 归档输出不再只要求“可写入”，还要求 Front Matter 模板字段完整输出；
2. 时间字段格式冻结为 `YYYY-MM-DDTHH:mm:ss`（无时区后缀）；
3. 标题/别名/描述的截断链路固定为：先 `\n` 归一化为空格，再做 `sub(0,50/100)`；
4. `source` 在 person 场景允许空字符串，channel 场景优先构造可访问链接。

## 本轮新增架构洞察（2026-02-19：Front Matter 逻辑落地）

1. `internal/service/archiveservice/service.go`
   - 变化：归档落盘前统一生成强制 Front Matter，并与模板正文拼接输出；
   - 变化：时间字段与持久化时间共享同一归档时间点，降低字段漂移风险；
   - 变化：摘要生成链路固定为“换行归一化 -> 截断 -> 组装字段”。

2. `internal/service/archiveservice/service_test.go`
   - 变化：新增 Front Matter 规则测试（字段完整性、时间格式、换行处理、空 source）；
   - 意义：为步骤2前提供稳定回归基线，避免后续兼容读取改造引入格式回退。

## 本轮新增架构洞察（2026-02-19：步骤2读兼容落地）

1. `internal/service/archiveservice/service.go`
   - 变化：归档去重读取已实现切换期兼容策略：先查新路径 `source_id/message_id.md`，未命中再回退旧路径 `source_id.md`；
   - 意义：在不阻塞新路径写入的前提下，保证迁移窗口内旧数据仍可被识别，避免重复归档。

2. `internal/service/archiveservice/service_test.go`
   - 变化：新增读兼容测试覆盖（新路径命中、旧路径回退命中、双路径未命中）；
   - 意义：将步骤2行为固定为可回归基线，为后续全量补齐与删旧动作提供安全前提。

## 本轮新增架构洞察（2026-02-19：步骤3全量补齐与核对落地）

1. `internal/service/archivemigrationservice/service.go`
   - 变化：新增 `BackfillFromDatabase()`，以数据库为基线执行全量补齐（已存在跳过、缺失补齐）；
   - 变化：补齐输出路径固定为 `source_id/message_id.md`，并复用归档 Front Matter 生成逻辑；
   - 变化：补齐完成后执行键集合核对，发现缺失/孤儿立即返回错误。

2. `internal/service/archivemigrationservice/service_test.go`
   - 变化：新增迁移服务测试（补齐成功、已存在跳过、孤儿文件核对失败）；
   - 意义：把步骤3行为收敛为可验证、可回归的独立服务，降低后续“备份+删旧”阶段风险。

3. `internal/Database/Messages.go`
   - 变化：新增 `ListMessages()` 查询入口（含附件预加载）；
   - 意义：统一迁移读取来源，避免在迁移服务内直接拼接数据库查询细节。

## 本轮新增架构洞察（2026-02-19：步骤4备份后迁移到待删除目录）

1. `internal/service/archivemigrationservice/service.go`
   - 变化：新增 `BackupAndMoveLegacySingleFiles()`，移动前先生成 `260218-old-markdown-archives.zip`；
   - 变化：清理范围限定为 `person/channel` 根目录下旧单文件 `source_id.md`，统一移动到 `archives/260218-legacy-pending-delete`，不触碰新结构目录；
   - 意义：把“处理旧文件前必须备份”从流程约定固化为代码约束，并将硬删除改为可回退的待删除隔离。

2. `internal/service/archivemigrationservice/service_test.go`
   - 变化：新增备份与移动测试（成功路径、无旧文件跳过路径）；
   - 意义：确保步骤4行为可回归验证，避免误删新结构文件。

## 本轮新增架构洞察（2026-02-19：端到端联调与 Go/No-Go 判定）

1. `internal/service/archivemigrationservice/service.go`
   - 变化：新增 `RunMigrationDrill()`，将步骤3（补齐核对）与步骤4（备份+移动到待删除目录）串联为单次联调流程；
   - 变化：新增 `MigrationDrillReport` 与门禁判定（功能/数据/完整性/运维/审批）输出；
   - 变化：以全门禁通过为 `Go`，否则 `No-Go` 并附原因列表。

2. `internal/service/archivemigrationservice/service_test.go`
   - 变化：新增联调判定测试，覆盖审批通过（Go）与审批未通过（No-Go）路径；
   - 意义：把一次性切换门禁从文档规则映射为可执行、可回归的服务行为。

3. `docs/pre-release-regression-checklist.md`
   - 变化：新增 2026-02-19 联调执行记录与 Go/No-Go 结论（当前为 No-Go，待审批门禁完成）；
   - 意义：将门禁状态沉淀为可审计记录，降低发布沟通成本。

## 本轮新增架构洞察（2026-02-19：审批通过后的最终切换结论）

1. `docs/pre-release-regression-checklist.md`
   - 变化：审批门禁状态更新为“通过”，一次性切换结论更新为 `Go`；
   - 意义：当前迁移链路已从“技术可行”进入“可执行发布”状态。

2. `docs/memories/progress.md`
   - 变化：补充“正式迁移演练完成”记录并切换下一步为发布执行；
   - 意义：阶段目标从“实现+联调”正式转为“发布落地+发布后回归”。

3. 执行策略补充（2026-02-19）
   - 变化：迁移执行阶段的文件操作（备份/移动）优先采用 shell 命令直接执行，不再依赖临时 Go 脚本；
   - 意义：减少一次性 runner 维护成本，执行路径更直接、可审计。

## 本轮新增架构洞察（2026-02-19：合并 CLI 步骤1）

1. `main.go`
   - 变化：新增 `buildRootCommand()`，统一构建主 CLI 命令树；
   - 变化：命令入口调整为 `tg` 根命令，下挂 `sync` 与 `migrate`；
   - 变化：新增 `migrate backfill` 子命令，直接复用归档迁移服务的 DB 全量补齐能力；
   - 意义：开始将迁移能力从独立 runner 收敛到主程序入口，降低维护分叉。

2. `main_test.go`
   - 变化：新增命令层测试，覆盖 `sync`/`migrate` 命令发现与 `--help` 可执行性；
   - 意义：为后续继续扩展 `migrate` 子命令提供稳定回归基线。

3. `docs/implementation-plans/260219-combine-cli-progress.md`
   - 变化：新增本轮执行记录，明确“步骤1已完成、步骤2未开始”；
   - 意义：保证分步门禁可追踪，避免未验收跨步实现。

## 本轮新增架构洞察（2026-02-19：合并 CLI 步骤2）

1. `main.go`
   - 变化：在 `migrate` 下新增 `move-legacy` 子命令；
   - 变化：`move-legacy` 直接复用迁移服务 `BackupAndMoveLegacySingleFiles()`；
   - 意义：主 CLI 已覆盖迁移两条核心动作（`backfill` 与 `move-legacy`），进一步减少独立 runner 依赖。

2. `main_test.go`
   - 变化：新增 `move-legacy` 命令测试（可发现性、help 可执行、config 必填门禁）；
   - 意义：保持命令层回归覆盖与参数门禁一致性，降低后续 CLI 收口改造风险。

3. `docs/implementation-plans/260219-combine-cli-progress.md`
   - 变化：记录步骤2完成状态，并将下一步切换为“下线 drill 与 approved 复杂度”；
   - 意义：确保实施节奏可追踪，继续遵守逐步验收门禁。

## 本轮新增架构洞察（2026-02-19：合并 CLI 步骤3）

1. `internal/service/archivemigrationservice/service.go`
   - 变化：移除 `RunMigrationDrill` 与 `approved` / `Go-NoGo` 相关结构；
   - 意义：迁移能力收敛为两个原子动作（`backfill`、`move-legacy`），降低入口复杂度。

2. `internal/service/archivemigrationservice/service_test.go`
   - 变化：删除 `drill` 场景测试，保留并强化核心迁移动作测试；
   - 意义：测试目标与当前产品命令面保持一致，避免“已下线功能”残留测试噪音。

3. `cmd/migrate-runner/main.go`
   - 变化：下线 `drill` 模式与 `approved` 参数，仅保留 `backfill | move-legacy`；
   - 意义：在删除独立 runner 前先完成行为简化，降低下一步删除改动风险。

## 本轮新增架构洞察（2026-02-19：合并 CLI 步骤4）

1. `cmd/migrate-runner/main.go`
   - 变化：独立 runner 已删除；
   - 意义：迁移入口正式收敛为主程序命令树，避免双入口维护漂移。

2. `main.go`
   - 变化：`tg migrate` 成为唯一迁移命令入口（当前包含 `backfill`、`move-legacy`）；
   - 意义：与 `tg sync` 同级，满足“要么同步、要么迁移”的简化操作模型。

3. `README.md`
   - 变化：迁移命令文档统一为 `./tg migrate ...`，移除 runner 与 drill 旧入口；
   - 意义：文档与运行行为一致，降低使用歧义与运维成本。
