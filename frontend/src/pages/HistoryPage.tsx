import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Ebook } from '../api/ebooks';
import type { HistoryItem } from '../api/history';
import { historyApi, historyItemToEbook } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EbookDetailModal from '../components/ebook/EbookDetailModal';
import EmptyState from '../components/common/EmptyState';
import ViewToggle from '../components/common/ViewToggle';
import type { ViewMode } from '../components/common/ViewToggle';
import { usePersistentState } from '../hooks/usePersistentState';

export default function HistoryPage() {
  const [reading, setReading] = useState<HistoryItem[]>([]);
  const [completed, setCompleted] = useState<HistoryItem[]>([]);
  const [tab, setTab] = useState<'reading' | 'completed'>('reading');
  const [detail, setDetail] = useState<Ebook | null>(null);
  const [viewMode, setViewMode] = usePersistentState<ViewMode>('history:viewMode', 'grid');
  const navigate = useNavigate();

  useEffect(() => {
    historyApi.getHistory().then((data) => {
      setReading(data.reading || []);
      setCompleted(data.completed || []);
    });
  }, []);

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  const items = tab === 'reading' ? reading : completed;

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
            In Progress ({reading.length})
          </button>
          <button
            onClick={() => setTab('completed')}
            className={`px-4 py-2 text-sm rounded-md ${
              tab === 'completed' ? 'bg-white shadow text-gray-900' : 'text-gray-600'
            }`}
          >
            Completed ({completed.length})
          </button>
        </div>
        <ViewToggle value={viewMode} onChange={setViewMode} />
      </div>

      {items.length === 0 ? (
        <EmptyState
          title={tab === 'reading' ? 'No books in progress' : 'No completed books'}
          description={
            tab === 'reading'
              ? 'Start reading a book to see it here'
              : 'Finish reading a book to see it here'
          }
        />
      ) : (
        <div className={containerClass}>
          {items.map((item) => {
            // A completed book still sitting on its last page has not been
            // re-read yet, so a full bar would overstate what happened.
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
      )}

      <EbookDetailModal ebook={detail} onClose={() => setDetail(null)} />
    </div>
  );
}
