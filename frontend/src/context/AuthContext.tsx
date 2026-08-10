import { createContext, useContext, useState, useEffect, useRef, useCallback } from 'react';
import type { ReactNode } from 'react';
import { authApi } from '../api/auth';
import type { AuthResponse, User } from '../api/auth';
import supabase from '../api/supabaseClient';

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  loginWithGoogle: () => Promise<void>;
  logout: () => void;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);

  // Stays true until we know whether this browser is signed in — including the
  // round trip that turns a Supabase session into one of our own tokens.
  // Releasing it early is what used to bounce a successful Google sign-in back
  // to the login page: ProtectedRoute saw isLoading false with no user yet and
  // navigated away while the exchange was still in flight.
  const [isLoading, setIsLoading] = useState(true);

  // Google's redirect can be noticed twice — once by the boot check below and
  // once by onAuthStateChange — so remember the exchange in progress and let
  // both waiters share its result instead of racing two /auth/oauth calls.
  const exchangeRef = useRef<Promise<void> | null>(null);
  const tokenRef = useRef<string | null>(null);

  const adopt = useCallback((data: AuthResponse) => {
    localStorage.setItem('token', data.token);
    tokenRef.current = data.token;
    setToken(data.token);
    setUser(data.user);
  }, []);

  const clear = useCallback(() => {
    localStorage.removeItem('token');
    tokenRef.current = null;
    setToken(null);
    setUser(null);
  }, []);

  const exchangeSupabaseSession = useCallback(
    (accessToken: string) => {
      if (exchangeRef.current) return exchangeRef.current;

      const pending = authApi
        .oauth(accessToken)
        .then(adopt)
        .finally(() => {
          exchangeRef.current = null;
        });

      exchangeRef.current = pending;
      return pending;
    },
    [adopt],
  );

  // One boot sequence, run once: prefer a token we already hold, otherwise pick
  // up the Supabase session Google just handed us. Whichever path is taken, the
  // app has its own JWT and the matching users row before anything renders.
  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const stored = localStorage.getItem('token');
        if (stored) {
          try {
            const me = await authApi.me();
            if (!cancelled) {
              tokenRef.current = stored;
              setToken(stored);
              setUser(me);
            }
            return;
          } catch {
            localStorage.removeItem('token');
          }
        }

        const accessToken = await authApi.supabaseAccessToken();
        if (accessToken && !cancelled) {
          await exchangeSupabaseSession(accessToken);
        }
      } catch {
        if (!cancelled) clear();
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [exchangeSupabaseSession, clear]);

  // Covers a sign-in that completes after boot — a popup flow, or a Supabase
  // token refreshed in another tab.
  useEffect(() => {
    if (!supabase) return;

    const { data } = supabase.auth.onAuthStateChange((event, session) => {
      if (event !== 'SIGNED_IN' && event !== 'TOKEN_REFRESHED') return;
      if (!session?.access_token || tokenRef.current) return;

      exchangeSupabaseSession(session.access_token).catch(() => {
        /* the login page reports the failure; nothing to recover here */
      });
    });

    return () => data.subscription.unsubscribe();
  }, [exchangeSupabaseSession]);

  const login = async (email: string, password: string) => {
    adopt(await authApi.login(email, password));
  };

  const register = async (name: string, email: string, password: string) => {
    adopt(await authApi.register(name, email, password));
  };

  const loginWithGoogle = async () => {
    // Hands the browser to Google. Nothing after this runs on the current page:
    // the flow resumes in the boot effect above once Supabase redirects back.
    await authApi.signInWithGoogle();
  };

  const logout = () => {
    clear();
    // Without this the Supabase session outlives the logout, and the next boot
    // would sign the user straight back in.
    void authApi.signOutSupabase();
  };

  return (
    <AuthContext.Provider value={{ user, token, login, register, logout, isLoading, loginWithGoogle }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
