import { Outlet } from 'react-router-dom';
import Navbar from './Navbar';
import Footer from './Footer';

export default function Layout() {
  return (
    <div className="min-h-screen bg-gray-50 flex flex-col">
      <Navbar />
      {/* min-w-0: as a flex item, main would otherwise refuse to shrink below
          its content's min-content width, so one too-wide child stretches the
          whole page and the phone gets a sideways scrollbar. */}
      <main className="flex-1 w-full min-w-0 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>
      <Footer />
    </div>
  );
}
