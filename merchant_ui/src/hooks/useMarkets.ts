import { useQuery } from "@tanstack/react-query";

interface Market {
  ticker: string;
  event_ticker: string;
  title: string;
  no_bid_dollars: string;
  yes_bid_dollars: string;
  yes_sub_title: string;
  no_sub_title: string;
  close_time: string;
  yes_ask_dollars: string;
  no_ask_dollars: string;
}

interface MarketsResponse {
  markets: Market[];
  cursor: string;
}

interface UseMarketsOptions {
  limit: number;
  cursor?: string;
  minCloseTs?: number;
  maxCloseTs?: number;
}

export function useMarkets({ limit, cursor, minCloseTs, maxCloseTs }: UseMarketsOptions) {
  return useQuery<MarketsResponse>({
    queryKey: ["markets", limit, cursor, minCloseTs, maxCloseTs],
    queryFn: async () => {
      const params = new URLSearchParams({
        limit: limit.toString(),
        mve_filter: "exclude",
      });

      if (cursor) {
        params.append("cursor", cursor);
      }

      if (minCloseTs) {
        params.append("min_close_ts", minCloseTs.toString());
      }

      if (maxCloseTs) {
        params.append("max_close_ts", maxCloseTs.toString());
      }

      const response = await fetch(
        `${import.meta.env.VITE_API_URL || "http://localhost:8080"}/api/v1/markets?${params.toString()}`
      );

      if (!response.ok) {
        throw new Error("Failed to fetch markets");
      }

      return response.json();
    },
  });
}
