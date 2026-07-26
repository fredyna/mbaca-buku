interface PageControlsProps {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  dualPage: boolean;
  onToggleDualPage: () => void;
}

export default function PageControls({
  currentPage,
  totalPages,
  onPageChange,
  dualPage,
  onToggleDualPage,
}: PageControlsProps) {
  const step = dualPage ? 2 : 1;

  return (
    <div className="flex items-center justify-between bg-white border-t border-gray-200 px-4 py-3">
      <button
        onClick={() => onPageChange(Math.max(1, currentPage - step))}
        disabled={currentPage <= 1}
        className="px-4 py-2 text-sm bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Previous
      </button>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-600">Page</span>
          <input
            type="number"
            value={currentPage}
            onChange={(e) => {
              const p = parseInt(e.target.value);
              if (p >= 1 && p <= totalPages) onPageChange(p);
            }}
            className="w-16 px-2 py-1 text-center border border-gray-300 rounded text-sm"
            min={1}
            max={totalPages}
          />
          <span className="text-sm text-gray-600">of {totalPages}</span>
        </div>

        <button
          onClick={onToggleDualPage}
          className="px-3 py-1.5 text-xs border border-gray-300 rounded hover:bg-gray-50"
        >
          {dualPage ? '1 Page' : '2 Pages'}
        </button>
      </div>

      <button
        onClick={() => onPageChange(Math.min(totalPages, currentPage + step))}
        disabled={currentPage >= totalPages}
        className="px-4 py-2 text-sm bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Next
      </button>
    </div>
  );
}
