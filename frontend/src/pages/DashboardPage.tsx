import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Ebook } from '../api/ebooks';
import { ebooksApi } from '../api/ebooks';
import type { HistoryItem } from '../api/history';
import { historyApi, historyItemToEbook } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EbookDetailModal from '../components/ebook/EbookDetailModal';
import EmptyState from '../components/common/EmptyState';
import ViewToggle from '../components/common/ViewToggle';
import type { ViewMode } from '../components/common/ViewToggle';
import { usePersistentState } from '../hooks/usePersistentState';

export default function DashboardPage() {
  const [recentEbooks, setRecentEbooks] = useState<Ebook[]>([]);
  const [reading, setReading] = useState<HistoryItem[]>([]);
  const [detail, setDetail] = useState<Ebook | null>(null);
  const [viewMode, setViewMode] = usePersistentState<ViewMode>('dashboard:viewMode', 'grid');
  const navigate = useNavigate();

  useEffect(() => {
    // sort is explicit here: the list endpoint now defaults to title order.
    ebooksApi.list({ page: 1, perPage: 4, sort: 'latest' }).then(({ ebooks }) => setRecentEbooks(ebooks));
    historyApi.getHistory().then((data) => setReading(data.reading || []));
  }, []);

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  const containerClass =
    viewMode === 'grid'
      ? 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6'
      : 'flex flex-col gap-3';

  return (
    <div className="space-y-10">
      <div className="flex justify-end">
        <ViewToggle value={viewMode} onChange={setViewMode} />
      </div>

      <section>
        <h2 className="text-xl font-bold text-gray-900 mb-4">Continue Reading</h2>
        {reading.length === 0 ? (
          <EmptyState title="No books in progress" description="Start reading a book to see it here" />
        ) : (
          <div className={containerClass}>
            {reading.map((item) => (
              <EbookCard
                key={item.ebook_id}
                ebook={historyItemToEbook(item)}
                onRead={handleRead}
                onShowDetail={setDetail}
                progress={{ last_page: item.last_page, total_pages: item.total_pages }}
                viewMode={viewMode}
              />
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="text-xl font-bold text-gray-900 mb-4">Recently Added</h2>
        {recentEbooks.length === 0 ? (
          <EmptyState title="No ebooks yet" description="Upload your first ebook to get started" />
        ) : (
          <div className={containerClass}>
            {recentEbooks.map((ebook) => (
              <EbookCard
                key={ebook.id}
                ebook={ebook}
                onRead={handleRead}
                onShowDetail={setDetail}
                viewMode={viewMode}
              />
            ))}
          </div>
        )}
      </section>

      <EbookDetailModal ebook={detail} onClose={() => setDetail(null)} />
    </div>
  );
}
