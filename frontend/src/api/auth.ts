import client from './client';
import supabase from './supabaseClient';

export interface User {
  id: string;
  name: string;
  email: string;
  role: string;
}

export interface AuthResponse {
  user: User;
  token: string;
}

// Surfaces the API's own wording when it has some — "this account signs in with
// Google" is worth showing verbatim, since the generic fallback would send its
// owner to reset a password that does not exist.
export function authErrorMessage(err: unknown, fallback: string): string {
  const message = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
  return message || fallback;
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

  changePassword: async (oldPassword: string, newPassword: string) => {
    await client.put('/auth/password', { old_password: oldPassword, new_password: newPassword });
  },

  // Trades the Supabase access token for this API's own JWT. The identity is
  // read back from Supabase server-side, so nothing about the user is sent
  // from here — only proof of the session.
  oauth: async (accessToken: string) => {
    const res = await client.post<{ success: boolean; data: AuthResponse }>('/auth/oauth', {
      access_token: accessToken,
    });
    return res.data.data;
  },

  signInWithGoogle: async () => {
    if (!supabase) throw new Error('Google sign-in is not configured for this deployment');
    const { data, error } = await supabase.auth.signInWithOAuth({
      provider: 'google',
      // Come back to this app rather than whatever Site URL the Supabase
      // project happens to name, so local and deployed builds both land home.
      options: { redirectTo: window.location.origin },
    });
    if (error) throw error;
    return data;
  },

  // Returns the Supabase access token of the current session, if any. Present
  // right after the Google redirect, before the app has a JWT of its own.
  supabaseAccessToken: async () => {
    if (!supabase) return null;
    const { data } = await supabase.auth.getSession();
    return data.session?.access_token ?? null;
  },

  signOutSupabase: async () => {
    await supabase?.auth.signOut();
  },
};
