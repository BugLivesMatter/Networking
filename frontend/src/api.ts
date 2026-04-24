import type { UserResponse, CategoryResponse, ProductResponse, FileResponse, PaginatedResponse } from './types'

async function req<T>(url: string, options: RequestInit = {}): Promise<T> {
  const isFormData = options.body instanceof FormData
  const headers: Record<string, string> = isFormData ? {} : { 'Content-Type': 'application/json' }
  const res = await fetch(url, { ...options, credentials: 'include', headers: { ...headers, ...options.headers as Record<string, string> } })
  if (res.status === 204) return null as T
  const data = await res.json().catch(() => ({ error: res.statusText }))
  if (!res.ok) throw new Error(data.error || JSON.stringify(data))
  return data as T
}

export const api = {
  auth: {
    login: (email: string, password: string) =>
      req<{ message: string }>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
    register: (email: string, password: string, phone?: string) =>
      req<{ userId: string; message: string }>('/auth/register', { method: 'POST', body: JSON.stringify({ email, password, ...(phone ? { phone } : {}) }) }),
    logout: () => req<{ message: string }>('/auth/logout', { method: 'POST', body: '{}' }),
    logoutAll: () => req<{ message: string }>('/auth/logout-all', { method: 'POST', body: '{}' }),
    whoami: () => req<UserResponse>('/auth/whoami'),
    refresh: () => req<{ message: string }>('/auth/refresh', { method: 'POST', body: '{}' }),
  },
  profile: {
    get: () => req<UserResponse>('/profile'),
    update: (data: { displayName?: string; bio?: string; avatarFileId?: string }) =>
      req<UserResponse>('/profile', { method: 'POST', body: JSON.stringify(data) }),
  },
  categories: {
    list: (page = 1, limit = 10) => req<PaginatedResponse<CategoryResponse>>(`/categories?page=${page}&limit=${limit}`),
    get: (id: string) => req<CategoryResponse>(`/categories/${id}`),
    create: (data: { name: string; description?: string; status?: string }) =>
      req<CategoryResponse>('/categories', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: object) =>
      req<CategoryResponse>(`/categories/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    patch: (id: string, data: object) =>
      req<CategoryResponse>(`/categories/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) => req<null>(`/categories/${id}`, { method: 'DELETE' }),
  },
  products: {
    list: (page = 1, limit = 10, categoryId?: string) =>
      req<PaginatedResponse<ProductResponse>>(`/products?page=${page}&limit=${limit}${categoryId ? `&category_id=${categoryId}` : ''}`),
    get: (id: string) => req<ProductResponse>(`/products/${id}`),
    create: (data: object) => req<ProductResponse>('/products', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: object) =>
      req<ProductResponse>(`/products/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    patch: (id: string, data: object) =>
      req<ProductResponse>(`/products/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) => req<null>(`/products/${id}`, { method: 'DELETE' }),
  },
  files: {
    upload: (file: File) => {
      const fd = new FormData()
      fd.append('file', file)
      return req<{ file: FileResponse }>('/files', { method: 'POST', body: fd })
    },
    download: async (id: string, filename: string) => {
      const res = await fetch(`/files/${id}`, { credentials: 'include' })
      if (!res.ok) throw new Error('Download failed')
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url; a.download = filename; a.click()
      URL.revokeObjectURL(url)
    },
    delete: (id: string) => req<null>(`/files/${id}`, { method: 'DELETE' }),
  },
  health: {
    redis: () => req<unknown>('/health/redis'),
    diagnosis: (page = 1, limit = 5) => req<unknown>(`/health/diagnosis?page=${page}&limit=${limit}`),
  },
}
