import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Line, LineChart, XAxis, YAxis } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const data = [
  { day: "Mon", pnl: 120 },
  { day: "Tue", pnl: 340 },
  { day: "Wed", pnl: 290 },
  { day: "Thu", pnl: 480 },
  { day: "Fri", pnl: 610 },
  { day: "Sat", pnl: 720 },
  { day: "Sun", pnl: 860 },
];

export function PerformanceLine() {
  return (
    <Card className="bg-slate-900/60 border-slate-800">
      <CardHeader>
        <CardTitle>7 Day Performance</CardTitle>
      </CardHeader>
      <CardContent className="h-[320px]">
        <ChartContainer
          config={{
            pnl: {
              label: "PnL",
              color: "hsl(var(--chart-1))",
            },
          }}
        >
          <LineChart data={data}>
            <XAxis dataKey="day" tick={{ fill: "hsl(var(--muted-foreground))"}} />
            <YAxis />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Line
              type="monotone"
              dataKey="pnl"
              stroke="var(--color-pnl)"
              strokeWidth={2}
              dot={false}
            />
          </LineChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}
