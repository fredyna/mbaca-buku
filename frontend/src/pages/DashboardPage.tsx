import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Ebook } from '../api/ebooks';
import { ebooksApi } from '../api/ebooks';
import type { HistoryItem } from '../api/history';
import { historyApi } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EmptyState from '../components/common/EmptyState';

export default function DashboardPage() {
  const [recentEbooks, setRecentEbooks] = useState<Ebook[]>([]);
  const [reading, setReading] = useState<HistoryItem[]>([]);
  const navigate = useNavigate();

  useEffect(() => {
    ebooksApi.list(1, 4).then(({ ebooks }) => setRecentEbooks(ebooks));
    historyApi.getHistory().then((data) => setReading(data.reading || []));
  }, []);

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  return (
    <div className="space-y-10">
      <section>
        <h2 className="text-xl font-bold text-gray-900 mb-4">Continue Reading</h2>
        {reading.length === 0 ? (
          <EmptyState title="No books in progress" description="Start reading a book to see it here" />
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {reading.map((item) => (
              <EbookCard
                key={item.ebook_id}
                ebook={{
                  id: item.ebook_id,
                  title: item.title,
                  author: item.author,
                  cover_url: item.cover_url,
                  total_pages: item.total_pages,
                  file_size: 0,
                  created_at: item.last_opened,
                }}
                onRead={handleRead}
                progress={{ last_page: item.last_page, total_pages: item.total_pages }}
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
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
            {recentEbooks.map((ebook) => (
              <EbookCard key={ebook.id} ebook={ebook} onRead={handleRead} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
