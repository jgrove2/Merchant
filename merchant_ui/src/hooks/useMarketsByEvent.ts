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

export function useMarketsByEvent(
  eventTicker: string,
  minCloseTs?: number,
  maxCloseTs?: number
) {
  return useQuery<MarketsResponse>({
    queryKey: ["markets-by-event", eventTicker, minCloseTs, maxCloseTs],
    queryFn: async () => {
      const params = new URLSearchParams({
        event_ticker: eventTicker,
        limit: "100", // Fetch all markets for the event
      });

      if (minCloseTs) {
        params.append("min_close_ts", minCloseTs.toString());
      }

      if (maxCloseTs) {
        params.append("max_close_ts", maxCloseTs.toString());
      }

      const response = await fetch(
        `${import.meta.env.VITE_API_URL || "http://localhost:8080"}/api/v1/markets/by-event?${params.toString()}`
      );

      if (!response.ok) {
        if (response.status === 404) {
          return { markets: [], cursor: "" };
        }
        throw new Error("Failed to fetch markets for event");
      }

      return response.json();
    },
    enabled: !!eventTicker,
  });
}
