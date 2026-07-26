import client from './client';

export interface HistoryItem {
  ebook_id: string;
  title: string;
  author: string;
  cover_url: string;
  total_pages: number;
  last_page: number;
  status: string;
  last_opened: string;
}

export interface HistoryResponse {
  reading: HistoryItem[];
  completed: HistoryItem[];
}

export const historyApi = {
  getHistory: async () => {
    const res = await client.get<{ success: boolean; data: HistoryResponse }>('/history');
    return res.data.data;
  },

  openBook: async (ebookId: string) => {
    const res = await client.post<{ success: boolean; data: { ebook_id: string; last_page: number } }>(
      `/ebooks/${ebookId}/open`
    );
    return res.data.data;
  },
};
