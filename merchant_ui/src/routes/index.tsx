import { createFileRoute } from "@tanstack/react-router";
import { StatsRow } from "@/components/dashboard/stats-row";

export const Route = createFileRoute("/")({
  component: Dashboard,
});

function Dashboard() {
  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <div className="mx-auto w-full max-w-7xl space-y-8">
        <header>
          <h1 className="text-3xl font-bold tracking-tight">
            Arbitrage Dashboard
          </h1>
          <p className="text-muted-foreground">
            Polymarket ↔ Kalshi performance overview
          </p>
        </header>

        <StatsRow />
      </div>
    </div>
  );
}
