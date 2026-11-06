# Firefly Atlas v1.1.1「Hygiene」

一个面向私有云 / NAS / 家庭工作站的轻量导航面板。V1.1 在“开箱即用”的基础上补齐了页内弹窗、右键菜单、系统状态、在线图标、图库管理、云存储漫游、整站加密备份、服务健康探测与浏览器书签导入等实用能力，所有数据仍保存在本地 SQLite 中。

`v1.1.1` 是合规与发布整理补丁：项目正式更名为 **Firefly Atlas**，补充 MIT License，修正关于页/README 致谢，并同步 GitHub Actions、GHCR 与 Docker 版本注入。

> 视觉与交互参考了 [sun-panel](https://github.com/hslr-s/sun-panel)（MIT）以及朋友的 Firefly-Panel 二次开发版（作者 Roland），按 Vue 3 + Go 重新实现，保留原项目署名。
>
> 版本命名：`v1.1.1` 代号 **Hygiene（整理）**。项目中文名仍可称“萤火导航”，英文名统一为 **Firefly Atlas**；之后每个发布版本都会使用新的版本号与代号，避免功能批次混淆。

## 功能

### 面板与卡片

- 登录 / Token 会话，管理员与普通用户角色
- 管理员可创建、禁用、删除账号并切换角色，普通用户拥有独立空间
- 公开访客只读模式（访客不能编辑任何内容）
- 分组与卡片增删改、编辑模式拖拽排序
- 三种图标形态：图片、文字、SVG
- 图片图标与壁纸本地上传，保存到数据目录
- 一键抓取网站 favicon 并保存到本地，避免外链失效
- 内置 Iconify 在线图标搜索，选中后保存为内联 SVG，离线可用
- 内置“管理员图库选择器”，卡片编辑时可直接选用已上传图标
- 卡片右键菜单：打开内网 / 公网地址、新窗口、页内弹窗、编辑与删除
- 页内 iframe 弹窗，支持刷新、跳转新标签页与嵌入限制提示
- 可隐藏卡片标题 / 描述，悬浮显示名称
- 每张卡片可选开启“在线状态探测”，首页显示绿色 / 红色状态点
- 时钟、日期、多搜索引擎切换与增删改、地址直达
- 搜索时实时过滤面板中的项目，支持方向键与回车快速打开

### 外观与体验

- 毛玻璃卡片 / 通透模式、浅色与深色主题、主色自定义
- 自定义标题字体、字号、页脚 HTML、背景图与遮罩
- 内网 / 公网地址智能切换，支持「自动 / 强制内网 / 强制公网」三态
- 系统状态卡片：CPU、内存、磁盘、网络与运行时间，可设置访客可见
- 首页与登录页公告，内容更新后自动重新提示
- 自定义 CSS 与 JavaScript 注入（仅管理员）
- 设置弹窗采用「系统应用」左侧导航结构，支持拖拽、八方向缩放、最大化
- 响应式布局，适配手机与桌面
- 支持安装为 PWA（离线壳 + 静态资源缓存，API 始终走网络）
- 单二进制部署（前端已嵌入 Go 程序），Docker 一键构建
- GitHub Actions 自动构建 `linux/amd64` 与 `linux/arm64` 多架构镜像

### 图库管理

- 壁纸图库 / 应用图标双分类，支持多选、全选与批量删除
- 在壁纸与应用图标之间移动分类
- 点击文件名重命名，点击缩略图查看原图
- 显示文件大小、同步状态与图库总占用
- 普通用户只能看到和删除自己的文件；后端按 `user_id` 强校验，杜绝水平越权
- 普通用户可提交“同步到云端”申请，管理员在云端待审列表中批准 / 驳回
- 管理员可直接将文件推送到云端图床

### 云存储与漫游

- GitHub 仓库存储：Contents API、私有 / 公开仓库、Token、分支与 BasePath
- Cloudflare R2 / S3 兼容对象存储：Account ID 或自定义 Endpoint、Bucket、Region、公开域名
- 连接测试自动识别私有 / 公有存储，并给出安全提示
- 云端图库分类浏览、批量选择、单张 / 全部拉取到本地
- 云端图片通过后端签名代理预览，私有仓库无需暴露 Token
- 批量删除云端图片与清空云端图床（GitHub 自动限速，避免触发 Secondary Rate Limit）
- 本地 + 云端双写：默认关闭，开启后新上传文件自动镜像到云端
- 云端加密备份历史：下载、还原、删除
- 云端配置独立存放，普通用户的 panel payload 不会收到任何密钥

### 备份与恢复

- 普通用户 JSON 备份：导出 / 导入自己的分组、卡片、风格与搜索引擎
- 管理员整站 ZIP 备份：用户、分组、卡片、系统设置、图库记录与 `uploads/` 图片
- AES-256-GCM 加密备份：scrypt 派生密钥，分块认证加密，密码错误或文件损坏会被拒绝
- 支持从本地上传 `.zip` / `.ffbak` 还原，也支持从云端加密备份一键还原
- 还原时保留当前管理员的登录状态与密码，避免把自己锁在门外
- ZIP 还原包含路径穿越、解压体积与文件数量限制

### 导入与通知

- 导入 Chrome / Edge / Firefox 导出的书签 HTML
- 按书签文件夹分组，或全部追加到一个已有分组
- 自动跳过重复网址与 `javascript:` 等非 HTTP 书签
- 登录失败限流：同一 IP 连续 5 次错误锁定 60 秒

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3、TypeScript、Vite、Pinia、Naive UI、Tailwind CSS、lucide-vue-next |
| 后端 | Go、Gin、GORM |
| 数据库 | SQLite（纯 Go 驱动 `glebarez/sqlite`，无 CGO 依赖） |
| 云存储 | GitHub Contents API、Cloudflare R2 / S3（MinIO Go SDK） |
| 部署 | 单二进制 / Docker 多阶段构建 / GHCR 多架构镜像 |

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
go build -trimpath -o firefly-atlas.exe .

# 4. 启动（数据目录按需修改）
$env:DATA_DIR='..\data'
.\firefly-atlas.exe
```

> **不要直接双击 `service\firefly-atlas.exe`。**  
> 它会把当前目录当作数据目录，而且如果 3002 端口已有实例，会直接闪退。  
> 在 Windows 上请双击根目录的 **`start-firefly-atlas.cmd`**：它会自动使用根目录 `data\`、检测端口并在实例已运行时直接打开浏览器。

打开 <http://127.0.0.1:3002>，默认账号：

```text
用户名：admin
密码：首次启动时自动生成，见控制台日志 `initial password` 一行
```

> 首次登录后请立即在「系统应用 → 我的信息」中修改密码。
>
> 也可以通过 `ADMIN_PASSWORD` 环境变量在首次初始化时指定密码。

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

只使用镜像（推荐直接拉取 GHCR，无需本地构建）：

```bash
docker pull ghcr.io/visionarying/firefly-atlas:1.1.1
docker run -d --name firefly-atlas -p 3002:3002 \
  -v "$PWD/data:/app/data" \
  -e TZ=Asia/Shanghai \
  --restart unless-stopped \
  ghcr.io/visionarying/firefly-atlas:1.1.1
```

也可以本地构建：

```bash
docker build -t firefly-atlas:1.1.1 .
docker run -d --name firefly-atlas -p 3002:3002 \
  -v "$PWD/data:/app/data" \
  -e TZ=Asia/Shanghai \
  --restart unless-stopped \
  firefly-atlas:1.1.1
```

### 多架构镜像

仓库内置 GitHub Actions 工作流，推送 `v*.*.*` 标签或 `main` 分支时会自动构建：

- `linux/amd64`（x86_64 服务器 / PC）
- `linux/arm64`（树莓派、ARM NAS、Apple Silicon 等）

镜像地址为 `ghcr.io/<你的 GitHub 用户名>/firefly-atlas`，具体标签见工作流输出。

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

## 常见问题

### 双击 `service\firefly-atlas.exe` 后黑框一闪而过？

通常不是程序损坏，而是以下两种情况之一：

1. **3002 端口已经有一个实例在运行**  
   第二个实例无法绑定端口，会立即退出。先访问 <http://127.0.0.1:3002> 看看是否已经在运行。

2. **直接双击 exe 使用了错误的数据目录**  
   双击时工作目录是 `service\`，默认 `DATA_DIR=./data` 会指向 `service\data`，而不是根目录真实数据 `data\`。即使启动成功，看到的也是空面板。

正确做法：双击根目录的 **`start-firefly-atlas.cmd`**。它会：

- 使用根目录 `data\`
- 检测 3002 是否已有实例
- 已有实例时直接打开浏览器
- 启动失败时暂停窗口并显示原因

## 配置

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `3002` | HTTP 监听端口 |
| `DATA_DIR` | `./data` | 数据目录，存放 `database.db` 与 `uploads/` |
| `TZ` | 系统时区 | Docker 镜像内建议 `Asia/Shanghai` |
| `GIN_MODE` | `debug` | 生产环境设为 `release` |
| `ADMIN_PASSWORD` | 空 | 仅首次初始化时使用；为空则自动生成随机密码 |
| `TRUSTED_PROXIES` | 空 | 反向代理部署时填代理 IP / CIDR（逗号分隔）；留空表示不信任 `X-Forwarded-For` |

数据目录结构：

```text
data/
├── database.db          # SQLite 数据库
└── uploads/
    ├── wallpapers/      # 壁纸
    └── icons/           # 图标
```

## 云存储配置

在「系统应用 → 云存储与漫游 → 存储配置」中填写。配置只保存在本机数据库，不会出现在普通用户的 panel payload 中。

### GitHub

| 字段 | 说明 |
| --- | --- |
| Owner / Repo | 目标仓库，例如 `your-name/firefly-data` |
| Branch | 默认 `main` |
| Token | Personal Access Token，需 `repo` 读写权限 |
| Base Path | 云端根目录，默认 `firefly-atlas` |
| Use CDN | 仅公开仓库建议开启 jsDelivr |

图片会写入 `images/wallpapers/` 与 `images/icons/`，加密备份写入 `backups/`。GitHub Contents API 对单文件体积有限制，加密备份建议控制在 60MB 以内；更大的备份请改用 R2 / S3 或先导出到本地。

### Cloudflare R2 / S3

| 字段 | 说明 |
| --- | --- |
| Account ID | R2 账户 ID；填写后可自动拼出 Endpoint |
| Endpoint | 自定义 S3 兼容地址；R2 可留空 |
| Bucket | 存储桶名称 |
| Region | R2 使用 `auto` |
| Access Key / Secret | S3 API 凭证 |
| Public Base URL | 可选；留空视为私有存储 |
| 使用 HTTP | 仅局域网 MinIO 等自建服务需要开启 |

> 私有仓库 / 私有桶中的图片通过本面板后端的短时签名代理预览，不依赖公开外链，也不会把用户 Token 写进图片 URL。
>
> 云存储密钥仅管理员可见；请通过 HTTPS 反向代理访问面板。

## 备份与迁移

### 普通用户

「系统应用 → 导出导入 → 我的卡片数据（JSON）」可导出 / 导入当前账号的分组、卡片、风格与搜索引擎配置。JSON 不包含图片二进制。

### 管理员整站备份

「系统应用 → 导出导入 → 整站备份」可导出：

- 全部用户（含密码哈希；不含 Token）
- 分组与卡片
- 用户面板配置
- 系统设置
- 图库文件记录
- `uploads/` 下的壁纸与图标

设置密码后会导出 `.ffbak` 加密包。加密格式为 `FFBK1 + salt + 分块 AES-256-GCM`，密钥通过 scrypt 派生；密码错误、文件被篡改或截断都会拒绝还原。

> 整站备份不包含云存储 Token / Secret 配置，迁移到新服务器后需要重新填写一次。

还原时会保留当前登录管理员的密码与 Token，其他用户 Token 会重置。还原前建议先导出一份当前备份。

### 云端加密备份

在「云存储与漫游 → 加密备份」中设置密码后上传。云端备份文件位于 `backups/firefly-atlas-backup-<时间>.ffbak`，支持下载到本地、从云端还原和删除。

## PWA

v1.1.x 内置 `manifest.webmanifest` 与 `sw.js`：

- 首页与静态资源可离线打开
- `/api/` 请求始终走网络，不会缓存登录态或数据
- Chrome / Edge / Android 可通过浏览器菜单“安装应用”
- iOS Safari 可通过“添加到主屏幕”使用

Service Worker 只在生产构建中注册；开发模式不会缓存页面。

## 目录结构

```text
firefly-atlas/
├── .github/workflows/
│   ├── ci.yml                # 前后端类型检查与构建
│   └── docker-publish.yml    # amd64 / arm64 多架构镜像
├── Dockerfile
├── docker-compose.yml
├── start-firefly-atlas.cmd  # Windows 双击启动（自动使用 data\、检测端口）
├── scripts/
│   └── build.ps1             # 前端 + 后端一键构建
├── service/                  # Go 后端
│   ├── api/                  # 登录、面板、卡片、上传、设置、备份、云存储、健康探测、书签
│   ├── cloud/                # GitHub / S3 / R2 统一客户端
│   ├── database/             # SQLite 初始化与种子数据
│   ├── global/               # 端口、数据目录、全局缓存
│   ├── models/               # 数据模型与默认配置
│   ├── router/               # 路由、Token 中间件、SPA 内嵌
│   │   └── webdist/          # 前端构建产物（由 build:copy 生成）
│   └── main.go
└── web/                      # Vue 3 前端
    ├── public/               # PWA manifest / service worker / 图标
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
- 同一 IP 连续 5 次登录失败会锁定 60 秒（内存实现，重启服务后清零）。
- 默认不信任 `X-Forwarded-For`，避免伪造来源绕过登录限流；置于反向代理之后请设置 `TRUSTED_PROXIES`。
- 跨域请求只回显与本站同源的 `Origin`，不会对任意站点返回 `Access-Control-Allow-Origin`。
- 公开访客模式仅开放只读接口，数据固定读取管理员（UID 1）的空间。
- 系统状态、公告与自定义代码由管理员统一控制；访客只能读取白名单设置。
- 图库删除接口强制按 `user_id` 校验，普通用户无法删除他人或管理员文件。
- 云存储配置与云端备份接口仅管理员可访问；云存储密钥不会进入普通用户响应。
- 云存储 Token / Secret 保存在本地 SQLite 的 `cloud_storages` 表中，请限制数据目录权限并使用 HTTPS。
- 整站还原会校验 ZIP 路径、解压体积与文件数量，避免路径穿越与压缩炸弹。
- v1.1.x 不含 HTTPS，请通过反向代理（Nginx / Caddy / Traefik）配置 TLS 后再暴露到公网。
- 建议定期备份 `data` 目录，并妥善保管云存储 Token / Secret 与备份密码。

## 兼容性

本项目**不兼容** sun-panel 现有的 `database.db` 与备份包。如需迁移，请使用本项目的 JSON / ZIP 备份格式，或重新录入数据。

## 版本与路线图

### 版本命名

- `v1.0.0`：首个可用版本
- `v1.1.0` **Aurora**：页内弹窗、右键菜单、系统状态、图库管理、云存储、加密备份、健康探测、书签导入、PWA
- `v1.1.1` **Hygiene**：项目更名 Firefly Atlas、补 MIT LICENSE、修正关于页/README 致谢、同步 GHCR 与 CI、Docker 版本注入、反向代理与 CORS 安全文档
- 后续每个发布版本使用独立版本号与代号（例如 `v1.2.0 Lumen`），不再重复使用同一版本号。

### 后续方向

- `v1.2 Lumen`：多页面与分栏、图库缩略图持久化与压缩、书签增量导入、数字快捷键 / 搜索前缀 / 版本更新提示
- `v1.3 Sentinel`：服务状态历史与告警、自定义小组件、磁盘挂载多选、可选登录验证码
- `v1.4 Bilingual`：中英双语（只做中文 / 英文）
- 云存储断点续传与大文件分片（GitHub 100MB 限制）
- 反向代理 / HTTPS 一键部署向导

## 许可证

本项目采用 [MIT License](LICENSE) 发布。你可以自由使用、修改、分发和商用，但请保留版权与许可声明。

## 致谢

- [sun-panel](https://github.com/hslr-s/sun-panel)：面板信息架构与交互灵感（作者：红烧猎人）
- [Roland 的 Firefly-Panel 二次开发版](https://github.com/mysecret2022/firefly-panel)：界面与功能思路参考
- [Glance](https://github.com/glanceapp/glance)、[Dashy](https://github.com/Lissy93/dashy)、[Homarr](https://github.com/homarr-labs/homarr)、[Homepage](https://github.com/gethomepage/homepage)、[FlatNas](https://github.com/Garry-QD/FlatNas)：功能取舍与交互思路参考
- [Vue](https://vuejs.org/)、[Vite](https://vite.dev/)、[Naive UI](https://www.naiveui.com/)、[Tailwind CSS](https://tailwindcss.com/)
- [Gin](https://gin-gonic.com/)、[GORM](https://gorm.io/)、[glebarez/sqlite](https://github.com/glebarez/sqlite)
- [MinIO Go SDK](https://github.com/minio/minio-go)：S3 / R2 兼容客户端
- [lucide](https://lucide.dev/) 图标，[霞鹜文楷](https://github.com/lxgw/LxgwWenKai) 字体

原 sun-panel 项目采用 MIT 许可，本项目保留其版权与署名。Firefly Atlas 的代码为独立实现，未复用上述项目的业务代码。
