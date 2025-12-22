import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Pie, PieChart } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const data = [
  { name: "Polymarket", value: 68430 },
  { name: "Kalshi", value: 60000 },
];

export function AllocationPie() {
  return (
    <Card className="bg-slate-900/60 border-slate-800">
      <CardHeader>
        <CardTitle>Capital Allocation</CardTitle>
      </CardHeader>
      <CardContent className="h-[320px]">
        <ChartContainer
          config={{
            value: {
              label: "Capital",
            },
          }}
        >
          <PieChart>
            <ChartTooltip content={<ChartTooltipContent />} />
            <Pie
              data={data}
              dataKey="value"
              nameKey="name"
              outerRadius={110}
              fill="var(--chart-2)"
            />
          </PieChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}
