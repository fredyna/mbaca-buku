import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import type { Ebook, EbookSort, EbookVisibility } from '../api/ebooks';
import { EBOOKS_PER_PAGE, ebooksApi } from '../api/ebooks';
import { historyApi } from '../api/history';
import EbookCard from '../components/ebook/EbookCard';
import EbookUpload from '../components/ebook/EbookUpload';
import EbookEdit from '../components/ebook/EbookEdit';
import EbookDetailModal from '../components/ebook/EbookDetailModal';
import EmptyState from '../components/common/EmptyState';
import Pagination from '../components/common/Pagination';
import ViewToggle from '../components/common/ViewToggle';
import type { ViewMode } from '../components/common/ViewToggle';
import { usePersistentState } from '../hooks/usePersistentState';
import { useDebouncedCallback } from '../hooks/useDebounce';
import { useAuth } from '../context/AuthContext';

const VISIBILITIES: EbookVisibility[] = ['all', 'public', 'private'];
const SORTS: EbookSort[] = ['title', 'latest'];

const SORT_LABELS: Record<EbookSort, string> = {
  title: 'Title (A-Z)',
  latest: 'Latest',
};

const VISIBILITY_LABELS: Record<EbookVisibility, string> = {
  all: 'All ebooks',
  public: 'Public only',
  private: 'Private only',
};

const inputClass =
  'px-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500';

