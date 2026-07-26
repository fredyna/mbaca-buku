import client from './client';

export interface User {
  id: string;
  name: string;
  email: string;
}

export interface AuthResponse {
  user: User;
  token: string;
}

export const authApi = {
  login: async (email: string, password: string) => {
    const res = await client.post<{ success: boolean; data: AuthResponse }>('/auth/login', { email, password });
    return res.data.data;
  },

  register: async (name: string, email: string, password: string) => {
    const res = await client.post<{ success: boolean; data: AuthResponse }>('/auth/register', { name, email, password });
    return res.data.data;
  },

  me: async () => {
    const res = await client.get<{ success: boolean; data: User }>('/auth/me');
    return res.data.data;
  },
};
