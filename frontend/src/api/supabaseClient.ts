import { createClient } from '@supabase/supabase-js';
import type { SupabaseClient } from '@supabase/supabase-js';

const SUPABASE_URL = import.meta.env.VITE_SUPABASE_URL || '';
const SUPABASE_KEY = import.meta.env.VITE_SUPABASE_ANON_KEY || '';

// Google sign-in is optional: a deployment may ship password login only. Build
// the client only when both values were baked into the bundle — createClient
// throws on a blank URL, and at module scope that throw is a white page rather
// than a missing button.
export const isSupabaseConfigured = Boolean(SUPABASE_URL && SUPABASE_KEY);

export const supabase: SupabaseClient | null = isSupabaseConfigured
  ? createClient(SUPABASE_URL, SUPABASE_KEY)
  : null;

export default supabase;
