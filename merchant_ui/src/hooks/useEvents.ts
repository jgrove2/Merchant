import { useQuery } from "@tanstack/react-query";

export interface Event {
  event_ticker: string;
  title: string;
  sub_title: string;
  category: string;
  mutually_exclusive: boolean;
  series_ticker: string;
  strike_period: string;
  available_on_brokers: boolean;
  collateral_return_type: string;
}

interface EventsResponse {
  events: Event[];
  cursor: string;
  milestones: any[];
}

interface UseEventsOptions {
  limit: number;
  cursor?: string;
  status?: string;
}

export function useEvents({ limit, cursor }: UseEventsOptions) {
  return useQuery<EventsResponse>({
    queryKey: ["events", limit, cursor],
    queryFn: async () => {
      const params = new URLSearchParams({
        limit: limit.toString(),
      });

      if (cursor) {
        params.append("cursor", cursor);
      }

      const response = await fetch(
        `${import.meta.env.VITE_API_URL || "http://localhost:8080"}/api/v1/events?${params.toString()}`
      );

      if (!response.ok) {
        let errorMessage = "Failed to fetch events";
        try {
          const errorData = await response.json();
          if (errorData.error) {
            errorMessage = errorData.error;
          }
        } catch (e) {
          // Ignore JSON parse error
        }
        throw new Error(errorMessage);
      }

      return response.json();
    },
  });
}
