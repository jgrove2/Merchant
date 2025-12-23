import { useQuery } from "@tanstack/react-query";

interface BalanceResponse {
  total_balance: number;
  currency: string;
  breakdown: {
    kalshi: number;
  };
}

const fetchBalance = async (): Promise<BalanceResponse> => {
  const apiUrl = import.meta.env.VITE_API_URL || "http://localhost:8080";
  const response = await fetch(`${apiUrl}/api/v1/balance`);
  
  if (!response.ok) {
    throw new Error("Failed to fetch balance");
  }
  
  return response.json();
};

export const useBalance = () => {
  return useQuery({
    queryKey: ["balance"],
    queryFn: fetchBalance,
    staleTime: 30000, // Consider data fresh for 30 seconds
    refetchInterval: 60000, // Refetch every 60 seconds
  });
};
