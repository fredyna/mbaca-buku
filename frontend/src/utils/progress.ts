/**
 * Reading progress as a whole percentage, shared by every screen that shows it
 * so the dashboard, the history list and the reader can never disagree.
 */
export function progressPercent(currentPage: number, totalPages: number): number {
  if (!totalPages || totalPages < 1) return 0;
  const clamped = Math.min(Math.max(currentPage, 0), totalPages);
  return Math.round((clamped / totalPages) * 100);
}
