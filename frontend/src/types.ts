export interface UserResponse {
  id: string
  email: string
  phone?: string
  displayName?: string
  bio?: string
  avatarFileId?: string
  createdAt: string
  updatedAt: string
}

export interface CategoryResponse {
  id: string
  name: string
  description?: string
  status: 'active' | 'hidden'
  createdAt: string
}

export interface ProductResponse {
  id: string
  name: string
  description?: string
  price: number
  status: 'available' | 'out_of_stock' | 'discontinued'
  categoryId: string
  categoryName?: string
  createdAt: string
}

export interface FileResponse {
  id: string
  userId: string
  originalName: string
  size: number
  mimetype: string
  createdAt: string
  updatedAt: string
}

export interface PaginatedResponse<T> {
  data: T[]
  meta: { page: number; limit: number; total: number; totalPages: number }
}

export interface ToastMsg {
  text: string
  type: 'success' | 'error'
}

export type Section = 'auth' | 'profile' | 'categories' | 'products' | 'files' | 'health'
