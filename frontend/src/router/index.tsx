import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import Layout from '../components/layout/Layout';
import LoginPage from '../pages/LoginPage';
import RegisterPage from '../pages/RegisterPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-500">Loading...</div>
      </div>
    );
  }

  if (!user) return <Navigate to="/login" />;
  return <>{children}</>;
}

function PlaceholderPage({ title }: { title: string }) {
  return <h1 className="text-2xl font-bold">{title}</h1>;
}

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route path="/" element={<PlaceholderPage title="Dashboard" />} />
          <Route path="/ebooks" element={<PlaceholderPage title="Ebooks" />} />
          <Route path="/read/:id" element={<PlaceholderPage title="Reader" />} />
          <Route path="/history" element={<PlaceholderPage title="History" />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
