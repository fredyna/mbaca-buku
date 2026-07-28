import { useState, useEffect } from 'react';
import { Document, Page, pdfjs } from 'react-pdf';
import 'react-pdf/dist/Page/AnnotationLayer.css';
import 'react-pdf/dist/Page/TextLayer.css';

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.min.mjs',
  import.meta.url,
).toString();

interface PdfReaderProps {
  fileUrl: string;
  currentPage: number;
  dualPage: boolean;
  onPageChange: (page: number) => void;
  onDocumentLoad: (numPages: number) => void;
}

export default function PdfReader({
  fileUrl,
  currentPage,
  dualPage,
  onPageChange,
  onDocumentLoad,
}: PdfReaderProps) {
  const [numPages, setNumPages] = useState(0);
  const [containerWidth, setContainerWidth] = useState(800);

  useEffect(() => {
    const updateWidth = () => {
      const el = document.getElementById('pdf-container');
      if (el) setContainerWidth(el.clientWidth);
    };
    updateWidth();
    window.addEventListener('resize', updateWidth);
    return () => window.removeEventListener('resize', updateWidth);
  }, []);

  useEffect(() => {
    const handler = (e: Event) => {
      const page = (e as CustomEvent).detail;
      if (page >= 1 && page <= numPages) onPageChange(page);
    };
    window.addEventListener('goto-page', handler);
    return () => window.removeEventListener('goto-page', handler);
  }, [numPages, onPageChange]);

  const onDocLoad = ({ numPages: n }: { numPages: number }) => {
    setNumPages(n);
    onDocumentLoad(n);
  };

  const pageWidth = dualPage ? (containerWidth - 16) / 2 : containerWidth;

  return (
    <div id="pdf-container" className="flex-1 overflow-auto bg-gray-100 flex justify-center p-4">
      <Document file={{ url: fileUrl }} onLoadSuccess={onDocLoad} loading={<div className="text-gray-500">Loading PDF...</div>}>
        <div className={`flex ${dualPage ? 'gap-4' : ''}`}>
          <Page
            pageNumber={currentPage}
            width={pageWidth}
            renderTextLayer={true}
            renderAnnotationLayer={true}
          />
          {dualPage && currentPage + 1 <= numPages && (
            <Page
              pageNumber={currentPage + 1}
              width={pageWidth}
              renderTextLayer={true}
              renderAnnotationLayer={true}
            />
          )}
        </div>
      </Document>
    </div>
  );
}
