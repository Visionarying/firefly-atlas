# Firefly Atlas 项目约定

## 项目位置

```text
D:\ai\opencode\工作区\firefly-panel
```

Vue 3 + TypeScript 前端，Go + Gin + SQLite 后端，单二进制部署。前端构建产物会复制到 `service/router/webdist/` 并由 Go `embed` 打包。

## 开始工作前

- 涉及版本开发时，先读 `docs/v1.3-handoff.md` 了解当前状态和下一步。
- 不要修改真实管理员密码，不要删除 `data/`。
- 数据库结构或迁移变更前，先备份 `data/database.db`。
- 不确定是否要改动真实数据时，先问用户。

## 常用命令

```powershell
# 前端类型检查
cd web
npx vue-tsc --noEmit

# 前端构建并同步到后端
npm run build:copy

# 后端检查
cd ..\service
gofmt -w .
go vet ./...
go test ./...

# 全量构建（前端 + 后端）
cd ..
powershell -ExecutionPolicy Bypass -File scripts\build.ps1

# Windows GUI 启动器
powershell -ExecutionPolicy Bypass -File scripts\build-launcher.ps1
```

## 版本发布

- 版本号位置：`service/global/global.go`、`web/package.json`、`web/src/components/SettingsModal.vue`。
- 同步更新 `README.md`、`CHANGELOG.md`、`docs/ROADMAP.md`，再运行 `python scripts/build-docs.py`。
- 每个发布版本使用新版本号与代号；不要复用旧版本号。
- 流程：功能分支 → `vue-tsc` / `go vet` → 本地临时实例验证 → 合并 `main` → 推送 → 打标签 → 确认 GHCR 镜像。
- 本地验收不要直接拿真实数据试破坏性操作；优先复制 `data/database.db` 到临时 `DATA_DIR`，用不同端口运行。

## 技术约束

- 保持 `CGO_ENABLED=0`；不要引入需要 CGO 的图片/数据库库。
- 图片压缩使用 JPEG / PNG，不使用 WebP。
- 中文界面为主；i18n 计划在 v1.4，暂不混入功能版本。
- 数据库使用纯 Go SQLite 驱动 `glebarez/sqlite`。
- 备份格式与旧版本兼容优先；修改 `pages` / `groups` / `files` 结构时补兼容逻辑。

## 当前阶段

- 已发布：`v1.3.0「Sentinel」`（服务状态 5 分钟采样、24 小时在线率、在线 → 离线浏览器通知）。
- 镜像：`ghcr.io/visionarying/firefly-atlas:1.3.0`（`linux/amd64` + `linux/arm64`）。
- 线上进程仍运行 `v1.2.2`；`service/firefly-atlas.exe` 已更新为 1.3.0，**重启面板后生效并自动迁移数据库**。
- 下一版本：`v1.3.1`，自定义小组件与 v1.3.0 遗留项，见 `docs/v1.3-handoff.md` 第九节。
- 详细交接：`docs/v1.3-handoff.md`。
