import http, { rawHttp } from './request'
import type {
  AuditFile,
  BackupPayload,
  CloudBackup,
  CloudConfig,
  CloudImage,
  CloudSyncResult,
  FileRecord,
  HealthResult,
  ItemGroup,
  ItemIcon,
  MonitorData,
  PanelConfig,
  PanelData,
  RestoreStats,
  SearchEngine,
  SiteInfo,
  User,
} from '@/typings'

export const authApi = {
  login: (username: string, password: string) =>
    http.post<{ token: string; user: User }>('/auth/login', { username, password }),
  info: () => http.get<User>('/auth/info'),
  logout: () => http.post('/auth/logout'),
  changePassword: (oldPassword: string, newPassword: string) =>
    http.post('/auth/password', { oldPassword, newPassword }),
}

export const panelApi = {
  index: () => http.get<PanelData>('/panel/index'),
  public: () => http.get<PanelData>('/panel/public'),
  saveConfig: (panelConfig: PanelConfig, searchEngines: SearchEngine[]) =>
    http.post('/panel/config/save', { panelConfig, searchEngines }),
  saveItem: (item: Partial<ItemIcon>) => http.post<{ id: number }>('/panel/itemIcon/save', item),
  deleteItem: (id: number) => http.delete(`/panel/itemIcon/${id}`),
  sortItems: (ids: number[]) => http.post('/panel/itemIcon/sort', { ids }),
  saveGroup: (group: { id?: number; title: string }) =>
    http.post<{ id: number }>('/panel/itemIconGroup/save', group),
  deleteGroup: (id: number) => http.delete(`/panel/itemIconGroup/${id}`),
  sortGroups: (ids: number[]) => http.post('/panel/itemIconGroup/sort', { ids }),
  health: (urls: string[]) => http.post<HealthResult[]>('/panel/health', { urls }),
  importBookmarks: (form: FormData) =>
    http.post<{ groups: number; items: number; skipped: number; total: number }>(
      '/panel/import/bookmarks',
      form,
      { timeout: 120000 },
    ),
}

export const fileApi = {
  list: (fileType?: number, scope?: 'all') =>
    http.get<FileRecord[]>('/system/file/list', {
      params: { ...(fileType ? { fileType } : {}), ...(scope ? { scope } : {}) },
    }),
  upload: (file: File, fileType: 1 | 2) => {
    const form = new FormData()
    form.append('file', file)
    form.append('fileType', String(fileType))
    return http.post<FileRecord>('/system/file/upload', form)
  },
  remove: (id: number) => http.delete(`/system/file/${id}`),
  batchRemove: (ids: number[]) => http.post<{ deleted: number }>('/system/file/batch-delete', { ids }),
  move: (ids: number[], fileType: 1 | 2) =>
    http.post<{ moved: number }>('/system/file/move', { ids, fileType }),
  rename: (id: number, fileName: string) => http.post('/system/file/rename', { id, fileName }),
  applySync: (ids: number[]) => http.post<{ pending: number }>('/system/file/apply-sync', { ids }),
  uploadCloud: (ids: number[]) =>
    http.post<{ success: number; failed: string[] }>(
      '/system/file/upload-cloud',
      { ids },
      { timeout: 600000 },
    ),
  auditList: () => http.get<AuditFile[]>('/system/file/audit'),
  audit: (ids: number[], action: 'approve' | 'reject') =>
    http.post<{ success?: number; rejected?: number; failed?: string[] }>(
      '/system/file/audit',
      { ids, action },
      { timeout: 600000 },
    ),
  publicIcons: () => http.get<FileRecord[]>('/system/file/public-icons'),
}

export const settingApi = {
  get: () => http.get<{ configName: string; content: string; title: string }[]>('/system/setting'),
  save: (items: { configName: string; content: string }[]) =>
    http.post('/system/setting/save', items),
}

export const userApi = {
  list: () => http.get<User[]>('/system/user/list'),
  save: (data: {
    id?: number
    username: string
    nickname?: string
    password?: string
    role: number
    status: number
  }) => http.post<{ id: number }>('/system/user/save', data),
  remove: (id: number) => http.delete(`/system/user/${id}`),
}

export const backupApi = {
  export: () => http.get<BackupPayload>('/panel/backup/export'),
  import: (data: BackupPayload) => http.post('/panel/backup/import', data),
}

export const siteBackupApi = {
  export: (password: string): Promise<Blob> =>
    rawHttp
      .post('/system/backup/export', { password }, { responseType: 'blob', timeout: 300000 })
      .then((response) => response.data),
  import: (file: File, password: string) => {
    const form = new FormData()
    form.append('file', file)
    form.append('password', password)
    return http.post<RestoreStats>('/system/backup/import', form, { timeout: 600000 })
  },
}

export const cloudApi = {
  getConfig: () => http.get<CloudConfig>('/system/cloud/config'),
  saveConfig: (config: CloudConfig) => http.post('/system/cloud/config', { config }),
  test: (config: CloudConfig) =>
    http.post<{ private: boolean; description: string }>('/system/cloud/test', config, {
      timeout: 60000,
    }),
  listImages: () => http.get<CloudImage[]>('/system/cloud/images', { timeout: 60000 }),
  sync: (keys: string[]) =>
    http.post<CloudSyncResult>('/system/cloud/sync', { keys }, { timeout: 600000 }),
  syncAll: () =>
    http.post<CloudSyncResult>('/system/cloud/sync-all', undefined, { timeout: 1200000 }),
  deleteImages: (keys: string[]) =>
    http.post<{ success: number; failed: string[] }>(
      '/system/cloud/delete',
      { keys },
      { timeout: 600000 },
    ),
  clearImages: () =>
    http.post<{ success: number; total: number }>('/system/cloud/clear', undefined, {
      timeout: 1200000,
    }),
  listBackups: () => http.get<CloudBackup[]>('/system/cloud/backups', { timeout: 60000 }),
  uploadBackup: (password: string) =>
    http.post<{ key: string; name: string; size: number; url: string }>(
      '/system/cloud/backup',
      { password },
      { timeout: 300000 },
    ),
  restoreBackup: (key: string, password: string) =>
    http.post<RestoreStats>('/system/cloud/backup/restore', { key, password }, { timeout: 600000 }),
  downloadBackup: (key: string): Promise<Blob> =>
    rawHttp
      .get('/system/cloud/backup/download', {
        params: { key },
        responseType: 'blob',
        timeout: 600000,
      })
      .then((response) => response.data),
  deleteBackups: (keys: string[]) =>
    http.post('/system/cloud/backup/delete', { keys }, { timeout: 300000 }),
}

export const systemApi = {
  site: () => http.get<SiteInfo>('/panel/site'),
  monitor: () => http.get<MonitorData | null>('/system/monitor'),
  publicMonitor: () => http.get<MonitorData | null>('/panel/monitor'),
  favicon: (url: string) => http.get<FileRecord>('/system/favicon', { params: { url } }),
}
