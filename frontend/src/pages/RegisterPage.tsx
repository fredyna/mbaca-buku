import { useState } from 'react';
import { Link, Navigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { authErrorMessage } from '../api/auth';

export default function RegisterPage() {
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { loginWithGoogle, user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-500">Signing you in...</div>
      </div>
    );
  }

  if (user) return <Navigate to="/" replace />;

  // The redirect back from Google is handled by AuthProvider, so the loading
  // flag is only cleared when the handoff never happened.
  const handleGoogle = async () => {
    setError('');
    setLoading(true);
    try {
      await loginWithGoogle();
    } catch (err) {
      setError(authErrorMessage(err, 'Google sign-in failed'));
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-md bg-white rounded-lg shadow p-8">
        <h1 className="text-2xl font-bold text-center mb-6">Mbaca Buku</h1>
        <p className="text-center text-gray-500 mb-8">Register with Google</p>

        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{error}</div>
        )}

        <div className="space-y-4">
          <button
            onClick={handleGoogle}
            disabled={loading}
            className="w-full py-2 bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50"
          >
            {loading ? 'Opening Google...' : 'Continue with Google'}
          </button>
        </div>

        <p className="mt-6 text-center text-sm text-gray-600">
          Already have an account?{' '}
          <Link to="/login" className="text-blue-600 hover:underline">Sign in</Link>
        </p>
        <p className="mt-8 text-center text-xs text-gray-400">
          © {new Date().getFullYear()} Created by Fredy Nur Apriyanto
        </p>
      </div>
    </div>
  );
}
