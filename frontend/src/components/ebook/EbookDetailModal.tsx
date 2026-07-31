import type { Ebook } from '../../api/ebooks';
import Modal from '../common/Modal';

interface EbookDetailModalProps {
  ebook: Ebook | null;
  onClose: () => void;
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4 py-2 border-b border-gray-100 last:border-0">
      <span className="text-sm text-gray-500 shrink-0">{label}</span>
      <span className="text-sm text-gray-900 text-right break-words">{value}</span>
    </div>
  );
}

export default function EbookDetailModal({ ebook, onClose }: EbookDetailModalProps) {
  if (!ebook) return null;

  return (
    <Modal isOpen={!!ebook} onClose={onClose} title={ebook.title}>
      <div className="flex flex-col">
        <Row label="Author" value={ebook.author || 'Unknown author'} />
        <Row label="Pages" value={`${ebook.total_pages}`} />
        <Row label="Visibility" value={ebook.is_private ? 'Private' : 'Public'} />
        <Row label="Uploaded by" value={ebook.uploaded_by_name || 'Unknown'} />
        <Row label="Uploaded at" value={new Date(ebook.created_at).toLocaleString()} />
      </div>
    </Modal>
  );
}
