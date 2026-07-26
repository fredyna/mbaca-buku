import { useEffect, useRef } from 'react';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useDebouncedCallback<T extends (...args: any[]) => void>(
  callback: T,
  delay: number
): T {
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const callbackRef = useRef(callback);
  const pendingArgsRef = useRef<Parameters<T> | null>(null);

  callbackRef.current = callback;

  useEffect(() => {
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      if (pendingArgsRef.current) {
        callbackRef.current(...pendingArgsRef.current);
        pendingArgsRef.current = null;
      }
    };
  }, []);

  return ((...args: Parameters<T>) => {
    pendingArgsRef.current = args;
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    timeoutRef.current = setTimeout(() => {
      pendingArgsRef.current = null;
      callbackRef.current(...args);
    }, delay);
  }) as T;
}
