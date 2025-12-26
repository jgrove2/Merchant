import { createFileRoute } from "@tanstack/react-router";
import MarketTable from "@/components/market-table";


export const Route = createFileRoute("/markets")({
  component: MarketsPage,
});

function MarketsPage() {
  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <MarketTable />
    </div>
  );
}
