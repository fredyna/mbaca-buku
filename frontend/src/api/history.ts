import client from './client';
import type { Ebook } from './ebooks';

export interface HistoryItem {
  ebook_id: string;
  title: string;
  author: string;
  cover_url: string;
  total_pages: number;
  last_page: number;
  status: string;
  is_private: boolean;
  uploaded_by: string;
  uploaded_by_name: string;
  created_at: string;
  last_opened: string;
}

export interface HistoryResponse {
  reading: HistoryItem[];
  completed: HistoryItem[];
}

export interface HistoryListMeta {
  page: number;
  per_page: number;
  total: number;
}

export interface HistoryListResponse {
  items: HistoryItem[];
  meta: HistoryListMeta;
}

export function historyItemToEbook(item: HistoryItem): Ebook {
  return {
    id: item.ebook_id,
    title: item.title,
    author: item.author,
    cover_url: item.cover_url,
    total_pages: item.total_pages,
    file_size: 0,
    uploaded_by: item.uploaded_by,
    uploaded_by_name: item.uploaded_by_name,
    is_private: item.is_private,
    created_at: item.created_at,
  };
}

export const historyApi = {
  getHistory: async () => {
    const res = await client.get<{ success: boolean; data: HistoryResponse }>('/history');
    return res.data.data;
  },

  list: async (status: 'reading' | 'completed', page: number, perPage: number) => {
    const res = await client.get<{
      success: boolean;
      data: HistoryItem[];
      meta: HistoryListMeta;
    }>(`/history?status=${status}&page=${page}&per_page=${perPage}`);
    return { items: res.data.data || [], meta: res.data.meta };
  },

  openBook: async (ebookId: string) => {
    const res = await client.post<{
      success: boolean;
      data: { ebook_id: string; last_page: number; status: string };
    }>(`/ebooks/${ebookId}/open`);
    return res.data.data;
  },
};
