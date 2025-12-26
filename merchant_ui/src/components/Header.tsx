import { Link } from "@tanstack/react-router";
import { useBalance } from "@/hooks/useBalance";
import { Home, TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Separator } from "@/components/ui/separator";

export default function Header() {
  const { data: balanceData, isLoading } = useBalance();

  const formatCurrency = (cents: number) => {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: 2,
    }).format(cents / 100);
  };

  const totalBets = 0; // TODO: Get from API

  return (
    <header className="flex h-16 shrink-0 items-center gap-2 border-b px-4 bg-background">
      <div className="flex items-center gap-4">
        <Link to="/" className="text-lg font-semibold">
          Merchant
        </Link>
        <Separator orientation="vertical" className="h-4" />
        <nav className="flex items-center gap-2">
          <Button variant="ghost" size="sm" asChild>
            <Link to="/">
              <Home className="h-4 w-4 mr-2" />
              Home
            </Link>
          </Button>
          <Button variant="ghost" size="sm" asChild>
            <Link to="/markets">
              <TrendingUp className="h-4 w-4 mr-2" />
              Markets
            </Link>
          </Button>
        </nav>
      </div>

      <div className="flex flex-1 items-center justify-end">
        {!isLoading && balanceData && (
          <div className="flex items-center gap-3 text-sm font-medium">
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="text-muted-foreground hover:text-foreground transition-colors cursor-help">
                  {formatCurrency(totalBets)}
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <p>Total in Bets</p>
              </TooltipContent>
            </Tooltip>
            <Separator orientation="vertical" className="h-4" />
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="text-primary hover:text-primary/80 transition-colors cursor-help">
                  {formatCurrency(balanceData.total_balance)}
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <p>Cash Available</p>
              </TooltipContent>
            </Tooltip>
          </div>
        )}
      </div>
    </header>
  );
}
