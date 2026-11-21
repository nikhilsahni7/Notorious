"use client";

import { triggerFestiveQuickBlast } from "@/components/FestiveCelebration";
import { KnotoriousBrand } from "@/components/KnotoriousBrand";
import { NotificationBell } from "@/components/NotificationBell";
import { ChatNotificationBadge } from "@/components/chat/ChatNotificationBadge";
import { LatestAnnouncement } from "@/components/chat/LatestAnnouncement";
import { Pagination } from "@/components/Pagination";
import { RefineSearch } from "@/components/RefineSearch";
import { ResultsStats } from "@/components/ResultsStats";
import { ResultsTable } from "@/components/ResultsTable";
import { SearchForm } from "@/components/SearchForm";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useAuth } from "@/contexts/AuthContext";
import { useIstCountdown } from "@/hooks/useIstCountdown";
import { useSearch } from "@/hooks/useSearch";
import { notificationService } from "@/services/notification.service";
import { searchService } from "@/services/search.service";
import { optInToPush } from "@/lib/push";
import { ApiError } from "@/lib/api-client";
import {
    Person,
    Refinement,
    SearchFields,
    SearchOperator,
} from "@/types/person";
import {
    copyToClipboard,
    formatPersonForClipboard,
    formatPersonsForClipboard,
} from "@/utils/clipboard";
import {
  AlertCircle,
  Bell,
  Download,
  History,
  LogOut,
  MapPin,
  RotateCcw,
  Search as SearchIcon,
  User,
} from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const PAGE_SIZE = 50;
const LAST_SEARCH_KEY = "knotorious_last_search";
const DEFAULT_SEARCH_FIELDS: SearchFields = {
  id: "",
  oid: "",
  name: "",
  fname: "",
  mobile: "",
  alt: "",
  email: "",
  address: "",
};

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
  const istClock = useIstCountdown();

  const [searchFields, setSearchFields] = useState<SearchFields>({
    ...DEFAULT_SEARCH_FIELDS,
  });
  const [operator, setOperator] = useState<SearchOperator>("AND");
  const [results, setResults] = useState<Person[]>([]);
  const [totalResults, setTotalResults] = useState(0);
  const [searchTime, setSearchTime] = useState(0);
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);
  const [isCopyingAll, setIsCopyingAll] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [searchesUsed, setSearchesUsed] = useState(0);
  const [searchLimit, setSearchLimit] = useState(0);
  const [isDuplicateSearch, setIsDuplicateSearch] = useState(false);
  const [limitOptedIn, setLimitOptedIn] = useState<boolean | null>(null);
  const [optInProcessing, setOptInProcessing] = useState(false);

  // Refinement state
  const [activeRefinements, setActiveRefinements] = useState<Refinement[]>([]);
  const [isRefining, setIsRefining] = useState(false);
  const [baseQuery, setBaseQuery] = useState("");
  const [baseOperator, setBaseOperator] = useState<SearchOperator>("AND");
  const [isRefinedView, setIsRefinedView] = useState(false);

  // Load last search on mount
  useEffect(() => {
    if (!isLoading && token) {
      const savedSearch = localStorage.getItem(LAST_SEARCH_KEY);
      if (savedSearch) {
        try {
          const lastSearch: LastSearchData = JSON.parse(savedSearch);
          setSearchFields({
            ...DEFAULT_SEARCH_FIELDS,
            ...lastSearch.fields,
            oid: lastSearch.fields.oid ?? "",
          });
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

  // Load notification prefs once so the "Notify me on reset" state is accurate
  // even before a 429 is returned.
  useEffect(() => {
    if (!token) return;
    notificationService
      .getPrefs(token)
      .then((data) => setLimitOptedIn(data.prefs.quota_reset === true))
      .catch(() => {
        // unknown is fine — affordance just stays available
      });
  }, [token]);

  // Explicit opt-in trigger point: permission is requested ONLY on this tap
  // (per browser push permission UX rules), never on load or login.
  const handleNotifyOptIn = async () => {
    if (!token || optInProcessing) return;
    setOptInProcessing(true);
    try {
      await notificationService.updatePrefs({ quota_reset: true }, token);
      setLimitOptedIn(true);
      // Browser-level push: degrades silently to in-app only on failure/denial
      await optInToPush(token);
    } catch (error) {
      console.error("Failed to opt in to reset notification:", error);
      // Prefs save failed — don't latch the checkmark
      setLimitOptedIn(false);
    } finally {
      setOptInProcessing(false);
    }
  };

  const updateField = (field: keyof SearchFields, value: string) => {
    setSearchFields((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const resetSearch = () => {
    setSearchFields({ ...DEFAULT_SEARCH_FIELDS });
    setResults([]);
    setTotalResults(0);
    setCurrentPage(1);
    setSearchTime(0);
    setIsDuplicateSearch(false);
    setActiveRefinements([]);
    setIsRefinedView(false);
    setBaseQuery("");
    localStorage.removeItem(LAST_SEARCH_KEY);
  };

  const executeSearch = async (page: number = 1) => {
    setIsDuplicateSearch(false);
    setActiveRefinements([]);
    setIsRefinedView(false);

    try {
      const data = await performSearch(searchFields, operator, page, PAGE_SIZE);
      setResults(data.results || []);
      setTotalResults(data.total || 0);
      setSearchTime(data.took_ms || 0);
      setIsDuplicateSearch(data.is_duplicate || false);

      // Trigger festive blast for successful searches
      if (data.results && data.results.length > 0) {
        triggerFestiveQuickBlast();
      }

      // Store base query for refinement
      const queries: string[] = [];
      Object.entries(searchFields).forEach(([field, value]) => {
        if (value.trim()) {
          queries.push(`${field}:${value.trim()}`);
        }
      });
      setBaseQuery(queries.join(` ${operator} `));
      setBaseOperator(operator);

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
        fields: { ...searchFields },
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
      if (error instanceof ApiError && error.status === 429) {
        // Distinguish the daily-limit 429 (carries daily_search_limit) from
        // the per-minute SearchRateLimiter 429 (no limit fields) — only the
        // former should latch the "Daily limit reached!" state.
        const data = error.data as Record<string, unknown> | undefined;
        if (typeof data?.daily_search_limit === "number") {
          setSearchLimit(data.daily_search_limit);
          if (typeof data.searches_used_today === "number") {
            setSearchesUsed(data.searches_used_today);
          }
          // Only latch the opt-in when unknown — never clobber an opt-in
          // state already managed locally (e.g. from the profile page).
          const notifyOptIn = data.notify_opt_in;
          if (typeof notifyOptIn === "boolean" && limitOptedIn === null) {
            setLimitOptedIn(notifyOptIn);
          }
          alert(data.error as string || "Daily search limit exceeded");
          return;
        }
        alert(error.data?.error || "Search failed");
        return;
      }
      alert(error instanceof Error ? error.message : "Search failed");
    }
  };

  const handleRefine = async () => {
    if (!token || activeRefinements.length === 0) return;

    setIsRefining(true);
    try {
      const data = await searchService.refineSearch(
        {
          base_query: baseQuery,
          base_operator: baseOperator,
          refinements: activeRefinements,
          refinement_operator: "AND",
          size: PAGE_SIZE,
          from: 0,
        },
        token
      );

      setResults(data.results || []);
      setTotalResults(data.total || 0);
      setSearchTime(data.took_ms || 0);
      setCurrentPage(1);
      setIsRefinedView(true);

      // Refinement doesn't consume search credits
      if (data.searches_used_today !== undefined) {
        setSearchesUsed(data.searches_used_today);
      }
    } catch (error) {
      console.error("Refine failed:", error);
      alert(error instanceof Error ? error.message : "Refine failed");
    } finally {
      setIsRefining(false);
    }
  };

  const handleAddRefinement = (refinement: Refinement) => {
    setActiveRefinements((prev) => [...prev, refinement]);
  };

  const handleRemoveRefinement = (index: number) => {
    setActiveRefinements((prev) => prev.filter((_, i) => i !== index));
  };

  const handleClearRefinements = () => {
    setActiveRefinements([]);
    setIsRefinedView(false);
    // Re-execute original search
    executeSearch(1);
  };

  const handleCopy = (person: Person, index: number) => {
    const text = formatPersonForClipboard(person);
    copyToClipboard(text);
    setCopiedIndex(index);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  const handleCopyAll = () => {
    const text = formatPersonsForClipboard(results);
    copyToClipboard(text);
    setIsCopyingAll(true);
    setTimeout(() => setIsCopyingAll(false), 2000);
  };

  const handleExportEOD = async () => {
    try {
      // Sanitize NEXT_PUBLIC_API_URL which may include quotes in .env
      let base = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
      base = base.replace(/^"|"$/g, "");
      const exportUrl = `${base.replace(/\/$/, "")}/search/export-eod`;

      const response = await fetch(exportUrl, {
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to export EOD report");
      }

      // Get the filename from Content-Disposition header or use default
      const contentDisposition = response.headers.get("Content-Disposition");
      let filename = "EOD_Report.csv";
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename=(.+)/);
        if (filenameMatch) {
          filename = filenameMatch[1];
        }
      }

      // Download the CSV file
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error("Export failed:", error);
      alert(error instanceof Error ? error.message : "Export failed");
    }
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
      <div className="min-h-screen bg-[#040f25] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  const percentageUsed =
    searchLimit > 0 ? (searchesUsed / searchLimit) * 100 : 0;
  const limitReached = searchLimit > 0 && searchesUsed >= searchLimit;
  const searchesRemaining = searchLimit - searchesUsed;
  const totalPages = Math.ceil(totalResults / PAGE_SIZE);
  const startIndex = (currentPage - 1) * PAGE_SIZE;
  const endIndex = Math.min(startIndex + PAGE_SIZE, totalResults);

  return (
    <div className="min-h-screen bg-[#040f25] p-3 pb-24 md:pb-6">
      <div className="max-w-[1800px] mx-auto">
        {/* Product header */}
        <div className="mb-3 flex flex-wrap items-center gap-4 rounded-2xl border border-sky-900/50 bg-[#0a1c38] px-4 py-4 md:px-5">
          <KnotoriousBrand subtitle={user ? `${user.name} · ${user.email}` : "Search dashboard"} className="max-w-full shrink-0" />
          <h1 className="sr-only">Knotorious Search</h1>

          <div className="flex flex-1 items-center justify-end md:justify-start">
            <Link href="/find-my-location" className="inline-flex items-center gap-2 rounded-lg border border-sky-800/70 bg-[#102342] px-3 py-2 text-sm text-sky-200 transition-colors hover:border-sky-500 hover:bg-sky-900/40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-300">
              <MapPin className="h-4 w-4 text-red-400 motion-safe:animate-[location-pin_2.4s_ease-in-out_infinite]" aria-hidden="true" />
              Find my location
            </Link>
          </div>

          <div className="flex w-full flex-wrap items-center justify-end gap-2 md:gap-3 xl:w-auto">
            <NotificationBell />

            {/* Search Limit - Compact */}
            <div className="text-right mr-2">
              <div className="text-[10px] md:text-xs text-gray-400">
                Daily Limit
              </div>
              <div className="text-xs md:text-sm font-bold text-white">
                {searchesUsed} / {searchLimit}
                <span
                  className={`ml-1 text-[10px] md:text-xs ${
                    percentageUsed > 90
                      ? "text-red-400"
                      : percentageUsed > 70
                      ? "text-yellow-400"
                      : "text-green-400"
                  }`}
                >
                  ({searchesRemaining})
                </span>
              </div>
              <div className="w-24 md:w-32 h-2 bg-white/5 rounded-full mt-1.5 overflow-hidden ml-auto border border-white/5 shadow-inner">
                <div
                  className={`h-full transition-all duration-500 relative ${
                    percentageUsed > 90
                      ? "bg-gradient-to-r from-red-500 to-rose-600 shadow-[0_0_10px_rgba(239,68,68,0.4)]"
                      : percentageUsed > 70
                      ? "bg-gradient-to-r from-yellow-400 to-amber-500 shadow-[0_0_10px_rgba(245,158,11,0.4)]"
                      : "bg-gradient-to-r from-green-400 to-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.4)]"
                  }`}
                  style={{ width: `${Math.min(percentageUsed, 100)}%` }}
                />
              </div>
              {/* Live IST clock + reset countdown (locks to IST regardless of device timezone) */}
              {istClock && (
                <div className="text-[10px] text-gray-500 mt-0.5">
                  Now {istClock.nowLabel} IST · Resets in {istClock.resetsIn}
                </div>
              )}
            </div>

            <Button
              onClick={() => router.push("/profile")}
              variant="outline"
              size="sm"
              className="h-8 px-3 bg-transparent border-sky-800 text-sky-200 hover:bg-sky-900/50 hover:border-sky-500 hover:text-white transition-all duration-300 hover:-translate-y-[1px]"
            >
              <User className="h-4 w-4" />
              <span className="hidden md:inline ml-1">Profile</span>
            </Button>

            <Button
              onClick={() => router.push("/history")}
              variant="outline"
              size="sm"
              className="h-8 px-3 bg-transparent border-blue-500/50 text-blue-300 hover:bg-blue-500/20 hover:border-blue-400 hover:text-white transition-all duration-300 hover:-translate-y-[1px]"
            >
              <History className="h-4 w-4" />
              <span className="hidden md:inline ml-1">History</span>
            </Button>

            <Button
              onClick={() => router.push("/password-change")}
              variant="outline"
              size="sm"
              className="h-8 px-3 bg-transparent border-orange-500/50 text-orange-300 hover:bg-orange-500/20 hover:border-orange-400 hover:text-white transition-all duration-300 hover:-translate-y-[1px]"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-4 w-4"
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
              <span className="hidden md:inline ml-1">Password</span>
            </Button>

            <Button
              onClick={handleExportEOD}
              variant="outline"
              size="sm"
              className="h-8 px-3 bg-transparent border-green-500/50 text-green-300 hover:bg-green-500/20 hover:border-green-400 hover:text-white transition-all duration-300 hover:-translate-y-[1px]"
            >
              <Download className="h-4 w-4" />
              <span className="hidden md:inline ml-1">Export</span>
            </Button>

            <ChatNotificationBadge
              href="/chat"
              size="sm"
              className="h-8 px-2"
            />

            {user?.role === "admin" && (
              <Button
                onClick={() => router.push("/admin")}
                variant="outline"
                size="sm"
                className="h-8 px-3 bg-transparent border-sky-800 text-sky-200 hover:bg-sky-900/50 hover:border-sky-500 hover:text-white transition-all duration-300 hover:-translate-y-[1px]"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className="h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                  />
                </svg>
                <span className="hidden md:inline ml-1">Admin</span>
              </Button>
            )}

            <Button
              onClick={logout}
              variant="outline"
              size="sm"
              className="h-8 px-2 bg-transparent border-white/10 text-white hover:bg-white/10 hover:border-white/20 transition-all duration-300 hover:-translate-y-[1px]"
            >
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div className="mb-3 flex items-start gap-3 rounded-xl border border-amber-400/25 bg-amber-400/10 px-4 py-3 text-sm text-amber-100">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-amber-300" aria-hidden="true" />
          <p><span className="text-amber-300">Data update:</span> 50% of Rajasthan and Maharashtra data has been uploaded.</p>
        </div>
        <LatestAnnouncement />

        {searchesUsed >= searchLimit && (
          <div className="mb-3 bg-red-500/10 border border-red-500 text-red-400 p-3 rounded-lg text-sm flex flex-col sm:flex-row sm:items-center gap-3 justify-between">
            <span>
              <strong>Daily limit reached!</strong>{" "}
              {/* searchLimit can be 0 (admins can set it); phrase accordingly */}
              {searchLimit > 0
                ? `You've used all ${searchLimit} searches.`
                : "Your quota for today is 0."}{" "}
              {istClock
                ? `Resets at 12:00 AM IST — in ${istClock.resetsIn}.`
                : "Resets at 12:00 AM IST."}
            </span>
            {limitOptedIn === false ? (
              <Button
                onClick={handleNotifyOptIn}
                disabled={optInProcessing}
                size="sm"
                className="flex-shrink-0 h-8 bg-transparent border-yellow-500/50 text-yellow-300 hover:bg-yellow-500/20 hover:border-yellow-400 hover:text-yellow-200 transition-all"
              >
                {optInProcessing ? (
                  <Spinner size="sm" className="mr-1" />
                ) : (
                  <Bell className="h-4 w-4 mr-1" />
                )}
                Notify me on reset
              </Button>
            ) : (
              <span className="flex-shrink-0 text-green-400 font-semibold text-xs bg-green-500/10 border border-green-500/30 rounded-lg px-3 py-1.5">
                Notify on reset ✅{" "}
                <span className="text-green-400/70 font-normal">
                  {istClock
                    ? `(delivered 12:00 AM IST — in ${istClock.notifiesIn})`
                    : "(delivered 12:00 AM IST)"}
                </span>
              </span>
            )}
          </div>
        )}

        {/* In-app-only quota heads-up banner (single, at ≥90% used) */}
        {!limitReached && percentageUsed >= 90 && searchLimit > 0 && (
          <div className="mb-3 bg-yellow-500/10 border border-yellow-500/50 text-yellow-300 p-3 rounded-lg text-sm">
            <strong>Almost out!</strong> You&apos;ve used{" "}
            {Math.round(percentageUsed)}% of today&apos;s search quota.{" "}
            {istClock
              ? `Resets at 12:00 AM IST — in ${istClock.resetsIn}.`
              : "Resets at 12:00 AM IST."}
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
        <div className="mb-3 rounded-2xl border border-sky-900/50 bg-[#0a1c38] p-4 md:p-6">
          <SearchForm
            searchFields={searchFields}
            operator={operator}
            onFieldChange={updateField}
            onOperatorChange={setOperator}
            onSearch={() => executeSearch(1)}
          />

          <div className="mt-6 flex flex-wrap justify-end gap-3 border-t border-sky-900/50 pt-4">
            <Button
              onClick={() => executeSearch(1)}
              disabled={loading || searchesUsed >= searchLimit}
              className="h-11 flex-1 border-none bg-sky-500 px-8 text-slate-950 transition-colors hover:bg-sky-400 sm:flex-none"
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
              className="h-11 border-sky-800/70 bg-transparent px-5 text-slate-200 transition-colors hover:border-sky-600 hover:bg-sky-900/40 hover:text-white"
            >
              <RotateCcw className="h-4 w-4 mr-2" />
              Reset
            </Button>
          </div>
        </div>

        {/* Results - Compact */}
        {totalResults > 0 && (
          <>
            {/* Refine Search Component */}
            <RefineSearch
              totalResults={totalResults}
              activeRefinements={activeRefinements}
              onAddRefinement={handleAddRefinement}
              onRemoveRefinement={handleRemoveRefinement}
              onClearRefinements={handleClearRefinements}
              onRefine={handleRefine}
              isRefining={isRefining}
            />

            {isRefinedView && (
              <div className="mb-3 bg-blue-500/10 border border-blue-500 text-blue-400 p-3 rounded-lg text-sm flex items-center gap-2">
                <AlertCircle className="h-4 w-4" />
                <span>
                  <strong>Refined view active!</strong> Showing filtered
                  results. Clear filters to see all results.
                </span>
              </div>
            )}

            <div className="mb-3">
              <ResultsStats
                startIndex={startIndex}
                endIndex={endIndex}
                totalResults={totalResults}
                searchTime={searchTime}
                filterQuery=""
                onFilterChange={() => {}}
              />
            </div>

            <ResultsTable
              results={results}
              copiedIndex={copiedIndex}
              onCopy={handleCopy}
              onCopyAll={handleCopyAll}
              isCopyingAll={isCopyingAll}
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
          <div className="bg-[#081a35] p-12 rounded-lg border border-gray-700 text-center">
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
