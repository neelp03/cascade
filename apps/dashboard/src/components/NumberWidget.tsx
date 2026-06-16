import { useCount } from '../hooks/useCount';

interface Props {
  tenantId: string;
  event: string;
  from: string;
  to: string;
  title: string;
}

export function NumberWidget({ tenantId, event, from, to, title }: Props) {
  const { count, loading, error } = useCount(tenantId, event, from, to);

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
      <p className="text-sm font-medium text-gray-500">{title}</p>
      <div className="mt-2">
        {loading && count === null && (
          <span className="text-3xl font-bold text-gray-300">—</span>
        )}
        {error && (
          <span className="text-sm text-red-500">{error}</span>
        )}
        {count !== null && (
          <span className="text-3xl font-bold text-gray-900">
            {count.toLocaleString()}
          </span>
        )}
      </div>
      <p className="mt-1 text-xs text-gray-400">
        {event} · {loading ? 'refreshing…' : 'live'}
      </p>
    </div>
  );
}
