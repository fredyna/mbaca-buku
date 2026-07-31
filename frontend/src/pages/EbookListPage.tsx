import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Ebook } from '../api/ebooks';
import { ebooksApi } from '../api/ebooks';
import { historyApi } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EbookUpload from '../components/ebook/EbookUpload';
import EbookEdit from '../components/ebook/EbookEdit';
import EbookDetailModal from '../components/ebook/EbookDetailModal';
import EmptyState from '../components/common/EmptyState';
import ViewToggle from '../components/common/ViewToggle';
import type { ViewMode } from '../components/common/ViewToggle';
import { usePersistentState } from '../hooks/usePersistentState';
import { useAuth } from '../context/AuthContext';

export default function EbookListPage() {
  const [ebooks, setEbooks] = useState<Ebook[]>([]);
  const [showUpload, setShowUpload] = useState(false);
  const [editing, setEditing] = useState<Ebook | null>(null);
  const [detail, setDetail] = useState<Ebook | null>(null);
  const [viewMode, setViewMode] = usePersistentState<ViewMode>('ebooks:viewMode', 'grid');
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const canManage = (e: Ebook) => isAdmin || e.uploaded_by === user?.id;

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

  const cardProps = (ebook: Ebook) => ({
    ebook,
    onRead: handleRead,
    onEdit: canManage(ebook) ? setEditing : undefined,
    onDelete: canManage(ebook) ? handleDelete : undefined,
    onShowDetail: setDetail,
  });

  return (
    <div>
      <div className="flex items-center justify-between mb-6 gap-3">
        <h1 className="text-2xl font-bold text-gray-900">Ebooks</h1>
        <div className="flex items-center gap-3">
          <ViewToggle value={viewMode} onChange={setViewMode} />
          <button
            onClick={() => setShowUpload(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
          >
            Upload Ebook
          </button>
        </div>
      </div>

      {ebooks.length === 0 ? (
        <EmptyState
          title="No ebooks yet"
          description="Upload your first PDF ebook to get started"
        />
      ) : viewMode === 'grid' ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {ebooks.map((ebook) => (
            <EbookCard key={ebook.id} {...cardProps(ebook)} viewMode="grid" />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {ebooks.map((ebook) => (
            <EbookCard key={ebook.id} {...cardProps(ebook)} viewMode="list" />
          ))}
        </div>
      )}

      <EbookUpload isOpen={showUpload} onClose={() => setShowUpload(false)} onSuccess={loadEbooks} />
      <EbookEdit
        isOpen={!!editing}
        ebook={editing}
        onClose={() => setEditing(null)}
        onSuccess={loadEbooks}
      />
      <EbookDetailModal ebook={detail} onClose={() => setDetail(null)} />
    </div>
  );
}
