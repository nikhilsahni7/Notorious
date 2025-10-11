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
import {
  AlertCircle,
  History,
  LogOut,
  RotateCcw,
  Search as SearchIcon,
  User,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const PAGE_SIZE = 50;
const LAST_SEARCH_KEY = "notorious_last_search";

interface LastSearchData {
  fields: SearchFields;
  operator: SearchOperator;
  results: Person[];
  totalResults: number;
  searchTime: number;
  timestamp: number;
}

export default function SearchPage() {
  const { user, token, logout, isLoading, updateUser } = useAuth();
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
  const [isDuplicateSearch, setIsDuplicateSearch] = useState(false);

  const filteredResults = useClientFilter(results, clientSearchQuery);

  // Load last search on mount
  useEffect(() => {
    if (!isLoading && token) {
      const savedSearch = localStorage.getItem(LAST_SEARCH_KEY);
      if (savedSearch) {
        try {
          const lastSearch: LastSearchData = JSON.parse(savedSearch);
          setSearchFields(lastSearch.fields);
          setOperator(lastSearch.operator);
          setResults(lastSearch.results || []);
          setTotalResults(lastSearch.totalResults || 0);
          setSearchTime(lastSearch.searchTime || 0);
        } catch (error) {
          console.error("Failed to load last search:", error);
        }
      }
    }
  }, [isLoading, token]);

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

  const resetSearch = () => {
    setSearchFields({
      id: "",
      name: "",
      fname: "",
      mobile: "",
      alt: "",
      email: "",
      address: "",
    });
    setResults([]);
    setTotalResults(0);
    setClientSearchQuery("");
    setCurrentPage(1);
    setSearchTime(0);
    setIsDuplicateSearch(false);
    localStorage.removeItem(LAST_SEARCH_KEY);
  };

  const executeSearch = async (page: number = 1) => {
    setClientSearchQuery("");
    setIsDuplicateSearch(false);

    try {
      const data = await performSearch(searchFields, operator, page, PAGE_SIZE);
      setResults(data.results || []);
      setTotalResults(data.total || 0);
      setSearchTime(data.took_ms || 0);
      setIsDuplicateSearch(data.is_duplicate || false);

      // Update search limits in real-time
      if (data.searches_used_today !== undefined) {
        setSearchesUsed(data.searches_used_today);
        // Update user in context
        if (user) {
          updateUser({
            ...user,
            searches_used_today: data.searches_used_today,
          });
        }
      }
      if (data.daily_search_limit !== undefined) {
        setSearchLimit(data.daily_search_limit);
      }

      // Save last search with results to localStorage
      const lastSearchData: LastSearchData = {
        fields: searchFields,
        operator,
        results: data.results || [],
        totalResults: data.total || 0,
        searchTime: data.took_ms || 0,
        timestamp: Date.now(),
      };
      localStorage.setItem(LAST_SEARCH_KEY, JSON.stringify(lastSearchData));

      setCurrentPage(page);
    } catch (error) {
      console.error("Search failed:", error);
      alert(error instanceof Error ? error.message : "Search failed");
    }
  };

  const handleCopy = (person: Person, index: number) => {
    const text = formatPersonForClipboard(person);
    copyToClipboard(text);
    setCopiedIndex(index);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  const handleNextPage = () => {
    if (currentPage < totalPages) {
      executeSearch(currentPage + 1);
    }
  };

  const handlePrevPage = () => {
    if (currentPage > 1) {
      executeSearch(currentPage - 1);
    }
  };

  if (isLoading || !token) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  const percentageUsed =
    searchLimit > 0 ? (searchesUsed / searchLimit) * 100 : 0;
  const searchesRemaining = searchLimit - searchesUsed;
  const totalPages = Math.ceil(totalResults / PAGE_SIZE);
  const startIndex = (currentPage - 1) * PAGE_SIZE;
  const endIndex = Math.min(startIndex + PAGE_SIZE, totalResults);

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-3">
      <div className="max-w-[1800px] mx-auto">
        {/* Compact Header */}
        <div className="flex justify-between items-center mb-3 bg-[#1a0f2e] p-3 rounded-lg border border-gray-700">
          <div className="flex items-center gap-4">
            <div>
              <h1 className="text-xl font-bold text-white">Notorious Search</h1>
              <div className="text-xs text-gray-400 mt-0.5">
                {user?.name} • {user?.email}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {/* Search Limit - Compact */}
            <div className="text-right">
              <div className="text-xs text-gray-400">Daily Limit</div>
              <div className="text-sm font-bold text-white">
                {searchesUsed} / {searchLimit}
                <span
                  className={`ml-1 text-xs ${
                    percentageUsed > 90
                      ? "text-red-400"
                      : percentageUsed > 70
                      ? "text-yellow-400"
                      : "text-green-400"
                  }`}
                >
                  ({searchesRemaining} left)
                </span>
              </div>
              <div className="w-32 h-1.5 bg-gray-700 rounded-full mt-1 overflow-hidden">
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

            <Button
              onClick={() => router.push("/profile")}
              variant="outline"
              size="sm"
              className="bg-transparent border-purple-500 text-purple-400 hover:bg-purple-500/10"
            >
              <User className="h-4 w-4 mr-1" />
              Profile
            </Button>

            <Button
              onClick={() => router.push("/history")}
              variant="outline"
              size="sm"
              className="bg-transparent border-blue-500 text-blue-400 hover:bg-blue-500/10"
            >
              <History className="h-4 w-4 mr-1" />
              History
            </Button>

            <Button
              onClick={() => router.push("/password-change")}
              variant="outline"
              size="sm"
              className="bg-transparent border-orange-500 text-orange-400 hover:bg-orange-500/10"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-4 w-4 mr-1"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
                />
              </svg>
              Password
            </Button>

            {user?.role === "admin" && (
              <Button
                onClick={() => router.push("/admin")}
                variant="outline"
                size="sm"
                className="bg-transparent border-purple-500 text-purple-400 hover:bg-purple-500/10"
              >
                Admin
              </Button>
            )}

            <Button
              onClick={logout}
              variant="outline"
              size="sm"
              className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E]"
            >
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {searchesUsed >= searchLimit && (
          <div className="mb-3 bg-red-500/10 border border-red-500 text-red-400 p-3 rounded-lg text-sm">
            <strong>Daily limit reached!</strong> You&apos;ve used all{" "}
            {searchLimit} searches. Resets at 12 AM IST.
          </div>
        )}

        {isDuplicateSearch && (
          <div className="mb-3 bg-blue-500/10 border border-blue-500 text-blue-400 p-3 rounded-lg text-sm flex items-center gap-2">
            <AlertCircle className="h-4 w-4" />
            <span>
              <strong>Duplicate search detected!</strong> This is the same query
              as your last search, so your search count was not decremented.
            </span>
          </div>
        )}

        {/* Compact Search Form */}
        <div className="bg-[#1a0f2e] p-3 rounded-lg border border-gray-700 mb-3">
          <SearchForm
            searchFields={searchFields}
            operator={operator}
            onFieldChange={updateField}
            onOperatorChange={setOperator}
            onSearch={() => executeSearch(1)}
          />

          <div className="flex gap-2 mt-3">
            <Button
              onClick={() => executeSearch(1)}
              disabled={loading || searchesUsed >= searchLimit}
              className="flex-1 bg-pink-500 hover:bg-pink-600 text-white"
            >
              {loading ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Searching...
                </>
              ) : (
                <>
                  <SearchIcon className="h-4 w-4 mr-2" />
                  Search
                </>
              )}
            </Button>
            <Button
              onClick={resetSearch}
              variant="outline"
              className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E]"
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Reset
            </Button>
          </div>
        </div>

        {/* Results - Compact */}
        {totalResults > 0 && (
          <>
            <div className="mb-3">
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

            <div className="mt-3">
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                onPrevPage={handlePrevPage}
                onNextPage={handleNextPage}
              />
            </div>
          </>
        )}

        {/* No Results Message */}
        {!loading && totalResults === 0 && (
          <div className="bg-[#1a0f2e] p-12 rounded-lg border border-gray-700 text-center">
            <SearchIcon className="h-16 w-16 mx-auto mb-4 text-gray-600" />
            <h3 className="text-xl font-semibold text-white mb-2">
              No Results Found
            </h3>
            <p className="text-gray-400">
              Try different search terms or use the OR operator for broader
              results
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
