import { createFileRoute } from "@tanstack/react-router";
import { StatsRow } from "@/components/dashboard/stats-row";
import { PerformanceLine } from "@/components/dashboard/performance-line";
import { AllocationPie } from "@/components/dashboard/allocation-pie";

export const Route = createFileRoute("/")({
  component: Dashboard,
});

function Dashboard() {
  return (
    <div>
      <div className="mx-auto max-w-7xl space-y-8 px-6 py-10">
        <header>
          <h1 className="text-3xl font-bold tracking-tight">
            Arbitrage Dashboard
          </h1>
          <p>
            Polymarket ↔ Kalshi performance overview
          </p>
        </header>

        <StatsRow />

        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <PerformanceLine />
          </div>
          <AllocationPie />
        </div>
      </div>
    </div>
  );
}
