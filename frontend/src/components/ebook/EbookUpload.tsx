import { useState } from 'react';
import type { FormEvent } from 'react';
import Modal from '../common/Modal';
import { ebooksApi } from '../../api/ebooks';

interface EbookUploadProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function EbookUpload({ isOpen, onClose, onSuccess }: EbookUploadProps) {
  const [title, setTitle] = useState('');
  const [author, setAuthor] = useState('');
  const [totalPages, setTotalPages] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!file) return;

    setLoading(true);
    setError('');

    const formData = new FormData();
    formData.append('file', file);
    formData.append('title', title);
    formData.append('author', author);
    formData.append('total_pages', totalPages);

    try {
      await ebooksApi.upload(formData);
      setTitle('');
      setAuthor('');
      setTotalPages('');
      setFile(null);
      onSuccess();
      onClose();
    } catch {
      setError('Failed to upload ebook');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Upload Ebook">
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
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Total Pages</label>
          <input
            type="number"
            value={totalPages}
            onChange={(e) => setTotalPages(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            min="1"
            required
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">PDF File</label>
          <input
            type="file"
            accept=".pdf"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
            className="w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded file:border-0 file:text-sm file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
            required
          />
        </div>
        <button
          type="submit"
          disabled={loading}
          className="w-full py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
        >
          {loading ? 'Uploading...' : 'Upload'}
        </button>
      </form>
    </Modal>
  );
}
