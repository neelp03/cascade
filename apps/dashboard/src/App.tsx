import { NumberWidget } from './components/NumberWidget';

// M1 dashboard: a single number widget proving the full pipeline.
// Tenant ID comes from env or falls back to the dev seed org.
const TENANT_ID = import.meta.env.VITE_TENANT_ID ?? '00000000-0000-0000-0000-000000000001';

const now = new Date();
const from = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
const to = now.toISOString();

export default function App() {
  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <header className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Cascade Analytics</h1>
        <p className="mt-1 text-sm text-gray-500">M1 — last 24 hours</p>
      </header>

      <main className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <NumberWidget
          tenantId={TENANT_ID}
          event="page_view"
          from={from}
          to={to}
          title="Page Views (24h)"
        />
        <NumberWidget
          tenantId={TENANT_ID}
          event="signup"
          from={from}
          to={to}
          title="Signups (24h)"
        />
        <NumberWidget tenantId={TENANT_ID} event="click" from={from} to={to} title="Clicks (24h)" />
      </main>
    </div>
  );
}
