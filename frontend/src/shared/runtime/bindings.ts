export type NativeBootstrapState = {
  step: "IDLE" | "STARTING_BACKEND" | "WAITING_HEALTH" | "READY" | "FAILED";
  message: string;
  backendUrl: string;
  webSocketUrl: string;
};

export interface RuntimeBindings {
  GetBootstrapState(): Promise<NativeBootstrapState>;
  ChooseDirectory(): Promise<string>;
  SaveLauncherConfig(input: Record<string, unknown>): Promise<void>;
  RestartBackend(): Promise<void>;
  RestartSidecar(name: string): Promise<void>;
  OpenLogsFolder(): Promise<void>;
  GetLogBundlePath(): Promise<string>;
  OpenExternalURL(url: string): Promise<void>;
  QuitApp(): Promise<void>;
}

declare global {
  interface Window {
    desktopApp?: RuntimeBindings;
  }
}

export function getRuntimeBindings(): RuntimeBindings {
  if (!window.desktopApp) {
    throw new Error("Wails runtime bindings are not available");
  }
  return window.desktopApp;
}

