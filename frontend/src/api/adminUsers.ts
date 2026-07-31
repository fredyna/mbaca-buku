import client from './client';

export interface AdminUser {
  id: string;
  name: string;
  email: string;
  role: 'user' | 'admin';
  created_at: string;
  updated_at: string;
}

export interface AdminUserListResponse {
  success: boolean;
  data: AdminUser[];
  meta: { page: number; per_page: number; total: number };
}

export interface CreateUserPayload {
  name: string;
  email: string;
  password: string;
  role: 'user' | 'admin';
}

export interface UpdateUserPayload {
  name: string;
  email: string;
  role: 'user' | 'admin';
}

export const adminUsersApi = {
  list: async (page = 1, perPage = 20) => {
    const res = await client.get<AdminUserListResponse>(`/admin/users?page=${page}&per_page=${perPage}`);
    return { users: res.data.data || [], meta: res.data.meta };
  },

  create: async (payload: CreateUserPayload) => {
    const res = await client.post<{ success: boolean; data: AdminUser }>('/admin/users', payload);
    return res.data.data;
  },

  update: async (id: string, payload: UpdateUserPayload) => {
    const res = await client.put<{ success: boolean; data: AdminUser }>(`/admin/users/${id}`, payload);
    return res.data.data;
  },

  resetPassword: async (id: string, password: string) => {
    await client.put(`/admin/users/${id}/password`, { password });
  },

  delete: async (id: string) => {
    await client.delete(`/admin/users/${id}`);
  },
};
