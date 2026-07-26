interface ProgressBarProps {
  currentPage: number;
  totalPages: number;
}

export default function ProgressBar({ currentPage, totalPages }: ProgressBarProps) {
  const percent = totalPages > 0 ? Math.round((currentPage / totalPages) * 100) : 0;

  return (
    <div className="flex items-center gap-3">
      <div className="flex-1 bg-gray-200 rounded-full h-2">
        <div
          className="bg-blue-600 h-2 rounded-full transition-all"
          style={{ width: `${percent}%` }}
        />
      </div>
      <span className="text-sm text-gray-500 whitespace-nowrap">{percent}%</span>
    </div>
  );
}
