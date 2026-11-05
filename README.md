# Firefly Panel

一个面向私有云 / NAS / 家庭工作站的轻量导航面板。V1 聚焦「开箱即用」：登录后即可管理分组、卡片与图标，未登录访客可只读浏览，所有数据保存在本地 SQLite 中。

> 视觉与交互参考了 [sun-panel](https://github.com/hslr-s/sun-panel)（MIT）以及朋友的 Firefly-Panel 私有版本，按 Vue 3 + Go 重新实现，保留原项目署名。

## 功能（V1）

- 登录 / Token 会话，管理员与普通用户角色
- 管理员可创建、禁用、删除账号并切换角色，普通用户拥有独立空间
- 公开访客只读模式（访客不能编辑任何内容）
- 分组与卡片增删改、编辑模式拖拽排序
- 三种图标形态：图片、文字、SVG
- 图片图标与壁纸本地上传，保存到数据目录
- 时钟、日期、多搜索引擎切换与增删改、地址直达
- 毛玻璃卡片 / 通透模式、浅色与深色主题、主色自定义
- 自定义标题字体、字号、页脚 HTML、背景图与遮罩
- 内网 / 公网地址智能切换，支持「自动 / 强制内网 / 强制公网」三态
- JSON 备份导出与还原（分组、卡片、风格、系统设置）
- 设置弹窗采用「系统应用」左侧导航结构，支持拖拽、缩放、最大化
- 响应式布局，适配手机与桌面
- 单二进制部署（前端已嵌入 Go 程序），Docker 一键构建

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3、TypeScript、Vite、Pinia、Naive UI、Tailwind CSS、lucide-vue-next |
| 后端 | Go、Gin、GORM |
| 数据库 | SQLite（纯 Go 驱动 `glebarez/sqlite`，无 CGO 依赖） |
| 部署 | 单二进制 / Docker 多阶段构建 |

## 快速开始

### 方式一：本地运行（Windows）

前置：Node.js 18+、Go 1.23+。

```powershell
# 1. 安装前端依赖
cd web
npm install

# 2. 构建前端并同步到 Go embed 目录
npm run build:copy

# 3. 构建后端
cd ..\service
go build -trimpath -o firefly-panel.exe .

# 4. 启动（数据目录按需修改）
$env:DATA_DIR='..\data'
.\firefly-panel.exe
```

打开 <http://127.0.0.1:3002>，默认账号：

```text
用户名：admin
密码：首次启动时自动生成，见控制台日志 `initial password` 一行
```

> 首次登录后请立即在「系统应用 → 我的信息」中修改密码。

也可以使用根目录脚本一键构建：

```powershell
.\scripts\build.ps1
```

### 方式二：Docker Compose（推荐部署）

```bash
docker compose up -d --build
```

- 访问：<http://服务器IP:3002>
- 数据卷：`./data`（包含 `database.db` 与 `uploads/`）
- 停止：`docker compose down`
- 升级：`docker compose up -d --build`

只使用镜像：

```bash
docker build -t firefly-panel:1.0.0 .
docker run -d --name firefly-panel -p 3002:3002 \
  -v "$PWD/data:/app/data" \
  -e TZ=Asia/Shanghai \
  --restart unless-stopped \
  firefly-panel:1.0.0
```

## 开发模式

需要两个终端：

```powershell
# 终端 1：后端
cd service
$env:DATA_DIR='..\data'
go run .

# 终端 2：前端（Vite 已代理 /api 与 /uploads 到 3002）
cd web
npm run dev
```

打开 <http://127.0.0.1:5173>。

常用命令：

```powershell
cd web
npm run type-check   # vue-tsc 类型检查
npm run build:copy   # 构建并把 dist 复制到 service/router/webdist
```

## 配置

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `3002` | HTTP 监听端口 |
| `DATA_DIR` | `./data` | 数据目录，存放 `database.db` 与 `uploads/` |
| `TZ` | 系统时区 | Docker 镜像内建议 `Asia/Shanghai` |
| `GIN_MODE` | `debug` | 生产环境设为 `release` |

数据目录结构：

```text
data/
├── database.db          # SQLite 数据库
└── uploads/
    ├── wallpapers/      # 壁纸
    └── icons/           # 图标
```

## 备份与迁移

在「系统应用 → 导出导入」中导出 JSON 备份，包含：

- 分组与卡片
- 面板风格配置
- 搜索引擎列表
- 系统设置（仅管理员）

注意：JSON 备份不包含图片二进制。迁移服务器时，请同时复制整个 `data` 目录，或重新上传图标 / 壁纸。

## 目录结构

```text
firefly-panel/
├── Dockerfile
├── docker-compose.yml
├── scripts/
│   └── build.ps1             # 前端 + 后端一键构建
├── service/                  # Go 后端
│   ├── api/                  # 登录、面板、卡片、上传、设置、备份
│   ├── database/             # SQLite 初始化与种子数据
│   ├── global/               # 端口、数据目录、全局缓存
│   ├── models/               # 数据模型与默认配置
│   ├── router/               # 路由、Token 中间件、SPA 内嵌
│   │   └── webdist/          # 前端构建产物（由 build:copy 生成）
│   └── main.go
└── web/                      # Vue 3 前端
    ├── scripts/
    │   ├── screenshot.mjs    # Playwright 截图脚本（本地验收用）
    │   └── sync-dist.mjs     # 构建后同步产物
    └── src/
        ├── components/       # 卡片、时钟、搜索、设置弹窗等
        ├── store/            # Pinia 状态与默认配置
        ├── views/            # 主页与登录页
        └── utils/url.ts      # 内外网地址解析与探测
```

## 安全说明

- 密码使用 bcrypt 哈希存储。
- 登录成功后轮换用户 Token，旧 Token 立即失效；前端保存在 `localStorage.firefly_token`。
- 公开访客模式仅开放只读接口，数据固定读取管理员（UID 1）的空间。
- V1 不含 HTTPS，请通过反向代理（Nginx / Caddy / Traefik）配置 TLS 后再暴露到公网。
- 建议定期备份 `data` 目录。

## 兼容性

V1 **不兼容** sun-panel 现有的 `database.db` 与备份包。如需迁移，请使用本项目的 JSON 备份格式，或重新录入数据。

## 路线图

- 多用户与子账号独立空间、图库上传审核
- 云存储与漫游（GitHub 私有仓库 / Cloudflare R2，本地优先回源）
- AES-256-GCM 加密备份
- 服务健康探测与在线状态灯
- 书签导入、浏览器插件、PWA

## 致谢

- [sun-panel](https://github.com/hslr-s/sun-panel)：面板信息架构与交互灵感
- [Vue](https://vuejs.org/)、[Vite](https://vite.dev/)、[Naive UI](https://www.naiveui.com/)、[Tailwind CSS](https://tailwindcss.com/)
- [Gin](https://gin-gonic.com/)、[GORM](https://gorm.io/)、[glebarez/sqlite](https://github.com/glebarez/sqlite)
- [lucide](https://lucide.dev/) 图标，[霞鹜文楷](https://github.com/lxgw/LxgwWenKai) 字体

原 sun-panel 项目采用 MIT 许可，本项目保留其版权与署名。
