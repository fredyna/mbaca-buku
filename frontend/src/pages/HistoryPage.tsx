import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { HistoryItem } from '../api/history';
import { historyApi } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EmptyState from '../components/common/EmptyState';

export default function HistoryPage() {
  const [reading, setReading] = useState<HistoryItem[]>([]);
  const [completed, setCompleted] = useState<HistoryItem[]>([]);
  const [tab, setTab] = useState<'reading' | 'completed'>('reading');
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

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Reading History</h1>

      <div className="flex gap-1 mb-6 bg-gray-100 p-1 rounded-lg w-fit">
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
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {items.map((item) => (
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
    </div>
  );
}
