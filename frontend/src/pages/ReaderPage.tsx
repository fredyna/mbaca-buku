import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ebooksApi } from '../api/ebooks';
import type { Ebook } from '../api/ebooks';
import { historyApi } from '../api/history';
import { progressApi } from '../api/progress';
import PdfReader from '../components/reader/PdfReader';
import PageControls from '../components/reader/PageControls';
import ProgressBar from '../components/reader/ProgressBar';
import BookmarkButton from '../components/reader/BookmarkButton';
import { useDebouncedCallback } from '../hooks/useDebounce';
import { useBeforeUnload } from '../hooks/useBeforeUnload';

export default function ReaderPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [ebook, setEbook] = useState<Ebook | null>(null);
  const [fileUrl, setFileUrl] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(0);
  const [dualPage, setDualPage] = useState(window.innerWidth > 1024);
  const [loading, setLoading] = useState(true);

  const saveProgress = useDebouncedCallback((page: number) => {
    if (id) progressApi.update(id, page);
  }, 3000);

  useBeforeUnload(() => {
    if (id) {
      const data = JSON.stringify({ page: currentPage });
      navigator.sendBeacon(`/api/ebooks/${id}/progress`, new Blob([data], { type: 'application/json' }));
    }
  });

  useEffect(() => {
    if (!id) return;

    const init = async () => {
      try {
        const [ebookData, openData, url] = await Promise.all([
          ebooksApi.getById(id),
          historyApi.openBook(id),
          ebooksApi.getFileUrl(id),
        ]);
        setEbook(ebookData);
        setCurrentPage(openData.last_page);
        setFileUrl(url);
      } catch {
        navigate('/');
      } finally {
        setLoading(false);
      }
    };
    init();
  }, [id, navigate]);

  const handlePageChange = useCallback(
    (page: number) => {
      setCurrentPage(page);
      saveProgress(page);
    },
    [saveProgress]
  );

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-500">Loading book...</div>
      </div>
    );
  }

  if (!ebook || !fileUrl) return null;

  return (
    <div className="h-screen flex flex-col bg-gray-100">
      <div className="bg-white border-b border-gray-200 px-4 py-2 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button onClick={() => navigate(-1)} className="text-gray-600 hover:text-gray-900">
            &larr; Back
          </button>
          <h1 className="text-lg font-semibold text-gray-900 truncate max-w-md">{ebook.title}</h1>
        </div>
        <div className="flex items-center gap-4">
          <BookmarkButton ebookId={ebook.id} currentPage={currentPage} />
          <div className="w-40">
            <ProgressBar currentPage={currentPage} totalPages={totalPages || ebook.total_pages} />
          </div>
        </div>
      </div>

      <PdfReader
        fileUrl={fileUrl}
        currentPage={currentPage}
        dualPage={dualPage}
        onPageChange={handlePageChange}
        onDocumentLoad={setTotalPages}
      />

      <PageControls
        currentPage={currentPage}
        totalPages={totalPages || ebook.total_pages}
        onPageChange={handlePageChange}
        dualPage={dualPage}
        onToggleDualPage={() => setDualPage(!dualPage)}
      />
    </div>
  );
}
