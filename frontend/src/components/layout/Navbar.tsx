import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';

export default function Navbar() {
  const { user, logout } = useAuth();

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
            </div>
          </div>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">{user?.name}</span>
            <button
              onClick={logout}
              className="text-sm text-red-600 hover:text-red-800"
            >
              Logout
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
}
