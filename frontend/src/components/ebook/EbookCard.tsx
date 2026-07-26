import type { Ebook } from '../../api/ebooks';

interface EbookCardProps {
  ebook: Ebook;
  onRead: (id: string) => void;
  onDelete?: (id: string) => void;
  progress?: { last_page: number; total_pages: number };
}

export default function EbookCard({ ebook, onRead, onDelete, progress }: EbookCardProps) {
  const progressPercent = progress
    ? Math.round((progress.last_page / progress.total_pages) * 100)
    : 0;

  return (
    <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow overflow-hidden">
      <div className="h-48 bg-gradient-to-br from-blue-500 to-blue-700 flex items-center justify-center">
        <span className="text-white text-4xl font-bold opacity-30">PDF</span>
      </div>
      <div className="p-4">
        <h3 className="font-semibold text-gray-900 truncate">{ebook.title}</h3>
        <p className="text-sm text-gray-500 mt-1">{ebook.author || 'Unknown author'}</p>
        <p className="text-xs text-gray-400 mt-1">{ebook.total_pages} pages</p>

        {progress && (
          <div className="mt-3">
            <div className="flex justify-between text-xs text-gray-500 mb-1">
              <span>Progress</span>
              <span>{progressPercent}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-1.5">
              <div
                className="bg-blue-600 h-1.5 rounded-full"
                style={{ width: `${progressPercent}%` }}
              />
            </div>
          </div>
        )}

        <div className="mt-4 flex gap-2">
          <button
            onClick={() => onRead(ebook.id)}
            className="flex-1 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            Read
          </button>
          {onDelete && (
            <button
              onClick={() => onDelete(ebook.id)}
              className="py-1.5 px-3 text-red-600 text-sm border border-red-200 rounded hover:bg-red-50"
            >
              Delete
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
