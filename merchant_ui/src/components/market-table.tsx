import { useState, useEffect } from "react";
import { useEvents } from "@/hooks/useEvents";
import { useMarketsByEvent } from "@/hooks/useMarketsByEvent";
import { format } from "date-fns";
import type { DateRange } from "react-day-picker";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  ChevronUp,
  Calendar as CalendarIcon,
} from "lucide-react";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

function EventMarkets({
  eventTicker,
  minCloseTs,
  maxCloseTs,
}: {
  eventTicker: string;
  minCloseTs?: number;
  maxCloseTs?: number;
}) {
  const { data, isLoading, error } = useMarketsByEvent(
    eventTicker,
    minCloseTs,
    maxCloseTs
  );

  if (isLoading) {
    return (
      <div className="p-4 text-center text-sm text-muted-foreground">
        Loading markets...
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 text-center text-sm text-destructive">
        Error loading markets
      </div>
    );
  }

  if (!data?.markets || data.markets.length === 0) {
    return (
      <div className="p-4 text-center text-sm text-muted-foreground">
        No Valid Markets
      </div>
    );
  }

  return (
    <div className="rounded-md border p-2 bg-muted/50">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Market Title</TableHead>
            <TableHead>Subtitle</TableHead>
            <TableHead>Yes %</TableHead>
            <TableHead>No %</TableHead>
            <TableHead>Close Time</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.markets.map((market) => (
            <TableRow key={market.ticker}>
              <TableCell className="font-medium">{market.title}</TableCell>
              <TableCell className="max-w-md truncate">
                {market.yes_sub_title}
              </TableCell>
              <TableCell>{`$${market.yes_ask_dollars}`}</TableCell>
              <TableCell className="max-w-md truncate">{`$${market.no_ask_dollars}`}</TableCell>
              <TableCell>{market.close_time}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function EventRow({
  event,
  minCloseTs,
  maxCloseTs,
}: {
  event: any;
  minCloseTs?: number;
  maxCloseTs?: number;
}) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen} asChild>
      <>
        <TableRow className="cursor-pointer hover:bg-muted/50">
          <TableCell className="w-[30px]">
            <CollapsibleTrigger asChild>
              <Button variant="ghost" size="sm" className="h-6 w-6 p-0">
                {isOpen ? (
                  <ChevronUp className="h-4 w-4" />
                ) : (
                  <ChevronDown className="h-4 w-4" />
                )}
                <span className="sr-only">Toggle markets</span>
              </Button>
            </CollapsibleTrigger>
          </TableCell>
          <TableCell className="font-medium" onClick={() => setIsOpen(!isOpen)}>
            {event.title}
          </TableCell>
          <TableCell onClick={() => setIsOpen(!isOpen)}>
            {event.category}
          </TableCell>
          <TableCell onClick={() => setIsOpen(!isOpen)}>
            {event.sub_title}
          </TableCell>
        </TableRow>
        <CollapsibleContent asChild>
          <TableRow>
            <TableCell colSpan={4} className="p-0">
              <div className="p-4">
                <EventMarkets
                  eventTicker={event.event_ticker}
                  minCloseTs={minCloseTs}
                  maxCloseTs={maxCloseTs}
                />
              </div>
            </TableCell>
          </TableRow>
        </CollapsibleContent>
      </>
    </Collapsible>
  );
}

export default function MarketTable() {
  const [limit, setLimit] = useState(10);
  const [cursors, setCursors] = useState<string[]>([""]);
  const [currentPageIndex, setCurrentPageIndex] = useState(0);

  // Date range state - default to 12 hours from now and 7 days from now
  const [dateRange, setDateRange] = useState<DateRange | undefined>(() => {
    const from = new Date();
    from.setHours(from.getHours() + 12);
    const to = new Date();
    to.setDate(to.getDate() + 7);
    return { from, to };
  });

  // Convert dates to Unix timestamps for API
  const minCloseTs = dateRange?.from
    ? Math.floor(dateRange.from.getTime() / 1000)
    : undefined;
  const maxCloseTs = dateRange?.to
    ? Math.floor(dateRange.to.getTime() / 1000)
    : undefined;

  const { data, isLoading, error, refetch } = useEvents({
    limit,
    cursor: cursors[currentPageIndex],
  });

  // Track if there's a next page available
  const hasNextPage = data?.cursor && data.cursor !== "";
  const hasPrevPage = currentPageIndex > 0;

  // When we get new data with a cursor, store it for potential forward navigation
  useEffect(() => {
    if (data?.cursor && data.cursor !== "") {
      setCursors((prev) => {
        const newCursors = [...prev];
        // Only add the cursor if we don't already have it at the next position
        if (newCursors[currentPageIndex + 1] !== data.cursor) {
          newCursors[currentPageIndex + 1] = data.cursor;
        }
        return newCursors;
      });
    }
  }, [data?.cursor, currentPageIndex]);

  const handleLimitChange = (value: string) => {
    setLimit(Number(value));
    setCursors([""]);
    setCurrentPageIndex(0);
  };

  const handleDateRangeChange = (range: DateRange | undefined) => {
    if (range) {
      setDateRange(range);
      // We don't need to reset pagination for events as the date filter applies to markets within events
    }
  };

  const handleNextPage = () => {
    if (hasNextPage) {
      setCurrentPageIndex((prev) => prev + 1);
    }
  };

  const handlePrevPage = () => {
    if (hasPrevPage) {
      setCurrentPageIndex((prev) => prev - 1);
    }
  };

  return (
    <div className="mx-auto w-full max-w-7xl space-y-8">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Events</h1>
          <p className="text-muted-foreground">
            Browse events and their markets
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Show:</span>
            <Select value={limit.toString()} onValueChange={handleLimitChange}>
              <SelectTrigger className="w-[80px]">
                <SelectValue placeholder="10" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="5">5</SelectItem>
                <SelectItem value="10">10</SelectItem>
                <SelectItem value="25">25</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button onClick={() => refetch()} variant="outline">
            Refresh
          </Button>
        </div>
      </header>

      {/* Date Range Filters */}
      <Card className="p-6">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">Market Close Date Range:</span>
            <span className="text-xs text-muted-foreground">
              (Filters markets within events)
            </span>
          </div>
          <Popover>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                className="w-[300px] justify-start text-left font-normal"
              >
                <CalendarIcon className="mr-2 h-4 w-4" />
                {dateRange?.from ? (
                  dateRange.to ? (
                    <>
                      {format(dateRange.from, "LLL dd, y")} -{" "}
                      {format(dateRange.to, "LLL dd, y")}
                    </>
                  ) : (
                    format(dateRange.from, "LLL dd, y")
                  )
                ) : (
                  <span>Pick a date range</span>
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0" align="start">
              <Calendar
                mode="range"
                defaultMonth={dateRange?.from}
                selected={dateRange}
                onSelect={handleDateRangeChange}
                numberOfMonths={2}
                disabled={(date) =>
                  date < new Date(new Date().setHours(0, 0, 0, 0))
                }
              />
            </PopoverContent>
          </Popover>
        </div>
      </Card>

      <Card className="p-6">
        {isLoading && (
          <div className="flex items-center justify-center py-8">
            <p className="text-muted-foreground">Loading events...</p>
          </div>
        )}

        {error && (
          <div className="flex items-center justify-center py-8">
            <p className="text-destructive">
              Error loading events: {error.message}
            </p>
          </div>
        )}

        {data && data.events && (
          <div className="space-y-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[30px]"></TableHead>
                  <TableHead>Event Title</TableHead>
                  <TableHead>Category</TableHead>
                  <TableHead>Subtitle</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.events.map((event) => (
                  <EventRow
                    key={event.event_ticker}
                    event={event}
                    minCloseTs={minCloseTs}
                    maxCloseTs={maxCloseTs}
                  />
                ))}
              </TableBody>
            </Table>

            {data.events.length === 0 && (
              <div className="flex items-center justify-center py-8">
                <p className="text-muted-foreground">No events found</p>
              </div>
            )}

            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Showing {data.events.length} events per page
              </p>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handlePrevPage}
                  disabled={!hasPrevPage}
                >
                  <ChevronLeft className="h-4 w-4" />
                  Previous
                </Button>
                <span className="text-sm text-muted-foreground px-2">
                  Page {currentPageIndex + 1}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleNextPage}
                  disabled={!hasNextPage}
                >
                  Next
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
