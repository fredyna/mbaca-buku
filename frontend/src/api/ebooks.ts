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

export type EbookVisibility = 'all' | 'public' | 'private';
export type EbookSort = 'title' | 'latest';

export interface EbookListParams {
  page?: number;
  perPage?: number;
  q?: string;
  author?: string;
  visibility?: EbookVisibility;
  sort?: EbookSort;
}

/** The ebook grid fits eight cards per page; the API defaults to the same. */
export const EBOOKS_PER_PAGE = 8;

export const ebooksApi = {
  list: async (params: EbookListParams = {}) => {
    const search = new URLSearchParams({
      page: String(params.page ?? 1),
      per_page: String(params.perPage ?? EBOOKS_PER_PAGE),
    });
    if (params.q) search.set('q', params.q);
    if (params.author) search.set('author', params.author);
    if (params.visibility && params.visibility !== 'all') search.set('visibility', params.visibility);
    if (params.sort) search.set('sort', params.sort);

    const res = await client.get<EbookListResponse>(`/ebooks?${search.toString()}`);
    return { ebooks: res.data.data || [], meta: res.data.meta };
  },

  // Authors across every ebook the user may see, for the list page filter.
  listAuthors: async () => {
    const res = await client.get<{ success: boolean; data: string[] }>('/ebooks/authors');
    return res.data.data || [];
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
