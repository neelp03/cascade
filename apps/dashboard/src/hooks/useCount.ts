import { useEffect, useState } from 'react';
import { fetchCount } from '../api/query';

interface State {
  count: number | null;
  loading: boolean;
  error: string | null;
}

export function useCount(
  tenantId: string,
  event: string,
  from: string,
  to: string,
  refreshMs = 10_000
): State {
  const [state, setState] = useState<State>({ count: null, loading: true, error: null });

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setState((s) => ({ ...s, loading: true, error: null }));
      try {
        const res = await fetchCount(tenantId, event, from, to);
        if (!cancelled) setState({ count: res.count, loading: false, error: null });
      } catch (e) {
        if (!cancelled) setState({ count: null, loading: false, error: String(e) });
      }
    };

    load();
    const id = setInterval(load, refreshMs);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [tenantId, event, from, to, refreshMs]);

  return state;
}
