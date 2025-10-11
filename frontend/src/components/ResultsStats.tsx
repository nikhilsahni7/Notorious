import { Input } from "@/components/ui/input";

interface ResultsStatsProps {
  startIndex: number;
  endIndex: number;
  totalResults: number;
  searchTime: number;
  filterQuery: string;
  onFilterChange: (value: string) => void;
}

export function ResultsStats({
  startIndex,
  endIndex,
  totalResults,
  searchTime,
  filterQuery,
  onFilterChange,
}: ResultsStatsProps) {
  return (
    <div className="text-sm text-gray-300 flex items-center justify-between">
      <div>
        Showing{" "}
        <span className="text-white font-bold">
          {startIndex + 1}-{endIndex}
        </span>{" "}
        of{" "}
        <span className="text-white font-bold">
          {totalResults.toLocaleString()}
        </span>{" "}
        results
        <span className="ml-3 text-xs bg-green-500/20 text-green-400 px-2 py-1 rounded">
          {searchTime}ms
        </span>
      </div>
      <Input
        placeholder="Filter results..."
        value={filterQuery}
        onChange={(e) => onFilterChange(e.target.value)}
        className="w-64 bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
      />
    </div>
  );
}
