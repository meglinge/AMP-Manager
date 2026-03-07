# SQLite 迁移到 PostgreSQL 方案

## 适用场景

本文档适用于当前仍运行在 **原始 SQLite 版本** 的 AMP Manager 实例，目标是迁移到支持 **SQLite / PostgreSQL 双栈** 的新版本，并最终切换到 PostgreSQL 运行。

适用前提：

1. 当前线上数据主库是 SQLite。
2. 目标版本已经包含数据库双栈支持、前端迁移入口、PostgreSQL dump 导入导出能力。
3. 可以接受一次短暂停机或只读维护窗口。

## 迁移目标

1. 保留现有 SQLite 数据完整性。
2. 将业务数据迁移到 PostgreSQL。
3. 让运行中的服务切换到 PostgreSQL。
4. 后续服务重启后仍然默认使用 PostgreSQL，而不是回退到 SQLite。

## 结论先说

推荐使用 **两阶段迁移**：

1. 先升级到当前双栈版本，但仍然继续跑 SQLite。
2. 再执行 SQLite → PostgreSQL 迁移，并更新部署环境变量切换到 PostgreSQL。

不要直接在老版本上“硬切” PostgreSQL。更稳的做法是先让新版本接管 SQLite，再迁移到 PostgreSQL。

## 风险评估

### 主要风险

1. 老版本 SQLite 表结构与新版本目标结构不完全一致。
2. 迁移时若仍有写入，可能导致 PostgreSQL 中数据落后于最终 SQLite 状态。
3. 如果只做了“运行时切换”，但没有同步修改部署环境变量，服务重启后仍会回到 SQLite。
4. PostgreSQL 连接串、账号权限、网络连通性配置错误，会导致切换失败。

### 风险应对

1. 迁移前先完整备份 SQLite 文件。
2. 迁移时进入维护窗口，暂停写入。
3. 切换完成后立即修改部署配置中的 `DB_TYPE` 和 `DATABASE_URL`。
4. 保留 SQLite 原文件作为回滚依据。

## 推荐迁移路径

### 阶段一：先升级到双栈版本，数据库仍用 SQLite

目的：先把程序升级到当前版本，让新版本在 SQLite 上完成必要的 schema 兼容和运行验证。

步骤：

1. 备份当前 SQLite 数据库文件。
2. 部署当前新版本代码或二进制。
3. 保持环境变量仍为 SQLite：

```bash
DB_TYPE=sqlite
SQLITE_PATH=./data/data.db
```

4. 启动新版本，确认可以正常登录、查看概览、请求日志、系统设置。

为什么这样做：

1. 可以先验证新版本逻辑本身没问题。
2. 可以让新版本在 SQLite 上先跑完兼容迁移。
3. 即使后续 PostgreSQL 切换出问题，也能快速退回到当前阶段。

### 阶段二：从 SQLite 迁移到 PostgreSQL

在新版本已经稳定运行 SQLite 后，再执行真正的数据库切换。

## 前置准备

### 1. 准备 PostgreSQL

需要准备一个可用的 PostgreSQL 实例，并确认：

1. 数据库已创建。
2. 用户有建表、索引、写入、删除权限。
3. 应用服务器能连通 PostgreSQL。

示例连接串：

```bash
postgres://postgres:mysecretpassword@localhost:5432/ampmanager?sslmode=disable
```

### 2. 确认目标版本已上线

目标版本应包含以下能力：

