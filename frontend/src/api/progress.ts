import client from './client';

export const progressApi = {
  get: async (ebookId: string) => {
    const res = await client.get<{ success: boolean; data: { last_page: number; status: string } }>(
      `/ebooks/${ebookId}/progress`
    );
    return res.data.data;
  },

  update: async (ebookId: string, page: number) => {
    await client.put(`/ebooks/${ebookId}/progress`, { page });
  },

  setStatus: async (ebookId: string, status: 'reading' | 'completed') => {
    await client.put(`/ebooks/${ebookId}/status`, { status });
  },
};
