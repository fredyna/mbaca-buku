import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import type { Ebook } from '../api/ebooks';
import type { HistoryItem } from '../api/history';
import { historyApi, historyItemToEbook } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EbookDetailModal from '../components/ebook/EbookDetailModal';
import EmptyState from '../components/common/EmptyState';
import Pagination from '../components/common/Pagination';
import ViewToggle from '../components/common/ViewToggle';
import type { ViewMode } from '../components/common/ViewToggle';
import { usePersistentState } from '../hooks/usePersistentState';

const HISTORY_PER_PAGE = 8;

export default function HistoryPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const tabParam = searchParams.get('tab');
  const page = Math.max(1, Number(searchParams.get('page')) || 1);
  const tab = tabParam === 'completed' ? 'completed' : 'reading';

  const [items, setItems] = useState<HistoryItem[]>([]);
  const [total, setTotal] = useState(0);
  const [detail, setDetail] = useState<Ebook | null>(null);
  const [viewMode, setViewMode] = usePersistentState<ViewMode>('history:viewMode', 'grid');
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState('');
  const navigate = useNavigate();

  const setTab = (nextTab: 'reading' | 'completed') => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set('tab', nextTab);
      next.delete('page');
      return next;
    });
  };

  const goToPage = (target: number, replace = false) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (target <= 1) next.delete('page');
        else next.set('page', String(target));
        return next;
      },
      { replace }
    );
  };

  const loadHistory = useCallback(() => {
    setLoading(true);
    historyApi
      .list(tab, page, HISTORY_PER_PAGE)
      .then(({ items, meta }) => {
        setItems(items);
        setTotal(meta.total);
        setErrorMsg('');
      })
      .catch(() => {
        setItems([]);
        setTotal(0);
        setErrorMsg('Failed to load history');
      })
      .finally(() => setLoading(false));
  }, [page, tab]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  useEffect(() => {
    const totalPages = Math.max(1, Math.ceil(total / HISTORY_PER_PAGE));
    if (!loading && page > totalPages) {
      goToPage(totalPages, true);
    }
  }, [loading, page, total]);

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  const containerClass =
    viewMode === 'grid'
      ? 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6'
      : 'flex flex-col gap-3';

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Reading History</h1>

      <div className="flex items-center justify-between gap-3 mb-6">
        <div className="flex gap-1 bg-gray-100 p-1 rounded-lg w-fit">
          <button
            onClick={() => setTab('reading')}
            className={`px-4 py-2 text-sm rounded-md ${
              tab === 'reading' ? 'bg-white shadow text-gray-900' : 'text-gray-600'
            }`}
          >
            In Progress
          </button>
          <button
            onClick={() => setTab('completed')}
            className={`px-4 py-2 text-sm rounded-md ${
              tab === 'completed' ? 'bg-white shadow text-gray-900' : 'text-gray-600'
            }`}
          >
            Completed
          </button>
        </div>
        <ViewToggle value={viewMode} onChange={setViewMode} />
      </div>

      {errorMsg && (
        <div className="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{errorMsg}</div>
      )}

      {items.length === 0 && !loading ? (
        <EmptyState
          title={tab === 'reading' ? 'No books in progress' : 'No completed books'}
          description={
            tab === 'reading'
              ? 'Start reading a book to see it here'
              : 'Finish reading a book to see it here'
          }
        />
      ) : (
        <>
          <div className={containerClass}>
            {items.map((item) => {
              const notReReadYet = tab === 'completed' && item.last_page >= item.total_pages;

              return (
                <EbookCard
                  key={item.ebook_id}
                  ebook={historyItemToEbook(item)}
                  onRead={handleRead}
                  onShowDetail={setDetail}
                  progress={
                    notReReadYet
                      ? undefined
                      : { last_page: item.last_page, total_pages: item.total_pages }
                  }
                  progressLabel={tab === 'completed' ? 'Re-read progress' : 'Progress'}
                  note={notReReadYet ? 'Finished' : undefined}
                  viewMode={viewMode}
                />
              );
            })}
          </div>

          <div className="mt-6">
            <Pagination
              page={page}
              perPage={HISTORY_PER_PAGE}
              total={total}
              onChange={goToPage}
            />
          </div>
        </>
      )}

      <EbookDetailModal ebook={detail} onClose={() => setDetail(null)} />
    </div>
  );
}
