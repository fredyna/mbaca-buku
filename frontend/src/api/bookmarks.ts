import client from './client';

export interface Bookmark {
  id: string;
  ebook_id: string;
  page_number: number;
  note: string;
  created_at: string;
}

export const bookmarksApi = {
  list: async (ebookId: string) => {
    const res = await client.get<{ success: boolean; data: Bookmark[] }>(`/ebooks/${ebookId}/bookmarks`);
    return res.data.data;
  },

  create: async (ebookId: string, pageNumber: number, note = '') => {
    const res = await client.post<{ success: boolean; data: Bookmark }>(`/ebooks/${ebookId}/bookmarks`, {
      page_number: pageNumber,
      note,
    });
    return res.data.data;
  },

  delete: async (id: string) => {
    await client.delete(`/bookmarks/${id}`);
  },
};
