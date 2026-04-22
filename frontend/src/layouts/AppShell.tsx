import type { ReactNode } from "react";

type AppShellProps = {
  children?: ReactNode;
};

export function AppShell({ children }: AppShellProps) {
  return (
    <main>
      <header>
        <h1>NetflixTorrent</h1>
      </header>
      <section>{children}</section>
    </main>
  );
}

