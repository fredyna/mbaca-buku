import axios from 'axios';

const client = axios.create({
  baseURL: '/api',
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// A 401 from these endpoints means the sign-in attempt itself failed, not that
// a session expired. Reloading the page there would throw away the error the
// form is about to show — and for /auth/oauth it would bounce the user back to
// the login page mid-handshake, which is the very thing OAuth is meant to end.
const SIGN_IN_ENDPOINTS = ['/auth/login', '/auth/register', '/auth/oauth'];

client.interceptors.response.use(
  (response) => response,
  (error) => {
    const url = error.config?.url ?? '';
    const isSignIn = SIGN_IN_ENDPOINTS.some((path) => url.startsWith(path));
    if (error.response?.status === 401 && !isSignIn) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default client;
