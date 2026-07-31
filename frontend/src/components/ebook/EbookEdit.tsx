import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import Modal from '../common/Modal';
import { ebooksApi } from '../../api/ebooks';
import type { Ebook } from '../../api/ebooks';

interface EbookEditProps {
  isOpen: boolean;
  ebook: Ebook | null;
  onClose: () => void;
  onSuccess: () => void;
}

export default function EbookEdit({ isOpen, ebook, onClose, onSuccess }: EbookEditProps) {
  const [title, setTitle] = useState('');
  const [author, setAuthor] = useState('');
  const [isPrivate, setIsPrivate] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isOpen || !ebook) return;
    setTitle(ebook.title);
    setAuthor(ebook.author);
    setIsPrivate(ebook.is_private);
    setError('');
  }, [isOpen, ebook]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!ebook) return;

    setLoading(true);
    setError('');

    try {
      await ebooksApi.update(ebook.id, {
        title,
        author,
        is_private: isPrivate,
      });
      onSuccess();
      onClose();
    } catch (err: unknown) {
      const message =
        (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error
          ?.message || 'Failed to update ebook';
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Edit Ebook">
      {error && <div className="bg-red-50 text-red-600 p-3 rounded mb-4 text-sm">{error}</div>}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Author</label>
          <input
            type="text"
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <label className="flex items-start gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={isPrivate}
            onChange={(e) => setIsPrivate(e.target.checked)}
            className="mt-0.5 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          <span className="text-sm">
            <span className="font-medium text-gray-700">Private</span>
            <span className="block text-xs text-gray-500">
              Only you {`(and admins)`} can see this ebook. Uncheck to share with everyone.
            </span>
          </span>
        </label>
        <div className="flex justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? 'Saving...' : 'Save'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
