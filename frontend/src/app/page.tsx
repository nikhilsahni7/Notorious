"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Check, ChevronLeft, ChevronRight, Copy, Search } from "lucide-react";
import { useMemo, useState } from "react";

interface SearchFields {
  id: string;
  name: string;
  fname: string;
  mobile: string;
  alt: string;
  email: string;
  address: string;
}

interface Person {
  mobile: string;
  name: string;
  fname: string;
  address: string;
  alt_address: string;
  alt: string;
  id: string;
  email: string;
  year_of_registration: number;
}

interface SearchResponse {
  total: number;
  results: Person[];
  took_ms: number;
}

export default function Home() {
  const [searchFields, setSearchFields] = useState<SearchFields>({
    id: "",
    name: "",
    fname: "",
    mobile: "",
    alt: "",
    email: "",
    address: "",
  });
  const [operator, setOperator] = useState<"AND" | "OR">("AND");
  const [results, setResults] = useState<Person[]>([]);
  const [totalResults, setTotalResults] = useState(0);
  const [loading, setLoading] = useState(false);
  const [searchTime, setSearchTime] = useState(0);
  const [clientSearchQuery, setClientSearchQuery] = useState("");
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(50);

  const updateField = (field: keyof SearchFields, value: string) => {
    setSearchFields((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const performSearch = async (page: number = 1) => {
    setLoading(true);
    setClientSearchQuery("");

    try {
      // Build query from non-empty fields with field:value format
      const queries: string[] = [];
      Object.entries(searchFields).forEach(([field, value]) => {
        if (value.trim()) {
          // Format as field:value for specific field search
          queries.push(`${field}:${value.trim()}`);
        }
      });

      if (queries.length === 0) {
        alert("Please enter at least one search term");
        setLoading(false);
        return;
      }

      const queryString = queries.join(` ${operator} `);
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

      console.log("Search query:", queryString); // Debug log

      // Calculate offset for pagination
      const from = (page - 1) * pageSize;

      // Use POST with JSON body for better reliability
      const response = await fetch(`${apiUrl}/search`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          query: queryString,
          fields: ["id", "name", "fname", "mobile", "alt", "email", "address"],
          and_or: operator,
          size: pageSize,
          from: from,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(
          errorData.error || `Search failed: ${response.statusText}`
        );
      }

      const data: SearchResponse = await response.json();
      setResults(data.results || []);
      setTotalResults(data.total || 0);
      setSearchTime(data.took_ms || 0);
    } catch (error) {
      console.error("Search error:", error);
      alert(
        `Search failed: ${
          error instanceof Error ? error.message : "Unknown error"
        }`
      );
      setResults([]);
      setTotalResults(0);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    setCurrentPage(1); // Reset to first page on new search
    await performSearch(1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  const copyToClipboard = async (person: Person, index: number) => {
    const text = `Name: ${person.name}
Father Name: ${person.fname}
Master ID: ${person.id}
Mobile: ${person.mobile}
Alternate Phone: ${person.alt}
Email: ${person.email}
Address: ${person.address
      ?.replace(/!/g, ", ")
      .replace(/, ,/g, ",")
      .replace(/^,/g, "")
      .replace(/,$/g, "")}
Alt Address: ${person.alt_address
      ?.replace(/!/g, ", ")
      .replace(/, ,/g, ",")
      .replace(/^,/g, "")
      .replace(/,$/g, "")}
Year of Registration: ${person.year_of_registration}`;

    await navigator.clipboard.writeText(text);
    setCopiedIndex(index);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  // Client-side filtering and pagination
  const filteredResults = useMemo(() => {
    if (!clientSearchQuery.trim()) {
      return results;
    }

    const query = clientSearchQuery.toLowerCase();
    return results.filter((person) => {
      return (
        person.name?.toLowerCase().includes(query) ||
        person.fname?.toLowerCase().includes(query) ||
        person.mobile?.toLowerCase().includes(query) ||
        person.address?.toLowerCase().includes(query) ||
        person.alt_address?.toLowerCase().includes(query) ||
        person.alt?.toLowerCase().includes(query) ||
        person.id?.toLowerCase().includes(query) ||
        person.email?.toLowerCase().includes(query)
      );
    });
  }, [results, clientSearchQuery]);

  // Pagination calculation - use total from backend for multi-page support
  const totalPages = Math.ceil(totalResults / pageSize);
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = Math.min(startIndex + filteredResults.length, totalResults);
  const paginatedResults = filteredResults; // Show all filtered results (already paginated from backend)

  const handleNextPage = async () => {
    if (currentPage < totalPages) {
      const nextPage = currentPage + 1;
      setCurrentPage(nextPage);
      await performSearch(nextPage);
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  };

  const handlePrevPage = async () => {
    if (currentPage > 1) {
      const prevPage = currentPage - 1;
      setCurrentPage(prevPage);
      await performSearch(prevPage);
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  };

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-6">
      <div className="max-w-[1800px] mx-auto">
        {/* Search Fields Section */}
        <div className="mb-6 space-y-3">
          {/* Row 1 */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <Input
              id="id"
              placeholder="Enter master ID..."
              value={searchFields.id}
              onChange={(e) => updateField("id", e.target.value)}
              onKeyDown={handleKeyDown}
              className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
            />
            <Input
              id="name"
              placeholder="Enter name..."
              value={searchFields.name}
              onChange={(e) => updateField("name", e.target.value)}
              onKeyDown={handleKeyDown}
              className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
            />
            <Input
              id="fname"
              placeholder="Enter father's name..."
              value={searchFields.fname}
              onChange={(e) => updateField("fname", e.target.value)}
              onKeyDown={handleKeyDown}
              className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
            />
            <Input
              id="mobile"
              placeholder="Enter mobile number..."
              value={searchFields.mobile}
              onChange={(e) => updateField("mobile", e.target.value)}
              onKeyDown={handleKeyDown}
              className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
            />
          </div>

          {/* Row 2 */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <Input
              id="alt"
              placeholder="Enter alternate number..."
              value={searchFields.alt}
              onChange={(e) => updateField("alt", e.target.value)}
              onKeyDown={handleKeyDown}
              className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400 md:col-span-1"
            />
            <Input
              id="email"
              placeholder="Enter email..."
              value={searchFields.email}
              onChange={(e) => updateField("email", e.target.value)}
              onKeyDown={handleKeyDown}
              className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400 md:col-span-1"
            />
            <Input
              id="address"
              placeholder="Enter address..."
              value={searchFields.address}
              onChange={(e) => updateField("address", e.target.value)}
              onKeyDown={handleKeyDown}
              className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400 md:col-span-2"
            />
          </div>

          {/* Search Button and Operator */}
          <div className="flex items-center justify-between gap-4">
            <Button
              onClick={handleSearch}
              disabled={loading}
              className="bg-pink-500 hover:bg-pink-600 text-white px-6"
            >
              {loading ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Searching...
                </>
              ) : (
                <>
                  <Search className="h-4 w-4 mr-2" />
                  Search
                </>
              )}
            </Button>
            <div className="flex items-center gap-2 text-white text-sm">
              <span
                className={operator === "AND" ? "text-white" : "text-gray-400"}
              >
                AND
              </span>
              <Switch
                id="operator"
                checked={operator === "OR"}
                onCheckedChange={(checked) =>
                  setOperator(checked ? "OR" : "AND")
                }
              />
              <span
                className={operator === "OR" ? "text-white" : "text-gray-400"}
              >
                OR
              </span>
            </div>
          </div>
        </div>

        {/* Results Section */}
        {results.length > 0 && (
          <>
            {/* Results Header */}
            <div className="mb-4 space-y-3">
              {/* Pagination Controls - TOP */}
              {totalPages > 1 && (
                <div className="flex items-center justify-center gap-3 bg-[#1a0f2e] p-3 rounded-lg">
                  <Button
                    onClick={handlePrevPage}
                    disabled={currentPage === 1}
                    className="bg-pink-500 hover:bg-pink-600 disabled:opacity-50 disabled:cursor-not-allowed text-white"
                    size="sm"
                  >
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    Previous
                  </Button>
                  <span className="text-white font-semibold text-sm px-4">
                    Page {currentPage} of {totalPages}
                  </span>
                  <Button
                    onClick={handleNextPage}
                    disabled={currentPage === totalPages}
                    className="bg-pink-500 hover:bg-pink-600 disabled:opacity-50 disabled:cursor-not-allowed text-white"
                    size="sm"
                  >
                    Next
                    <ChevronRight className="h-4 w-4 ml-1" />
                  </Button>
                </div>
              )}

              {/* Results Info and Filter */}
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
                  value={clientSearchQuery}
                  onChange={(e) => setClientSearchQuery(e.target.value)}
                  className="w-64 bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
                />
              </div>
            </div>

            {/* Table Header */}
            <div className="grid grid-cols-[repeat(13,minmax(0,1fr))] gap-2 mb-2 text-sm font-semibold">
              <div className="col-span-2 bg-[#4A5568] text-white p-2 rounded">
                Name
              </div>
              <div className="col-span-1 bg-[#4A5568] text-white p-2 rounded text-center">
                Father Name
              </div>
              <div className="col-span-1 bg-[#4A5568] text-white p-2 rounded text-center">
                Master ID
              </div>
              <div className="col-span-2 bg-[#9AE6B4] text-gray-900 p-2 rounded">
                Address
              </div>
              <div className="col-span-2 bg-[#FC8181] text-white p-2 rounded">
                Alt Address
              </div>
              <div className="col-span-1 bg-[#D69E2E] text-white p-2 rounded text-center">
                Email
              </div>
              <div className="col-span-1 bg-[#4299E1] text-white p-2 rounded text-center">
                Year
              </div>
              <div className="col-span-1 bg-[#ED64A6] text-white p-2 rounded text-center">
                Mobile
              </div>
              <div className="col-span-1 bg-[#805AD5] text-white p-2 rounded text-center">
                Alt Phone
              </div>
              <div className="col-span-1 bg-gray-600 text-white p-2 rounded text-center">
                Action
              </div>
            </div>

            {/* Table Rows */}
            <div className="space-y-2">
              {paginatedResults.map((person, index) => (
                <div
                  key={`${person.id}-${person.mobile}-${index}`}
                  className="grid grid-cols-[repeat(13,minmax(0,1fr))] gap-2 text-sm bg-[#1a0f2e]/50 hover:bg-[#1a0f2e] transition-colors rounded overflow-hidden"
                >
                  {/* Name */}
                  <div className="col-span-2 bg-[#2D3748] text-white p-3 flex items-center">
                    <div className="truncate">{person.name || "-"}</div>
                  </div>

                  {/* Father Name */}
                  <div className="col-span-1 bg-[#2D3748] text-white p-3 flex items-center justify-center">
                    <div className="truncate text-xs">
                      {person.fname || "-"}
                    </div>
                  </div>

                  {/* Master ID */}
                  <div className="col-span-1 bg-[#2D3748] text-white p-3 flex items-center justify-center">
                    <div className="truncate text-xs font-mono">
                      {person.id || "-"}
                    </div>
                  </div>

                  {/* Address - Full text with wrapping */}
                  <div className="col-span-2 bg-[#68D391] text-gray-900 p-3 flex items-center">
                    <div className="text-xs break-words">
                      {person.address
                        ? person.address
                            .replace(/!/g, ", ")
                            .replace(/, ,/g, ",")
                            .replace(/^,/g, "")
                            .replace(/,$/g, "")
                        : "-"}
                    </div>
                  </div>

                  {/* Alt Address - Full text with wrapping */}
                  <div className="col-span-2 bg-[#F56565] text-white p-3 flex items-center">
                    <div className="text-xs break-words">
                      {person.alt_address
                        ? person.alt_address
                            .replace(/!/g, ", ")
                            .replace(/, ,/g, ",")
                            .replace(/^,/g, "")
                            .replace(/,$/g, "")
                        : "-"}
                    </div>
                  </div>

                  {/* Email - Full text with wrapping */}
                  <div className="col-span-1 bg-[#ECC94B] text-white p-3 flex items-center">
                    <div className="text-xs break-all">
                      {person.email || "-"}
                    </div>
                  </div>

                  {/* Year of Registration */}
                  <div className="col-span-1 bg-[#4299E1] text-white p-3 flex items-center justify-center">
                    <div className="font-semibold text-xs">
                      {person.year_of_registration || "-"}
                    </div>
                  </div>

                  {/* Mobile Number */}
                  <div className="col-span-1 bg-[#ED64A6] text-white p-3 flex items-center justify-center">
                    <div className="text-xs">{person.mobile || "-"}</div>
                  </div>

                  {/* Alternate Phone Number */}
                  <div className="col-span-1 bg-[#805AD5] text-white p-3 flex items-center justify-center">
                    <div className="text-xs">{person.alt || "-"}</div>
                  </div>

                  {/* Copy Button */}
                  <div className="col-span-1 bg-gray-700 text-white p-3 flex items-center justify-center">
                    <button
                      onClick={() => copyToClipboard(person, index)}
                      className="hover:bg-gray-600 p-1 rounded transition-colors"
                      title="Copy all data"
                    >
                      {copiedIndex === index ? (
                        <Check className="h-4 w-4 text-green-400" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                    </button>
                  </div>
                </div>
              ))}
            </div>

            {/* Bottom Pagination Controls */}
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-center gap-3 bg-[#1a0f2e] p-3 rounded-lg">
                <Button
                  onClick={handlePrevPage}
                  disabled={currentPage === 1}
                  className="bg-pink-500 hover:bg-pink-600 disabled:opacity-50 disabled:cursor-not-allowed text-white"
                  size="sm"
                >
                  <ChevronLeft className="h-4 w-4 mr-1" />
                  Previous
                </Button>
                <span className="text-white font-semibold text-sm px-4">
                  Page {currentPage} of {totalPages}
                </span>
                <Button
                  onClick={handleNextPage}
                  disabled={currentPage === totalPages}
                  className="bg-pink-500 hover:bg-pink-600 disabled:opacity-50 disabled:cursor-not-allowed text-white"
                  size="sm"
                >
                  Next
                  <ChevronRight className="h-4 w-4 ml-1" />
                </Button>
              </div>
            )}

            {filteredResults.length === 0 && Boolean(clientSearchQuery) && (
              <div className="p-8 text-center bg-[#1a0f2e] rounded-lg">
                <p className="text-gray-400">
                  No results match your filter &quot;{clientSearchQuery}&quot;
                </p>
              </div>
            )}
          </>
        )}

        {/* Empty State */}
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

        {/* Loading State */}
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
