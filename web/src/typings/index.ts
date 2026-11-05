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
  icon: IconConfig
  sort: number
}

export interface ItemGroup {
  id: number
  title: string
  sort: number
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
  fileName: string
  src: string
  fileType: number
  size: number
  createdAt: string
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
