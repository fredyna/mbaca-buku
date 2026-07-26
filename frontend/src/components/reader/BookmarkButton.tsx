import { useState, useEffect } from 'react';
import { bookmarksApi } from '../../api/bookmarks';
import type { Bookmark } from '../../api/bookmarks';

interface BookmarkButtonProps {
  ebookId: string;
  currentPage: number;
}

export default function BookmarkButton({ ebookId, currentPage }: BookmarkButtonProps) {
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
  const [showList, setShowList] = useState(false);

  const loadBookmarks = () => {
    bookmarksApi.list(ebookId).then(setBookmarks);
  };

  useEffect(() => {
    loadBookmarks();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ebookId]);

  const isCurrentPageBookmarked = bookmarks.some((b) => b.page_number === currentPage);

  const toggleBookmark = async () => {
    if (isCurrentPageBookmarked) {
      const bm = bookmarks.find((b) => b.page_number === currentPage);
      if (bm) {
        await bookmarksApi.delete(bm.id);
        loadBookmarks();
      }
    } else {
      await bookmarksApi.create(ebookId, currentPage);
      loadBookmarks();
    }
  };

  return (
    <div className="relative">
      <div className="flex items-center gap-2">
        <button
          onClick={toggleBookmark}
          className={`px-3 py-1.5 text-sm rounded border ${
            isCurrentPageBookmarked
              ? 'bg-yellow-50 border-yellow-300 text-yellow-700'
              : 'border-gray-300 text-gray-600 hover:bg-gray-50'
          }`}
        >
          {isCurrentPageBookmarked ? 'Bookmarked' : 'Bookmark'}
        </button>
        <button
          onClick={() => setShowList(!showList)}
          className="px-2 py-1.5 text-sm border border-gray-300 rounded hover:bg-gray-50"
        >
          {bookmarks.length}
        </button>
      </div>

      {showList && bookmarks.length > 0 && (
        <div className="absolute right-0 mt-2 w-48 bg-white border border-gray-200 rounded-lg shadow-lg z-10">
          <div className="p-2 max-h-60 overflow-y-auto">
            {bookmarks.map((b) => (
              <button
                key={b.id}
                onClick={() => {
                  window.dispatchEvent(new CustomEvent('goto-page', { detail: b.page_number }));
                  setShowList(false);
                }}
                className="w-full text-left px-3 py-2 text-sm hover:bg-gray-50 rounded"
              >
                Page {b.page_number}
                {b.note && <span className="block text-xs text-gray-400">{b.note}</span>}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
