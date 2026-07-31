import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';

export default function Navbar() {
  const { user, logout } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);
  const isAdmin = user?.role === 'admin';

  return (
    <nav className="bg-white border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16 items-center">
          <div className="flex items-center gap-8">
            <Link to="/" className="text-xl font-bold text-gray-900">
              Mbaca Buku
            </Link>
            <div className="hidden sm:flex gap-6">
              <Link to="/" className="text-gray-600 hover:text-gray-900">Dashboard</Link>
              <Link to="/ebooks" className="text-gray-600 hover:text-gray-900">Ebooks</Link>
              <Link to="/history" className="text-gray-600 hover:text-gray-900">History</Link>
              {isAdmin && (
                <Link to="/users" className="text-gray-600 hover:text-gray-900">Users</Link>
              )}
            </div>
          </div>
          <div className="hidden sm:flex items-center gap-4">
            <span className="text-sm text-gray-600">{user?.name}</span>
            <button
              onClick={logout}
              className="text-sm text-red-600 hover:text-red-800"
            >
              Logout
            </button>
          </div>
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="sm:hidden p-2 text-gray-600"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d={mobileOpen ? 'M6 18L18 6M6 6l12 12' : 'M4 6h16M4 12h16M4 18h16'}
              />
            </svg>
          </button>
        </div>
      </div>

      {mobileOpen && (
        <div className="sm:hidden border-t border-gray-200 bg-white px-4 py-3 space-y-2">
          <Link
            to="/"
            onClick={() => setMobileOpen(false)}
            className="block py-2 text-gray-600"
          >
            Dashboard
          </Link>
          <Link
            to="/ebooks"
            onClick={() => setMobileOpen(false)}
            className="block py-2 text-gray-600"
          >
            Ebooks
          </Link>
          <Link
            to="/history"
            onClick={() => setMobileOpen(false)}
            className="block py-2 text-gray-600"
          >
            History
          </Link>
          {isAdmin && (
            <Link
              to="/users"
              onClick={() => setMobileOpen(false)}
              className="block py-2 text-gray-600"
            >
              Users
            </Link>
          )}
          <div className="pt-2 border-t border-gray-100 flex justify-between items-center">
            <span className="text-sm text-gray-600">{user?.name}</span>
            <button onClick={logout} className="text-sm text-red-600">
              Logout
            </button>
          </div>
        </div>
      )}
    </nav>
  );
}
