import type { EventSourceLike } from '@/infrastructure/api/httpPriceStreamGateway';

/** Fake für EventSource: Tests lösen Events, Fehler und Öffnen selbst aus. */
export class FakeEventSource implements EventSourceLike {
  static instances: FakeEventSource[] = [];
  static reset(): void {
    FakeEventSource.instances = [];
  }
  static last(): FakeEventSource {
    const instance = FakeEventSource.instances.at(-1);
    if (!instance) throw new Error('keine EventSource erzeugt');
    return instance;
  }
  closed = false;
  onerror: ((event: Event) => void) | null = null;
  onopen: ((event: Event) => void) | null = null;
  private listeners = new Map<string, Array<(event: MessageEvent) => void>>();

  constructor(readonly url: string) {
    FakeEventSource.instances.push(this);
  }
  addEventListener(type: string, listener: EventListenerOrEventListenerObject): void {
    const list = this.listeners.get(type) ?? [];
    list.push(listener as (event: MessageEvent) => void);
    this.listeners.set(type, list);
  }
  close(): void {
    this.closed = true;
  }
  open(): void {
    this.onopen?.(new Event('open'));
  }
  emit(type: string, data: string): void {
    for (const listener of this.listeners.get(type) ?? []) {
      listener(new MessageEvent(type, { data }));
    }
  }
  fail(): void {
    this.onerror?.(new Event('error'));
  }
}
