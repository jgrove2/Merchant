import { StatCard } from "./stat-card";
import {
  TrendingUp,
  Wallet,
  DollarSign,
  Activity,
} from "lucide-react";

export function StatsRow() {
  // mock data (replace with real backend values later)
  const stats = {
    sevenDayChange: "+4.21%",
    betsValue: "$128,430",
    cashRemaining: "$41,200",
    transactionsToday: "37",
  };

  return (
    <div className="grid gap-4 md:grid-cols-4">
      <StatCard
        title="7D Performance"
        value={stats.sevenDayChange}
        icon={<TrendingUp size={18} />}
      />
      <StatCard
        title="Open Bets Value"
        value={stats.betsValue}
        icon={<Wallet size={18} />}
      />
      <StatCard
        title="Cash Remaining"
        value={stats.cashRemaining}
        icon={<DollarSign size={18} />}
      />
      <StatCard
        title="Transactions Today"
        value={stats.transactionsToday}
        icon={<Activity size={18} />}
      />
    </div>
  );
}
