import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Ebook } from '../api/ebooks';
import { ebooksApi } from '../api/ebooks';
import { historyApi } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EbookUpload from '../components/ebook/EbookUpload';
import EmptyState from '../components/common/EmptyState';

export default function EbookListPage() {
  const [ebooks, setEbooks] = useState<Ebook[]>([]);
  const [showUpload, setShowUpload] = useState(false);
  const navigate = useNavigate();

  const loadEbooks = () => {
    ebooksApi.list(1, 50).then(({ ebooks }) => setEbooks(ebooks));
  };

  useEffect(() => {
    loadEbooks();
  }, []);

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this ebook?')) return;
    await ebooksApi.delete(id);
    loadEbooks();
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Ebooks</h1>
        <button
          onClick={() => setShowUpload(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
        >
          Upload Ebook
        </button>
      </div>

      {ebooks.length === 0 ? (
        <EmptyState title="No ebooks yet" description="Upload your first PDF ebook to get started" />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {ebooks.map((ebook) => (
            <EbookCard key={ebook.id} ebook={ebook} onRead={handleRead} onDelete={handleDelete} />
          ))}
        </div>
      )}

      <EbookUpload isOpen={showUpload} onClose={() => setShowUpload(false)} onSuccess={loadEbooks} />
    </div>
  );
}
