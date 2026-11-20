// jsdom does not implement ResizeObserver; react-flow (Canvas) needs it.
// A no-op observer is enough for these component tests, which do not assert on resize.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

if (typeof globalThis.ResizeObserver === 'undefined') {
  (globalThis as any).ResizeObserver = ResizeObserverStub;
}
