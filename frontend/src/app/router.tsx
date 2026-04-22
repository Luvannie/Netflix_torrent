import { AppShell } from "../layouts/AppShell";
import { CatalogRoute } from "../features/catalog/route";

export function AppRouter() {
  return (
    <AppShell>
      <CatalogRoute />
    </AppShell>
  );
}

