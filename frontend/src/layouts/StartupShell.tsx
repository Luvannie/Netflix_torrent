type StartupShellProps = {
  message?: string;
};

export function StartupShell({ message = "Starting desktop services..." }: StartupShellProps) {
  return (
    <main>
      <h1>NetflixTorrent</h1>
      <p>{message}</p>
    </main>
  );
}

