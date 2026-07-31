import type { Ebook } from '../../api/ebooks';
import type { ViewMode } from '../common/ViewToggle';
import { progressPercent } from '../../utils/progress';

interface EbookCardProps {
  ebook: Ebook;
  onRead: (id: string) => void;
  onEdit?: (ebook: Ebook) => void;
  onDelete?: (id: string) => void;
  onShowDetail?: (ebook: Ebook) => void;
  progress?: { last_page: number; total_pages: number };
  /** Caption above the progress bar, e.g. 'Re-read progress' for a book being read again. */
  progressLabel?: string;
  /** Short status text shown in place of the progress bar, e.g. 'Finished'. */
  note?: string;
  viewMode?: ViewMode;
}

function PrivacyBadge({ isPrivate }: { isPrivate: boolean }) {
  if (isPrivate) {
    return (
      <span className="inline-block px-1.5 py-0.5 text-[10px] font-medium rounded bg-gray-200 text-gray-700">
        Private
      </span>
    );
  }
  return (
    <span className="inline-block px-1.5 py-0.5 text-[10px] font-medium rounded bg-green-100 text-green-700">
      Public
    </span>
  );
}

export default function EbookCard({
  ebook,
  onRead,
  onEdit,
  onDelete,
  onShowDetail,
  progress,
  progressLabel = 'Progress',
  note,
  viewMode = 'grid',
}: EbookCardProps) {
  const TitleTag = onShowDetail ? 'button' : 'h3';
  const titleProps = onShowDetail
    ? { onClick: () => onShowDetail(ebook), title: 'View details' }
    : {};
  const percent = progress ? progressPercent(progress.last_page, progress.total_pages) : 0;

  if (viewMode === 'list') {
    return (
      <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow p-3 flex items-center gap-4">
        <div className="w-14 h-20 shrink-0 rounded bg-gradient-to-br from-blue-500 to-blue-700 flex items-center justify-center">
          <span className="text-white text-xs font-bold opacity-40">PDF</span>
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 min-w-0">
            <TitleTag
              {...titleProps}
              className={`font-semibold text-gray-900 truncate text-left ${
                onShowDetail ? 'hover:text-blue-600 hover:underline cursor-pointer' : ''
              }`}
            >
              {ebook.title}
            </TitleTag>
            <PrivacyBadge isPrivate={ebook.is_private} />
          </div>
          <p className="text-sm text-gray-500 truncate">{ebook.author || 'Unknown author'}</p>
          <p className="text-xs text-gray-400 mt-0.5">{ebook.total_pages} pages</p>

          {progress ? (
            <div className="mt-2 max-w-xs">
              <div className="flex justify-between text-xs text-gray-500 mb-1">
                <span>{progressLabel}</span>
                <span>{percent}%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-1.5">
                <div
                  className="bg-blue-600 h-1.5 rounded-full"
                  style={{ width: `${percent}%` }}
                />
              </div>
            </div>
          ) : (
            note && <p className="mt-2 text-xs font-medium text-green-700">{note}</p>
          )}
        </div>

        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 shrink-0">
          <button
            onClick={() => onRead(ebook.id)}
            className="px-3 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            Read
          </button>
          {onEdit && (
            <button
              onClick={() => onEdit(ebook)}
              className="px-3 py-1.5 text-gray-700 text-sm border border-gray-300 rounded hover:bg-gray-50"
            >
              Edit
            </button>
          )}
          {onDelete && (
            <button
              onClick={() => onDelete(ebook.id)}
              className="px-3 py-1.5 text-red-600 text-sm border border-red-200 rounded hover:bg-red-50"
            >
              Delete
            </button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow overflow-hidden">
      <div className="relative h-48 bg-gradient-to-br from-blue-500 to-blue-700 flex items-center justify-center">
        <span className="text-white text-4xl font-bold opacity-30">PDF</span>
        <div className="absolute top-2 right-2">
          <PrivacyBadge isPrivate={ebook.is_private} />
        </div>
      </div>
      <div className="p-4">
        <TitleTag
          {...titleProps}
          className={`font-semibold text-gray-900 truncate text-left w-full ${
            onShowDetail ? 'hover:text-blue-600 hover:underline cursor-pointer' : ''
          }`}
        >
          {ebook.title}
        </TitleTag>
        <p className="text-sm text-gray-500 mt-1">{ebook.author || 'Unknown author'}</p>
        <p className="text-xs text-gray-400 mt-1">{ebook.total_pages} pages</p>

        {progress ? (
          <div className="mt-3">
            <div className="flex justify-between text-xs text-gray-500 mb-1">
              <span>{progressLabel}</span>
              <span>{percent}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-1.5">
              <div
                className="bg-blue-600 h-1.5 rounded-full"
                style={{ width: `${percent}%` }}
              />
            </div>
          </div>
        ) : (
          note && <p className="mt-3 text-xs font-medium text-green-700">{note}</p>
        )}

        <div className="mt-4 flex gap-2">
          <button
            onClick={() => onRead(ebook.id)}
            className="flex-1 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            Read
          </button>
          {onEdit && (
            <button
              onClick={() => onEdit(ebook)}
              className="py-1.5 px-3 text-gray-700 text-sm border border-gray-300 rounded hover:bg-gray-50"
            >
              Edit
            </button>
          )}
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
