# AGENTS.md

## 构建和测试命令
```bash
# 后端
go build ./...
go test ./...
golangci-lint run

# 前端
pnpm install
pnpm run build
pnpm run lint
pnpm run dev
```

## 架构
- **项目类型**: Amp 反代管理系统 (全栈)
- **后端语言**: Golang
- **前端框架**: React + shadcn/ui
- **功能模块**: 用户管理、管理员界面、Amp 代理
- **参考实现**: `CLIProxyAPI/` (只读，包含 amp 代理核心实现)
- **入口文件**: TODO - 请指定主文件 (例如: main.go, src/main.tsx)

## 代码风格
- 遵循代码库中的现有模式
- 使用一致的命名规范 (根据语言选择 camelCase/snake_case)
- 显式处理错误，不要静默忽略
- 保持函数小而专注

## 备注
- 这是一个新项目，请随着代码库的发展更新此文件

## Playwright 浏览器工具

使用 Playwright MCP 工具查看和测试网页效果：

```bash
# 导航和截图
mcp__playwright__browser_navigate      # 打开 URL
mcp__playwright__browser_snapshot      # 获取页面快照（推荐用于交互）
mcp__playwright__browser_take_screenshot  # 截图保存

# 页面交互
mcp__playwright__browser_click         # 点击元素
mcp__playwright__browser_type          # 输入文本
mcp__playwright__browser_hover         # 悬停元素
mcp__playwright__browser_fill_form     # 填写表单

# 调试
mcp__playwright__browser_console_messages  # 查看控制台日志
mcp__playwright__browser_network_requests  # 查看网络请求
mcp__playwright__browser_close         # 关闭浏览器
```

## shadcn/ui 组件

使用 MCP 工具查询和添加 shadcn 组件：

```bash
# 查询可用组件
mcp__shadcn__search_items_in_registries  # 搜索组件
mcp__shadcn__list_items_in_registries    # 列出所有组件
mcp__shadcn__view_items_in_registries    # 查看组件详情

# 获取组件示例
mcp__shadcn__get_item_examples_from_registries  # 获取使用示例

# 添加组件
mcp__shadcn__get_add_command_for_items  # 获取添加命令
# 例如: npx shadcn@latest add button card
```

## 提交规范

### Commit 格式
```
feat(api): 添加批量注册功能
fix(core): 修复 cookies 过期检测
refactor(ui): 优化任务列表渲染
docs: 更新 README
```
