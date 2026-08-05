import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import ChangePasswordModal from '../account/ChangePasswordModal';

export default function Navbar() {
  const { user, logout } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [changePasswordOpen, setChangePasswordOpen] = useState(false);
  const isAdmin = user?.role === 'admin';

  const openChangePassword = () => {
    setUserMenuOpen(false);
    setMobileOpen(false);
    setChangePasswordOpen(true);
  };

  return (
    <>
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
            <div className="hidden sm:block relative">
              <button
                onClick={() => setUserMenuOpen(!userMenuOpen)}
                className="flex items-center gap-1 text-sm text-gray-600 hover:text-gray-900"
              >
                {user?.name}
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 9l-7 7-7-7"
                  />
                </svg>
              </button>
              {userMenuOpen && (
                <>
                  <div className="fixed inset-0 z-10" onClick={() => setUserMenuOpen(false)} />
                  <div className="absolute right-0 top-full mt-2 w-48 z-20 bg-white border border-gray-200 rounded-md shadow-lg py-1">
                    <button
                      onClick={openChangePassword}
                      className="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                    >
                      Change Password
                    </button>
                    <button
                      onClick={logout}
                      className="block w-full text-left px-4 py-2 text-sm text-red-600 hover:bg-gray-50"
                    >
                      Logout
                    </button>
                  </div>
                </>
              )}
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
            <div className="pt-2 border-t border-gray-100">
              <div className="py-2 text-sm text-gray-600">{user?.name}</div>
              <button
                onClick={openChangePassword}
                className="block w-full text-left py-2 text-sm text-gray-600"
              >
                Change Password
              </button>
              <button onClick={logout} className="block w-full text-left py-2 text-sm text-red-600">
                Logout
              </button>
            </div>
          </div>
        )}
      </nav>

      <ChangePasswordModal
        isOpen={changePasswordOpen}
        onClose={() => setChangePasswordOpen(false)}
      />
    </>
  );
}
