export interface IconConfig {
  itemType: 1 | 2 | 3
  src?: string
  text?: string
  backgroundColor?: string
}

export interface ItemIcon {
  id: number
  itemIconGroupId: number
  title: string
  url: string
  lanUrl: string
  description: string
  openMethod: number
  healthCheck: boolean
  icon: IconConfig
  sort: number
}

export interface ItemGroup {
  id: number
  pageId: number
  title: string
  sort: number
}

export interface ItemIconPage {
  id: number
  title: string
  sort: number
  columns: number
}

export interface SearchEngine {
  name: string
  url: string
  icon?: string
}

export type ThemeMode = 'light' | 'dark' | 'auto'

export interface PanelConfig {
  logoText: string
  logoColor: string
  logoFontFamily: string
  logoGlassEffect: boolean
  headerOpacity: number
  cardGlassOpacity: number
  footerOffsetY: number
  footerHtml: string
  iconStyle: number
  maxWidth: string
  maxWidthUnit: string
  marginTop: number
  marginBottom: number
  marginX: number
  itemGap: number
  showHeader: boolean
  showClock: boolean
  showSearch: boolean
  showFooter: boolean
  hideTitle: boolean
  hideDescription: boolean
  searchFilterItems: boolean
  showSystemMonitor: boolean
  systemMonitorShowTitle: boolean
  systemMonitorPublic: boolean
  cardRadius: number
  backgroundImage: string
  backgroundBlur: number
  backgroundMask: number
  clockShowSeconds: boolean
  clockShowDate: boolean
  searchPlaceholder: string
  searchOpenInNewTab: boolean
  themeMode: ThemeMode
  accentColor: string
  headerAlign: 'center' | 'left'
}

export interface PanelSettings {
  public_visit_mode?: string
  site_title?: string
  allow_register?: string
  [key: string]: string | undefined
}

export interface PanelData {
  publicVisitMode: boolean
  panelConfig: PanelConfig | null
  searchEngines: SearchEngine[] | null
  pages: ItemIconPage[]
  groups: ItemGroup[]
  items: ItemIcon[]
  settings: PanelSettings
}

export interface User {
  id: number
  username: string
  nickname: string
  role: number
  status: number
  createdAt: string
}

export interface FileRecord {
  id: number
  userId: number
  fileName: string
  src: string
  thumb: string
  fileType: number
  size: number
  syncStatus: number
  auditRemark: string
  cloudKey: string
  cloudUrl: string
  syncedAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface AuditFile extends FileRecord {
  userName: string
}

export interface HealthResult {
  url: string
  online: boolean
  status: number
  latencyMs: number
  error?: string
  checkedAt: number
}

export interface CloudGitHubConfig {
  owner: string
  repo: string
  branch: string
  token: string
  basePath: string
  useCdn: boolean
}

export interface CloudS3Config {
  endpoint: string
  accountId: string
  region: string
  accessKeyId: string
  secretAccessKey: string
  bucket: string
  publicBaseUrl: string
  insecure: boolean
}

export interface CloudConfig {
  enabled: boolean
  provider: 'github' | 's3'
  dualWrite: boolean
  github: CloudGitHubConfig
  s3: CloudS3Config
}

export interface CloudImage {
  key: string
  name: string
  size: number
  url: string
  category: number
  updatedAt: string
}

export interface CloudBackup {
  key: string
  name: string
  size: number
  url: string
  updatedAt: string
}

export interface CloudSyncResult {
  success: number
  failed: string[]
  total?: number
}

export interface RestoreStats {
  users: number
  pages?: number
  groups: number
  items: number
  files: number
  uploads: number
}

export interface BackupPayload {
  version: string
  exportedAt: string
  panelConfig: PanelConfig
  searchEngines: SearchEngine[]
  groups: ItemGroup[]
  items: ItemIcon[]
  settings?: unknown[]
}

export interface MonitorData {
  cpuPercent: number
  cpuCores: number
  cpuModel: string
  memTotal: number
  memUsed: number
  memPercent: number
  diskPath: string
  diskTotal: number
  diskUsed: number
  diskFree: number
  diskPercent: number
  netSent: number
  netRecv: number
  uptime: number
  hostname: string
  platform: string
  updatedAt: number
}

export interface SiteInfo {
  version: string
  publicVisitMode: boolean
  settings: Record<string, string>
}
