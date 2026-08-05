interface PaginationProps {
  page: number;
  perPage: number;
  total: number;
  onChange: (page: number) => void;
}

type PageItem = number | 'gap';

// Shows every page while the list is short, and collapses the middle once it
// is not: 1 … 4 5 6 … 20. The current page always keeps a neighbour on each
// side so the numbers do not jump around as the user pages through.
function pageItems(page: number, totalPages: number): PageItem[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  const items: PageItem[] = [1];
  const start = Math.max(2, Math.min(page - 1, totalPages - 4));
  const end = Math.min(totalPages - 1, Math.max(page + 1, 5));

  if (start > 2) items.push('gap');
  for (let p = start; p <= end; p++) items.push(p);
  if (end < totalPages - 1) items.push('gap');

  items.push(totalPages);
  return items;
}

export default function Pagination({ page, perPage, total, onChange }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / perPage));
  if (totalPages <= 1) return null;

  const first = (page - 1) * perPage + 1;
  const last = Math.min(page * perPage, total);

  const arrow = 'px-3 py-1.5 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed';

  return (
    <nav
      aria-label="Pagination"
      className="mt-6 flex flex-col sm:flex-row items-center justify-between gap-3"
    >
      <p className="text-sm text-gray-500">
        Showing {first}–{last} of {total} ebooks
      </p>

      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={() => onChange(page - 1)}
          disabled={page <= 1}
          className={arrow}
        >
          Prev
        </button>

        {pageItems(page, totalPages).map((item, i) =>
          item === 'gap' ? (
            <span key={`gap-${i}`} className="px-2 text-gray-400">
              …
            </span>
          ) : (
            <button
              key={item}
              type="button"
              onClick={() => onChange(item)}
              aria-current={item === page ? 'page' : undefined}
              className={`min-w-9 px-3 py-1.5 text-sm rounded border ${
                item === page
                  ? 'bg-blue-600 border-blue-600 text-white'
                  : 'border-gray-300 text-gray-700 hover:bg-gray-50'
              }`}
            >
              {item}
            </button>
          )
        )}

        <button
          type="button"
          onClick={() => onChange(page + 1)}
          disabled={page >= totalPages}
          className={arrow}
        >
          Next
        </button>
      </div>
    </nav>
  );
}
