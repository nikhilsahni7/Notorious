"use client";

import { Pagination } from "@/components/Pagination";
import { ResultsStats } from "@/components/ResultsStats";
import { ResultsTable } from "@/components/ResultsTable";
import { SearchForm } from "@/components/SearchForm";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useAuth } from "@/contexts/AuthContext";
import { useClientFilter } from "@/hooks/useClientFilter";
import { useSearch } from "@/hooks/useSearch";
import { Person, SearchFields, SearchOperator } from "@/types/person";
import { copyToClipboard, formatPersonForClipboard } from "@/utils/clipboard";
import { LogOut, Search } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const PAGE_SIZE = 50;

export default function SearchPage() {
  const { user, token, logout, isLoading } = useAuth();
  const router = useRouter();
  const { performSearch, loading } = useSearch(token);

  const [searchFields, setSearchFields] = useState<SearchFields>({
    id: "",
    name: "",
    fname: "",
    mobile: "",
    alt: "",
    email: "",
    address: "",
  });
  const [operator, setOperator] = useState<SearchOperator>("AND");
  const [results, setResults] = useState<Person[]>([]);
  const [totalResults, setTotalResults] = useState(0);
  const [searchTime, setSearchTime] = useState(0);
  const [clientSearchQuery, setClientSearchQuery] = useState("");
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [searchesUsed, setSearchesUsed] = useState(0);
  const [searchLimit, setSearchLimit] = useState(0);

  const filteredResults = useClientFilter(results, clientSearchQuery);

  useEffect(() => {
    if (!isLoading && !token) {
      router.push("/login");
    }
  }, [isLoading, token, router]);

  useEffect(() => {
    if (user) {
      setSearchesUsed(user.searches_used_today);
      setSearchLimit(user.daily_search_limit);
    }
  }, [user]);

  const updateField = (field: keyof SearchFields, value: string) => {
    setSearchFields((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const executeSearch = async (page: number = 1) => {
    setClientSearchQuery("");

    try {
      const data = await performSearch(searchFields, operator, page, PAGE_SIZE);
      setResults(data.results || []);
      setTotalResults(data.total || 0);
      setSearchTime(data.took_ms || 0);

      if (data.searches_used_today !== undefined) {
        setSearchesUsed(data.searches_used_today);
      }
      if (data.daily_search_limit !== undefined) {
        setSearchLimit(data.daily_search_limit);
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes("Session expired")) {
        logout();
        router.push("/login");
        return;
      }

      alert(
        `Search failed: ${
          error instanceof Error ? error.message : "Unknown error"
        }`
      );
      setResults([]);
      setTotalResults(0);
    }
  };

  const handleSearch = async () => {
    setCurrentPage(1);
    await executeSearch(1);
  };

  const handleCopy = async (person: Person, index: number) => {
    const text = formatPersonForClipboard(person);
    await copyToClipboard(text);
    setCopiedIndex(index);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  const totalPages = Math.ceil(totalResults / PAGE_SIZE);
  const startIndex = (currentPage - 1) * PAGE_SIZE;
  const endIndex = Math.min(startIndex + filteredResults.length, totalResults);

  const handleNextPage = async () => {
    if (currentPage < totalPages) {
      const nextPage = currentPage + 1;
      setCurrentPage(nextPage);
      await executeSearch(nextPage);
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  };

  const handlePrevPage = async () => {
    if (currentPage > 1) {
      const prevPage = currentPage - 1;
      setCurrentPage(prevPage);
      await executeSearch(prevPage);
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!token) {
    return null;
  }

  const searchesRemaining = searchLimit - searchesUsed;
  const percentageUsed =
    searchLimit > 0 ? (searchesUsed / searchLimit) * 100 : 0;

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-6">
      <div className="max-w-[1800px] mx-auto">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white">Notorious Search</h1>
            <div className="text-sm space-y-1 mt-1">
              <p className="text-gray-300">
                <span className="text-gray-500">Signed in as:</span>{" "}
                <span className="text-white">{user?.name}</span>
              </p>
              <p className="text-gray-300">
                <span className="text-gray-500">Email:</span>{" "}
                <span className="text-white">{user?.email}</span>
              </p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-right">
              <div className="text-sm text-gray-400">Search Limit</div>
              <div className="text-lg font-bold text-white">
                {searchesUsed} / {searchLimit}
                <span
                  className={`ml-2 text-sm ${
                    percentageUsed > 90
                      ? "text-red-400"
                      : percentageUsed > 70
                      ? "text-yellow-400"
                      : "text-green-400"
                  }`}
                >
                  ({searchesRemaining} remaining)
                </span>
              </div>
              <div className="w-48 h-2 bg-gray-700 rounded-full mt-1 overflow-hidden">
                <div
                  className={`h-full transition-all ${
                    percentageUsed > 90
                      ? "bg-red-500"
                      : percentageUsed > 70
                      ? "bg-yellow-500"
                      : "bg-green-500"
                  }`}
                  style={{ width: `${Math.min(percentageUsed, 100)}%` }}
                />
              </div>
            </div>
            {user?.role === "admin" && (
              <Button
                onClick={() => router.push("/admin")}
                variant="outline"
                className="bg-transparent border-purple-500 text-purple-400 hover:bg-purple-500/10"
              >
                Admin Dashboard
              </Button>
            )}
            <Button
              onClick={logout}
              variant="outline"
              className="bg-transparent border-gray-600 text-white hover:bg-[#1a0f2e]"
            >
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        </div>

        {searchesUsed >= searchLimit && (
          <div className="mb-4 bg-red-500/10 border border-red-500 text-red-400 p-4 rounded-lg">
            <strong>Daily limit reached!</strong> You have used all{" "}
            {searchLimit} searches for today. Your limit will reset at 12:00 AM
            IST.
          </div>
        )}

        <SearchForm
          searchFields={searchFields}
          operator={operator}
          loading={loading || searchesUsed >= searchLimit}
          onFieldChange={updateField}
          onOperatorChange={setOperator}
          onSearch={handleSearch}
        />

        {results.length > 0 && (
          <>
            <div className="mb-4 space-y-3">
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPrevPage={handlePrevPage}
                onNextPage={handleNextPage}
              />
              <ResultsStats
                startIndex={startIndex}
                endIndex={endIndex}
                totalResults={totalResults}
                searchTime={searchTime}
                filterQuery={clientSearchQuery}
                onFilterChange={setClientSearchQuery}
              />
            </div>

            <ResultsTable
              results={filteredResults}
              copiedIndex={copiedIndex}
              onCopy={handleCopy}
            />

            <div className="mt-4">
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPrevPage={handlePrevPage}
                onNextPage={handleNextPage}
              />
            </div>

            {filteredResults.length === 0 && Boolean(clientSearchQuery) && (
              <div className="p-8 text-center bg-[#1a0f2e] rounded-lg">
                <p className="text-gray-400">
                  No results match your filter &quot;{clientSearchQuery}&quot;
                </p>
              </div>
            )}
          </>
        )}

        {!loading && results.length === 0 && (
          <div className="p-12 text-center bg-[#1a0f2e] rounded-lg">
            <Search className="h-12 w-12 mx-auto mb-4 text-gray-500" />
            <h3 className="text-lg font-semibold text-white mb-2">
              No Search Results
            </h3>
            <p className="text-gray-400">
              Enter your search criteria above and click Search
            </p>
          </div>
        )}

        {loading && (
          <div className="p-12 text-center bg-[#1a0f2e] rounded-lg">
            <Spinner size="lg" className="mx-auto mb-4" />
            <p className="text-gray-400">Searching database...</p>
          </div>
        )}
      </div>
    </div>
  );
}
