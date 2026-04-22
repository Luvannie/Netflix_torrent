export type SubscriptionTarget = `/topic/search/jobs/${number}` | `/topic/downloads/${number}`;

export type RealtimeMessageHandler = (payload: unknown) => void;

export interface RealtimeAdapter {
  connect(): Promise<void>;
  disconnect(): Promise<void>;
  subscribe(destination: SubscriptionTarget, onMessage: RealtimeMessageHandler): () => void;
}

export function createDisabledRealtimeAdapter(): RealtimeAdapter {
  return {
    async connect() {},
    async disconnect() {},
    subscribe() {
      return () => {};
    },
  };
}

