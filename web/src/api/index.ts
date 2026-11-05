import http from './request'
import type {
  BackupPayload,
  FileRecord,
  ItemGroup,
  ItemIcon,
  PanelConfig,
  PanelData,
  SearchEngine,
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
}

export const fileApi = {
  list: (fileType?: number) =>
    http.get<FileRecord[]>('/system/file/list', { params: fileType ? { fileType } : {} }),
  upload: (file: File, fileType: 1 | 2) => {
    const form = new FormData()
    form.append('file', file)
    form.append('fileType', String(fileType))
    return http.post<FileRecord>('/system/file/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
  remove: (id: number) => http.delete(`/system/file/${id}`),
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
