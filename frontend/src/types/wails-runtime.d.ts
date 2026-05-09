declare global {
  interface WailsRuntime {
    EventsOn(
      name: string,
      callback: (data: unknown) => void
    ): void;

    EventsOnce(
      name: string,
      callback: (data: unknown) => void
    ): void;

    EventsOff(name: string): void;
    EventsEmit(name: string, data?: unknown): void;
  }

  interface Window {
    runtime: WailsRuntime;
  }
}

export {};
