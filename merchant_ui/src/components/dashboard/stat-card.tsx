import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { ReactNode } from "react";

type Props = {
  title: string;
  value: string;
  change?: string;
  icon: ReactNode;
};

export function StatCard({ title, value, change, icon }: Props) {
  return (
    <Card className="bg-slate-900/60 border-slate-800">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-slate-400">
          {title}
        </CardTitle>
        <div className="text-slate-500">{icon}</div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {change && (
          <p className="text-xs text-emerald-400 mt-1">{change}</p>
        )}
      </CardContent>
    </Card>
  );
}
