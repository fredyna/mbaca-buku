import client from './client';

export interface Ebook {
  id: string;
  title: string;
  author: string;
  cover_url: string;
  file_size: number;
  total_pages: number;
  uploaded_by: string;
  uploaded_by_name: string;
  is_private: boolean;
  created_at: string;
}

export interface EbookListResponse {
  success: boolean;
  data: Ebook[];
  meta: { page: number; per_page: number; total: number };
}

export const ebooksApi = {
  list: async (page = 1, perPage = 20) => {
    const res = await client.get<EbookListResponse>(`/ebooks?page=${page}&per_page=${perPage}`);
    return { ebooks: res.data.data || [], meta: res.data.meta };
  },

  getById: async (id: string) => {
    const res = await client.get<{ success: boolean; data: Ebook }>(`/ebooks/${id}`);
    return res.data.data;
  },

  upload: async (formData: FormData) => {
    const res = await client.post<{ success: boolean; data: Ebook }>('/ebooks', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return res.data.data;
  },

  update: async (id: string, data: { title: string; author: string; is_private: boolean }) => {
    const res = await client.put<{ success: boolean; data: Ebook }>(`/ebooks/${id}`, data);
    return res.data.data;
  },

  delete: async (id: string) => {
    await client.delete(`/ebooks/${id}`);
  },

  getFileUrl: async (id: string) => {
    const res = await client.get<{ success: boolean; data: { url: string } }>(`/ebooks/${id}/file`);
    return res.data.data.url;
  },
};
