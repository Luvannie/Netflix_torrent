import type { ReactNode } from "react";

type SetupShellProps = {
  children?: ReactNode;
};

export function SetupShell({ children }: SetupShellProps) {
  return (
    <main>
      <h1>Setup</h1>
      {children ?? <p>Setup wizard</p>}
    </main>
  );
}

