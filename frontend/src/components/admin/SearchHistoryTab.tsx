import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { adminService, SearchHistoryItem } from "@/services/admin.service";
import { useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, AlertTriangle } from "lucide-react";

interface SearchHistoryTabProps {
  token: string;
}

export function SearchHistoryTab({ token }: SearchHistoryTabProps) {
  const [loading, setLoading] = useState(true);
  const [fetching, setFetching] = useState(false);
  const [hasLoadedInitially, setHasLoadedInitially] = useState(false);
  const [history, setHistory] = useState<SearchHistoryItem[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalCount, setTotalCount] = useState(0);

  const limit = 50;
  const totalPages = Math.ceil(totalCount / limit);
  const maxPages = Math.min(50, totalPages);

  // Reset page to 1 when search query changes
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery]);

  // Load history when current page or query changes (with debounce)
  useEffect(() => {
    const isFirstLoad = !hasLoadedInitially;
    if (isFirstLoad) {
      setHasLoadedInitially(true);
    }

    const delayDebounceFn = setTimeout(() => {
      loadHistory(currentPage, searchQuery, isFirstLoad);
    }, isFirstLoad ? 0 : 300);

    return () => clearTimeout(delayDebounceFn);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPage, searchQuery]);

  const loadHistory = async (page: number, query: string, isInitialLoad: boolean) => {
    if (isInitialLoad) {
      setLoading(true);
    } else {
      setFetching(true);
    }
    try {
      const offset = (page - 1) * limit;
      const data = await adminService.getSearchHistory(token, limit, offset, query);
      setHistory(data.history || []);
      setTotalCount(data.total || 0);
    } catch (error) {
      console.error("Failed to load history:", error);
      alert("Failed to load search history");
    } finally {
      setLoading(false);
      setFetching(false);
    }
  };

  const handlePageChange = (page: number) => {
    if (page >= 1 && page <= maxPages) {
      setCurrentPage(page);
    }
  };

  const getPageNumbers = () => {
    const pages: (number | string)[] = [];
    const siblingCount = 1;

    // Always show page 1
    pages.push(1);

    const leftSiblingIndex = Math.max(currentPage - siblingCount, 2);
    const rightSiblingIndex = Math.min(currentPage + siblingCount, maxPages - 1);

    const showLeftSpill = leftSiblingIndex > 2;
    const showRightSpill = rightSiblingIndex < maxPages - 1;

    if (showLeftSpill) {
      pages.push("...");
    }

    for (let i = leftSiblingIndex; i <= rightSiblingIndex; i++) {
      pages.push(i);
    }

    if (showRightSpill) {
      pages.push("...");
    }

    if (maxPages > 1) {
      pages.push(maxPages);
    }

    return pages;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-4">
        <div>
          <h2 className="text-xl font-bold text-white">Search History</h2>
          <p className="text-sm text-gray-400 mt-1">
            Showing {history.length} of {totalCount} total searches
          </p>
        </div>
      </div>

      {/* 50 Pages Limit Banner */}
      {totalPages > 50 && (
        <div className="mb-4 bg-amber-500/10 border border-amber-500/30 rounded-lg p-3 flex items-center gap-3 text-amber-400 text-sm">
          <AlertTriangle className="h-4 w-4 flex-shrink-0" />
          <span>
            Showing up to 50 pages (2,500 searches) max. If you are looking for older records, please refine your query using the search box.
          </span>
        </div>
      )}

      <div className="mb-4">
        <Input
          placeholder="Search by user, email, or query..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-400"
        />
      </div>

      <div className="bg-[#2D1B4E] rounded-lg border border-gray-700 overflow-hidden relative">
        {/* Table opacity transition during page loads */}
        <div className={`overflow-x-auto transition-opacity duration-200 ${fetching ? "opacity-40" : "opacity-100"}`}>
          <table className="w-full">
            <thead className="bg-[#1a0f2e] border-b border-gray-700">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  User
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Query
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Results
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Searched At
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700">
              {history.map((item) => (
                <tr
                  key={item.id}
                  className="hover:bg-[#1a0f2e] transition-colors"
                >
                  <td className="px-4 py-3 text-sm">
                    <div>
                      <p className="text-white font-medium">{item.user_name || "Unknown User"}</p>
                      <p className="text-gray-400 text-xs">{item.user_email || "N/A"}</p>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    <code className="bg-[#1a0f2e] px-2 py-1 rounded text-xs break-all">
                      {item.query}
                    </code>
                  </td>
                  <td className="px-4 py-3 text-sm text-white">
                    {item.total_results}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    {new Date(item.searched_at).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {fetching && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/10">
            <Spinner size="md" />
          </div>
        )}
      </div>

      {history.length === 0 && (
        <div className="text-center py-12 text-gray-400">
          {searchQuery
            ? "No search history found matching your search"
            : "No search history found"}
        </div>
      )}

      {/* Pagination Controls */}
      {maxPages > 1 && (
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 mt-6 px-1">
          <div className="text-sm text-gray-400">
            Page <span className="text-white font-medium">{currentPage}</span> of{" "}
            <span className="text-white font-medium">{maxPages}</span>
          </div>

          <div className="flex items-center gap-1.5">
            {/* Previous Button */}
            <button
              onClick={() => handlePageChange(currentPage - 1)}
              disabled={currentPage === 1 || fetching}
              className={`p-2 rounded-md border text-sm transition-all ${
                currentPage === 1 || fetching
                  ? "bg-[#1a0f2e] border-gray-800 text-gray-600 cursor-not-allowed opacity-50"
                  : "bg-[#1a0f2e] border-gray-700 text-gray-300 hover:bg-[#2D1B4E] hover:text-white"
              }`}
              title="Previous Page"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>

            {/* Page Numbers */}
            {getPageNumbers().map((pageNum, idx) => {
              if (pageNum === "...") {
                return (
                  <span
                    key={`ellipsis-${idx}`}
                    className="px-3 py-1.5 text-gray-500 text-sm"
                  >
                    ...
                  </span>
                );
              }

              const isCurrent = pageNum === currentPage;
              return (
                <button
                  key={`page-${pageNum}`}
                  onClick={() => handlePageChange(pageNum as number)}
                  disabled={fetching}
                  className={`px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
                    isCurrent
                      ? "bg-purple-600 text-white shadow-[0_0_8px_rgba(147,51,234,0.4)]"
                      : "bg-[#1a0f2e] border border-gray-700 text-gray-300 hover:bg-[#2D1B4E] hover:text-white"
                  }`}
                >
                  {pageNum}
                </button>
              );
            })}

            {/* Next Button */}
            <button
              onClick={() => handlePageChange(currentPage + 1)}
              disabled={currentPage === maxPages || fetching}
              className={`p-2 rounded-md border text-sm transition-all ${
                currentPage === maxPages || fetching
                  ? "bg-[#1a0f2e] border-gray-800 text-gray-600 cursor-not-allowed opacity-50"
                  : "bg-[#1a0f2e] border-gray-700 text-gray-300 hover:bg-[#2D1B4E] hover:text-white"
              }`}
              title="Next Page"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
