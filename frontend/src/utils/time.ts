/**
 * Timestamp formatting for the admin activity columns.
 *
 * Relative wording answers "is this person around?" at a glance, which an
 * absolute date does not; absolute formatting is kept for the sign-in column,
 * where the exact moment is the useful part.
 */

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/**
 * How recent counts as online. Matches the activity throttle window on the
 * server: a row is written at most once every five minutes, so a shorter
 * threshold here would show an actively reading user as offline.
 */
export const ACTIVE_WINDOW_MS = 5 * MINUTE;

export function formatRelativeTime(iso: string | null): string {
  if (!iso) return 'Never';

  const diff = Date.now() - new Date(iso).getTime();
  // A negative difference means the server clock is ahead of the browser's.
  // "Just now" is honest there; "in 3 minutes" would look broken.
  if (diff < MINUTE) return 'Just now';
  if (diff < HOUR) return `${Math.floor(diff / MINUTE)} min ago`;

  if (diff < DAY) {
    const hours = Math.floor(diff / HOUR);
    return `${hours} hour${hours === 1 ? '' : 's'} ago`;
  }

  const days = Math.floor(diff / DAY);
  // Past a month, "47 days ago" is harder to read than the date itself.
  if (days < 30) return `${days} day${days === 1 ? '' : 's'} ago`;
  return formatDateTime(iso);
}

export function formatDateTime(iso: string | null): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleString(undefined, {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function isRecentlyActive(iso: string | null): boolean {
  if (!iso) return false;
  return Date.now() - new Date(iso).getTime() < ACTIVE_WINDOW_MS;
}