1. 双数据库支持，入口在 [internal/config/config.go](file:///e:/MegAiTools/AMPManager/internal/config/config.go#L41) 和 [internal/database/sqlite.go](file:///e:/MegAiTools/AMPManager/internal/database/sqlite.go#L41)。
2. 内置数据库迁移逻辑，入口在 [internal/database/migration.go](file:///e:/MegAiTools/AMPManager/internal/database/migration.go#L46)。
3. 管理后台数据库迁移任务接口，入口在 [internal/handler/system.go](file:///e:/MegAiTools/AMPManager/internal/handler/system.go#L98)。
4. PostgreSQL dump 导入导出能力，入口在 [internal/database/postgres_dump.go](file:///e:/MegAiTools/AMPManager/internal/database/postgres_dump.go#L23)。

### 3. 备份当前 SQLite 文件

至少保留以下文件：

1. `data.db`
2. 如存在则保留 `data.db-wal`
3. 如存在则保留 `data.db-shm`
4. 如使用请求详情归档，保留 `data_details_archive.db`

## 实施步骤

### 方案 A：推荐，使用前端系统设置页迁移

适合：已有新版本管理后台，且可登录管理员。

步骤：

1. 通知进入维护窗口，暂停写入流量。
2. 登录管理后台，进入“系统设置 → 数据库管理”。
3. 当前数据库应显示为 SQLite。
4. 目标数据库类型选择 PostgreSQL。
5. 填写目标 PostgreSQL 连接串。
6. 建议打开“迁移请求详情归档”。
7. 建议打开“清空目标数据库”。
8. 点击“开始迁移并切换”。
9. 等待任务进度完成到 100%。
10. 观察系统设置页是否显示当前模式已变为 PostgreSQL。
11. 验证用户登录、概览、请求日志、系统设置等核心页面。
12. 立即修改线上部署环境变量并重启服务。

### 方案 B：CLI 迁移，适合维护窗口中手工执行

步骤：

1. 停止应用写入。
2. 在服务器项目根目录执行：

```bash
go run ./cmd/dbtool migrate --source-type sqlite --source ./data/data.db --target-type postgres --target postgres://postgres:mysecretpassword@localhost:5432/ampmanager?sslmode=disable
```

3. 修改部署环境变量：

```bash
DB_TYPE=postgres
DATABASE_URL=postgres://postgres:mysecretpassword@localhost:5432/ampmanager?sslmode=disable
SQLITE_PATH=./data/data.db
```

4. 重启服务。
5. 验证系统设置页显示 PostgreSQL。

## 切换后必须做的事

这是最容易遗漏的一步。

### 必须修改线上部署环境变量

如果你不修改部署环境变量，服务下次重启时仍然可能回到 SQLite。

线上最终应改成：

```bash
DB_TYPE=postgres
DATABASE_URL=postgres://postgres:mysecretpassword@localhost:5432/ampmanager?sslmode=disable
SQLITE_PATH=./data/data.db
```

其中：

1. `DB_TYPE=postgres` 决定服务启动时连 PostgreSQL。
2. `DATABASE_URL` 是 PostgreSQL 连接串。
3. `SQLITE_PATH` 可以保留，不影响 PostgreSQL 启动，但建议保留作为回滚参考。

## 验证清单

迁移完成后，至少检查以下内容：

1. 管理员登录正常。
2. 普通用户登录正常。
3. 概览页正常，无 500。
4. 请求日志页可打开。
5. 系统设置页显示当前模式为 PostgreSQL。
6. API Key 管理、模型列表、用户列表可正常读取。
7. 新增一条测试数据后，在 PostgreSQL 中可查到。
8. 重启服务后，当前模式仍然是 PostgreSQL。

## 回滚方案

如果 PostgreSQL 切换后出现问题，可按以下方式回滚：

1. 停止服务。
2. 把部署环境变量改回：

```bash
DB_TYPE=sqlite
SQLITE_PATH=./data/data.db
DATABASE_URL=
```

3. 使用原来的 SQLite 文件重新启动服务。
4. 如果你在 PostgreSQL 中已经产生新写入，回滚后这部分新写入不会自动回到 SQLite，需要单独处理。

## 生产建议

### 建议一：迁移窗口内暂停写入

虽然系统支持迁移，但为了避免最终一致性问题，生产迁移仍建议在维护窗口执行。

### 建议二：先在预发布环境演练

最好在一份 SQLite 备份上先演练一次完整流程：

1. 升级到新版本跑 SQLite。
2. SQLite → PostgreSQL 迁移。
3. 改环境变量。
4. 重启验证。
5. 演练回滚。

### 建议三：迁移后保留 SQLite 备份至少一段时间

建议保留原始 SQLite 文件和归档库至少 7 天，直到确认 PostgreSQL 稳定。

## 对当前项目的建议结论

如果你现在是“原始 SQLite 版本”，最稳的生产迁移方案是：

1. 先把程序升级到当前双栈版本，但继续跑 SQLite。
2. 验证新版本运行正常。
3. 在维护窗口中通过前端或 CLI 执行 SQLite → PostgreSQL 迁移。
4. 立刻修改线上部署环境变量为 PostgreSQL。
5. 重启服务。
6. 观察一段时间后再决定是否下线旧 SQLite 文件。

这条路线风险最低，也最容易回滚。
