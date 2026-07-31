interface PageControlsProps {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  dualPage: boolean;
  onToggleDualPage: () => void;
  zoom: number;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onZoomReset: () => void;
  canZoomIn: boolean;
  canZoomOut: boolean;
  isCompleted: boolean;
  onToggleCompleted: () => void;
}

export default function PageControls({
  currentPage,
  totalPages,
  onPageChange,
  dualPage,
  onToggleDualPage,
  zoom,
  onZoomIn,
  onZoomOut,
  onZoomReset,
  canZoomIn,
  canZoomOut,
  isCompleted,
  onToggleCompleted,
}: PageControlsProps) {
  const step = dualPage ? 2 : 1;

  const goPrev = () => onPageChange(Math.max(1, currentPage - step));
  const goNext = () => onPageChange(Math.min(totalPages, currentPage + step));
  const atStart = currentPage <= 1;
  const atEnd = currentPage >= totalPages;

  // In dual-page mode the final spread already shows the last page, so the
  // confirmation belongs there too.
  const lastPageVisible =
    totalPages > 0 && (currentPage >= totalPages || (dualPage && currentPage + 1 >= totalPages));

  return (
    <div className="bg-white border-t border-gray-200 px-3 py-2 sm:px-4 sm:py-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
        <div className="flex items-center justify-between gap-2 sm:justify-start sm:gap-4">
          <button
            onClick={goPrev}
            disabled={atStart}
            aria-label="Previous page"
            className="px-3 py-2 text-sm bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed sm:px-4"
          >
            <span className="sm:hidden">&larr;</span>
            <span className="hidden sm:inline">Previous</span>
          </button>

          <div className="flex items-center gap-1.5 sm:gap-2">
            <span className="hidden text-sm text-gray-600 sm:inline">Page</span>
            <input
              type="number"
              value={currentPage}
              onChange={(e) => {
                const p = parseInt(e.target.value);
                if (p >= 1 && p <= totalPages) onPageChange(p);
              }}
              className="w-14 px-2 py-1 text-center border border-gray-300 rounded text-sm sm:w-16"
              min={1}
              max={totalPages}
            />
            <span className="text-sm text-gray-600 whitespace-nowrap">/ {totalPages}</span>
          </div>

          <button
            onClick={goNext}
            disabled={atEnd}
            aria-label="Next page"
            className="px-3 py-2 text-sm bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed sm:px-4"
          >
            <span className="sm:hidden">&rarr;</span>
            <span className="hidden sm:inline">Next</span>
          </button>
        </div>

        <div className="flex items-center justify-center gap-3 sm:gap-4">
          <div className="flex items-center border border-gray-300 rounded">
            <button
              onClick={onZoomOut}
              disabled={!canZoomOut}
              aria-label="Zoom out"
              className="px-3 py-1.5 text-base leading-none hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed sm:px-2 sm:py-1 sm:text-sm"
            >
              &minus;
            </button>
            <button
              onClick={onZoomReset}
              aria-label="Reset zoom"
              className="px-2 py-1.5 text-xs w-14 text-center border-x border-gray-300 hover:bg-gray-50 sm:w-12 sm:py-1"
            >
              {Math.round(zoom * 100)}%
            </button>
            <button
              onClick={onZoomIn}
              disabled={!canZoomIn}
              aria-label="Zoom in"
              className="px-3 py-1.5 text-base leading-none hover:bg-gray-100 disabled:opacity-40 disabled:cursor-not-allowed sm:px-2 sm:py-1 sm:text-sm"
            >
              +
            </button>
          </div>

          <button
            onClick={onToggleDualPage}
            className="hidden px-3 py-1.5 text-xs border border-gray-300 rounded hover:bg-gray-50 lg:inline-block"
          >
            {dualPage ? '1 Page' : '2 Pages'}
          </button>

          {lastPageVisible && (
            <button
              onClick={onToggleCompleted}
              className={`px-3 py-1.5 text-xs font-medium rounded whitespace-nowrap ${
                isCompleted
                  ? 'border border-gray-300 text-gray-700 hover:bg-gray-50'
                  : 'bg-green-600 text-white hover:bg-green-700'
              }`}
            >
              {isCompleted ? 'Mark as In Progress' : 'Mark as Completed'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