export default function EbookListPage() {
  const [searchParams, setSearchParams] = useSearchParams();

  // The URL is the single source of truth for the filters, so a filtered view
  // can be shared, survives the browser Back button, and is still there after
  // opening a book and coming back.
  const q = searchParams.get('q')?.trim() ?? '';
  const author = searchParams.get('author') ?? '';
  const visibilityParam = searchParams.get('visibility') as EbookVisibility | null;
  const visibility = visibilityParam && VISIBILITIES.includes(visibilityParam) ? visibilityParam : 'all';
  const sortParam = searchParams.get('sort') as EbookSort | null;
  const sort = sortParam && SORTS.includes(sortParam) ? sortParam : 'title';
  const page = Math.max(1, Number(searchParams.get('page')) || 1);

  const [ebooks, setEbooks] = useState<Ebook[]>([]);
  const [total, setTotal] = useState(0);
  const [authors, setAuthors] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState('');

  const [searchInput, setSearchInput] = useState(q);
  const [showUpload, setShowUpload] = useState(false);
  const [editing, setEditing] = useState<Ebook | null>(null);
  const [detail, setDetail] = useState<Ebook | null>(null);
  const [viewMode, setViewMode] = usePersistentState<ViewMode>('ebooks:viewMode', 'grid');

  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const hasFilters = q !== '' || author !== '' || visibility !== 'all';

  const canManage = (e: Ebook) => isAdmin || e.uploaded_by === user?.id;

  // Changing a filter always returns to the first page: page 4 of the old
  // result set says nothing about the new one.
  const setFilter = (changes: Record<string, string>, replace = false) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        Object.entries(changes).forEach(([key, value]) =>
          value ? next.set(key, value) : next.delete(key)
        );
        next.delete('page');
        return next;
      },
      { replace }
    );
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

  // Typing replaces the current history entry instead of pushing one per
  // keystroke, so Back leaves the search rather than walking it backwards.
  const commitSearch = useDebouncedCallback((value: string) => {
    setFilter({ q: value.trim() }, true);
  }, 300);

  // Keeps the box in step when the query changes from somewhere else: Reset,
  // the Back button, or a shared link.
  useEffect(() => {
    setSearchInput(q);
  }, [q]);

  // Only the newest request may write to state. Typing fires several in a row
  // and they can finish out of order, which would otherwise leave the grid
  // showing results for a query the user has already moved past.
  const requestRef = useRef(0);

  const loadEbooks = useCallback(() => {
    const seq = ++requestRef.current;
    setLoading(true);

    ebooksApi
      .list({ page, q, author, visibility, sort })
      .then(({ ebooks, meta }) => {
        if (seq !== requestRef.current) return;
        setEbooks(ebooks);
        setTotal(meta.total);
        setErrorMsg('');
      })
      .catch(() => {
        if (seq !== requestRef.current) return;
        setErrorMsg('Failed to load ebooks');
      })
      .finally(() => {
        if (seq === requestRef.current) setLoading(false);
      });
  }, [page, q, author, visibility, sort]);

  const loadAuthors = useCallback(() => {
    ebooksApi.listAuthors().then(setAuthors).catch(() => setAuthors([]));
  }, []);

  useEffect(() => {
    loadEbooks();
  }, [loadEbooks]);

  useEffect(() => {
    loadAuthors();
  }, [loadAuthors]);

  // Deleting the last ebook on the last page leaves the URL pointing past the
  // end of the list; step back to the last page that still has results.
  const totalPages = Math.max(1, Math.ceil(total / EBOOKS_PER_PAGE));
  useEffect(() => {
    if (!loading && page > totalPages) goToPage(totalPages, true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading, page, totalPages]);

  const refresh = () => {
    loadEbooks();
    loadAuthors();
  };

  const handleRead = async (id: string) => {
    await historyApi.openBook(id);
    navigate(`/read/${id}`);
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this ebook?')) return;
    await ebooksApi.delete(id);
    refresh();
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
      <div className="flex items-center justify-between mb-4 gap-3">
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

      <div className="flex flex-wrap items-center gap-2 mb-6">
        <input
          type="search"
          value={searchInput}
          onChange={(e) => {
            setSearchInput(e.target.value);
            commitSearch(e.target.value);
          }}
          placeholder="Search by title..."
          aria-label="Search by title"
          className={`${inputClass} flex-1 min-w-50`}
        />

        <select
          value={author}
          onChange={(e) => setFilter({ author: e.target.value })}
          aria-label="Filter by author"
          className={inputClass}
        >
          <option value="">All authors</option>
          {/* The active author may have vanished from the list after an edit;
              keep an entry for it so the select still shows what is applied. */}
          {author && !authors.includes(author) && <option value={author}>{author}</option>}
          {authors.map((name) => (
            <option key={name} value={name}>
              {name}
            </option>
          ))}
        </select>

        <select
          value={visibility}
          onChange={(e) => setFilter({ visibility: e.target.value })}
          aria-label="Filter by visibility"
          className={inputClass}
        >
          {VISIBILITIES.map((value) => (
            <option key={value} value={value}>
              {VISIBILITY_LABELS[value]}
            </option>
          ))}
        </select>

        <select
          value={sort}
          onChange={(e) => setFilter({ sort: e.target.value })}
          aria-label="Sort ebooks"
          className={inputClass}
        >
          {SORTS.map((value) => (
            <option key={value} value={value}>
              {SORT_LABELS[value]}
            </option>
          ))}
        </select>

        {hasFilters && (
          <button
            type="button"
            onClick={() => setFilter({ q: '', author: '', visibility: '' })}
            className="px-3 py-2 text-sm text-gray-600 hover:text-gray-900 underline"
          >
            Reset filters
          </button>
        )}
      </div>

      {errorMsg && <div className="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{errorMsg}</div>}

      {loading ? (
        <div className="text-gray-500">Loading ebooks...</div>
      ) : ebooks.length === 0 ? (
        hasFilters ? (
          <EmptyState
            title="No matching ebooks"
            description="No ebook matches the current filters. Try a different keyword or reset the filters."
          />
        ) : (
          <EmptyState
            title="No ebooks yet"
            description="Upload your first PDF ebook to get started"
          />
        )
      ) : (
        <>
          {viewMode === 'grid' ? (
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

          <Pagination page={page} perPage={EBOOKS_PER_PAGE} total={total} onChange={goToPage} />
        </>
      )}

      <EbookUpload isOpen={showUpload} onClose={() => setShowUpload(false)} onSuccess={refresh} />
      <EbookEdit
        isOpen={!!editing}
        ebook={editing}
        onClose={() => setEditing(null)}
        onSuccess={refresh}
      />
      <EbookDetailModal ebook={detail} onClose={() => setDetail(null)} />
    </div>
  );
}
