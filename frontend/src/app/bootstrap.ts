export type BootstrapState = {
  step: "IDLE" | "STARTING_BACKEND" | "WAITING_HEALTH" | "READY" | "FAILED";
  message: string;
  backendUrl: string;
  webSocketUrl: string;
};

export function decideInitialRoute(state: BootstrapState): string {
  if (state.step === "READY") {
    return "/app/catalog";
  }
  if (state.step === "FAILED") {
    return "/diagnostics";
  }
  return "/startup";
}

