export type BootstrapState = {
  step:
    | "IDLE"
    | "ACQUIRING_LOCK"
    | "STARTING_SERVICES"
    | "WAITING_HEALTH"
    | "SETUP_REQUIRED"
    | "READY"
    | "FAILED";
  message: string;
  backendUrl: string;
  webSocketUrl: string;
};

export function decideInitialRoute(state: BootstrapState): string {
  if (state.step === "SETUP_REQUIRED") {
    return "/setup";
  }
  if (state.step === "READY") {
    return "/app/catalog";
  }
  if (state.step === "FAILED") {
    return "/diagnostics";
  }
  return "/startup";
}
