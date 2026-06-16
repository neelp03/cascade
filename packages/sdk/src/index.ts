import type { CaptureEvent } from '@cascade/types';

export interface CascadeConfig {
  apiKey: string;       // tenant_id (public key in v1)
  ingestUrl: string;    // e.g. http://localhost:8080
  flushInterval?: number; // ms between auto-flushes (default 5000)
  maxBatchSize?: number;  // max events per batch (default 50)
  debug?: boolean;
}

export class CascadeSDK {
  private readonly config: Required<CascadeConfig>;
  private queue: CaptureEvent[] = [];
  private flushTimer: ReturnType<typeof setInterval> | null = null;

  constructor(config: CascadeConfig) {
    this.config = {
      flushInterval: 5_000,
      maxBatchSize: 50,
      debug: false,
      ...config,
    };
  }

  init(): void {
    this.flushTimer = setInterval(() => {
      this.flush().catch((e) => {
        if (this.config.debug) console.error('[cascade] flush error:', e);
      });
    }, this.config.flushInterval);
  }

  capture(event: string, properties?: CaptureEvent['properties']): void {
    const ev: CaptureEvent = {
      event,
      timestamp: new Date().toISOString(),
      distinct_id: this.getDistinctId(),
      ...(properties ? { properties } : {}),
    };
    this.queue.push(ev);
    if (this.config.debug) console.debug('[cascade] queued:', ev);
    if (this.queue.length >= this.config.maxBatchSize) {
      this.flush().catch((e) => {
        if (this.config.debug) console.error('[cascade] flush error:', e);
      });
    }
  }

  async flush(): Promise<void> {
    if (this.queue.length === 0) return;
    const batch = this.queue.splice(0, this.config.maxBatchSize);
    await this.sendBatch(batch);
  }

  destroy(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
  }

  private async sendBatch(events: CaptureEvent[]): Promise<void> {
    const res = await fetch(`${this.config.ingestUrl}/v1/batch`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': this.config.apiKey,
      },
      body: JSON.stringify({ events }),
    });
    if (!res.ok) {
      throw new Error(`ingest returned ${res.status}`);
    }
    if (this.config.debug) {
      const body = await res.json();
      console.debug('[cascade] sent batch, response:', body);
    }
  }

  private getDistinctId(): string {
    if (typeof localStorage === 'undefined') return 'anonymous';
    let id = localStorage.getItem('cascade_distinct_id');
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem('cascade_distinct_id', id);
    }
    return id;
  }
}

// Convenience singleton
let _instance: CascadeSDK | null = null;

export function init(config: CascadeConfig): CascadeSDK {
  _instance = new CascadeSDK(config);
  _instance.init();
  return _instance;
}

export function capture(event: string, properties?: CaptureEvent['properties']): void {
  if (!_instance) throw new Error('cascade: call init() first');
  _instance.capture(event, properties);
}

export async function flush(): Promise<void> {
  if (!_instance) return;
  await _instance.flush();
}
