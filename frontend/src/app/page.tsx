"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Search } from "lucide-react";
import { useMemo, useState } from "react";

interface SearchFields {
  name: string;
  fname: string;
  mobile: string;
  address: string;
  alt: string;
  id: string;
  email: string;
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
    name: "",
    fname: "",
    mobile: "",
    address: "",
    alt: "",
    id: "",
    email: "",
  });
  const [operator, setOperator] = useState<"AND" | "OR">("OR");
  const [results, setResults] = useState<Person[]>([]);
  const [totalResults, setTotalResults] = useState(0);
  const [loading, setLoading] = useState(false);
  const [searchTime, setSearchTime] = useState(0);
  const [clientSearchQuery, setClientSearchQuery] = useState("");

  const updateField = (field: keyof SearchFields, value: string) => {
    setSearchFields((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleSearch = async () => {
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

      // Use POST with JSON body for better reliability
      const response = await fetch(`${apiUrl}/search`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          query: queryString,
          fields: ["name", "fname", "mobile", "address", "alt", "id", "email"],
          and_or: operator,
          size: 100,
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

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  // Client-side filtering
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
        person.alt?.toLowerCase().includes(query) ||
        person.id?.toLowerCase().includes(query) ||
        person.email?.toLowerCase().includes(query)
      );
    });
  }, [results, clientSearchQuery]);

  const displayedCount = filteredResults.length;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-950 dark:to-slate-900">
      <div className="container mx-auto px-4 py-8 max-w-7xl">
        {/* Header */}
        <div className="mb-8 text-center">
          <h1 className="text-4xl font-bold text-slate-900 dark:text-slate-100 mb-2">
            People Search Dashboard
          </h1>
          <p className="text-slate-600 dark:text-slate-400">
            Search across 500GB of data with sub-second speed
          </p>
        </div>

        {/* Search Panel */}
        <Card className="mb-6 shadow-lg">
          <CardHeader>
            <div className="space-y-2">
              <CardTitle className="flex items-center justify-between flex-wrap gap-4">
                <span className="flex items-center gap-2">
                  <Search className="h-5 w-5" />
                  Search Filters
                </span>
                <div className="flex items-center gap-4">
                  <Label htmlFor="operator" className="text-sm font-medium">
                    Match Mode:
                  </Label>
                  <div className="flex items-center gap-2 bg-slate-100 dark:bg-slate-800 px-3 py-2 rounded-md">
                    <span
                      className={`text-sm font-medium ${
                        operator === "AND"
                          ? "text-slate-900 dark:text-slate-100"
                          : "text-slate-400"
                      }`}
                    >
                      ALL (AND)
                    </span>
                    <Switch
                      id="operator"
                      checked={operator === "OR"}
                      onCheckedChange={(checked) =>
                        setOperator(checked ? "OR" : "AND")
                      }
                    />
                    <span
                      className={`text-sm font-medium ${
                        operator === "OR"
                          ? "text-slate-900 dark:text-slate-100"
                          : "text-slate-400"
                      }`}
                    >
                      ANY (OR)
                    </span>
                  </div>
                </div>
              </CardTitle>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {operator === "AND"
                  ? "🔍 ALL mode: Results must match all filled fields"
                  : "🔍 ANY mode: Results match if any filled field matches"}
              </p>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* All Search Fields Visible */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="space-y-2">
                <Label
                  htmlFor="name"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Name
                </Label>
                <Input
                  id="name"
                  placeholder="Enter name..."
                  value={searchFields.name}
                  onChange={(e) => updateField("name", e.target.value)}
                  onKeyPress={handleKeyPress}
                />
              </div>

              <div className="space-y-2">
                <Label
                  htmlFor="fname"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Father&apos;s Name
                </Label>
                <Input
                  id="fname"
                  placeholder="Enter father's name..."
                  value={searchFields.fname}
                  onChange={(e) => updateField("fname", e.target.value)}
                  onKeyPress={handleKeyPress}
                />
              </div>

              <div className="space-y-2">
                <Label
                  htmlFor="mobile"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Mobile Number
                </Label>
                <Input
                  id="mobile"
                  placeholder="Enter mobile..."
                  value={searchFields.mobile}
                  onChange={(e) => updateField("mobile", e.target.value)}
                  onKeyPress={handleKeyPress}
                />
              </div>

              <div className="space-y-2">
                <Label
                  htmlFor="alt"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Alternate Number
                </Label>
                <Input
                  id="alt"
                  placeholder="Enter alternate number..."
                  value={searchFields.alt}
                  onChange={(e) => updateField("alt", e.target.value)}
                  onKeyPress={handleKeyPress}
                />
              </div>

              <div className="space-y-2">
                <Label
                  htmlFor="id"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Master ID
                </Label>
                <Input
                  id="id"
                  placeholder="Enter master ID..."
                  value={searchFields.id}
                  onChange={(e) => updateField("id", e.target.value)}
                  onKeyPress={handleKeyPress}
                />
              </div>

              <div className="space-y-2">
                <Label
                  htmlFor="email"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Email Address
                </Label>
                <Input
                  id="email"
                  placeholder="Enter email..."
                  value={searchFields.email}
                  onChange={(e) => updateField("email", e.target.value)}
                  onKeyPress={handleKeyPress}
                />
              </div>

              <div className="space-y-2 md:col-span-2">
                <Label
                  htmlFor="address"
                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                >
                  Address
                </Label>
                <Input
                  id="address"
                  placeholder="Enter address..."
                  value={searchFields.address}
                  onChange={(e) => updateField("address", e.target.value)}
                  onKeyPress={handleKeyPress}
                />
              </div>
            </div>

            {/* Search Button */}
            <div className="flex justify-end pt-4 border-t">
              <Button
                onClick={handleSearch}
                disabled={loading}
                size="lg"
                className="w-full md:w-auto"
              >
                {loading ? (
                  <>
                    <Spinner size="sm" className="mr-2" />
                    Searching...
                  </>
                ) : (
                  <>
                    <Search className="h-4 w-4 mr-2" />
                    Search Database
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Results Section */}
        {results.length > 0 && (
          <>
            {/* Results Header */}
            <div className="mb-4 flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between bg-white dark:bg-slate-900 p-4 rounded-lg shadow">
              <div className="text-sm">
                <span className="text-slate-600 dark:text-slate-400">
                  Showing{" "}
                </span>
                <span className="font-bold text-lg text-slate-900 dark:text-slate-100">
                  {displayedCount}
                </span>
                <span className="text-slate-600 dark:text-slate-400"> of </span>
                <span className="font-bold text-lg text-slate-900 dark:text-slate-100">
                  {totalResults.toLocaleString()}
                </span>
                <span className="text-slate-600 dark:text-slate-400">
                  {" "}
                  results
                </span>
                <span className="ml-3 text-xs bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 px-2 py-1 rounded">
                  {searchTime}ms
                </span>
              </div>

              {/* Client-side Search */}
              <div className="w-full sm:w-96">
                <Input
                  placeholder="🔍 Filter displayed results..."
                  value={clientSearchQuery}
                  onChange={(e) => setClientSearchQuery(e.target.value)}
                  className="w-full"
                />
              </div>
            </div>

            {/* Results Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {filteredResults.map((person, index) => (
                <Card
                  key={`${person.id}-${person.mobile}-${index}`}
                  className="hover:shadow-xl transition-all duration-200 border-l-4 border-l-blue-500"
                >
                  <CardContent className="p-5">
                    <div className="space-y-3">
                      {/* Name Section */}
                      <div className="pb-2 border-b-2 border-slate-200 dark:border-slate-700">
                        <p className="text-xl font-bold text-slate-900 dark:text-slate-100 mb-1">
                          {person.name}
                        </p>
                        <p className="text-sm font-medium text-slate-600 dark:text-slate-400">
                          Father&apos;s Name:{" "}
                          <span className="text-slate-800 dark:text-slate-300">
                            {person.fname}
                          </span>
                        </p>
                      </div>

                      {/* Contact Details */}
                      <div className="space-y-2">
                        {person.mobile && (
                          <div className="flex items-center justify-between bg-slate-50 dark:bg-slate-800 p-2 rounded">
                            <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase">
                              Mobile Number:
                            </span>
                            <span className="font-bold text-blue-600 dark:text-blue-400">
                              {person.mobile}
                            </span>
                          </div>
                        )}
                        {person.alt && (
                          <div className="flex items-center justify-between bg-slate-50 dark:bg-slate-800 p-2 rounded">
                            <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase">
                              Alternate Number:
                            </span>
                            <span className="font-bold text-green-600 dark:text-green-400">
                              {person.alt}
                            </span>
                          </div>
                        )}
                        {person.id && (
                          <div className="flex items-center justify-between bg-slate-50 dark:bg-slate-800 p-2 rounded">
                            <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase">
                              Master ID:
                            </span>
                            <span className="font-mono font-bold text-purple-600 dark:text-purple-400 text-sm">
                              {person.id}
                            </span>
                          </div>
                        )}
                        {person.email && (
                          <div className="bg-slate-50 dark:bg-slate-800 p-2 rounded">
                            <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase block mb-1">
                              Email:
                            </span>
                            <span className="font-medium text-xs text-slate-800 dark:text-slate-300 break-all">
                              {person.email}
                            </span>
                          </div>
                        )}
                        {person.address && (
                          <div className="bg-slate-50 dark:bg-slate-800 p-2 rounded">
                            <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase block mb-1">
                              Address:
                            </span>
                            <p className="text-xs leading-relaxed text-slate-700 dark:text-slate-300">
                              {person.address
                                .replace(/!/g, ", ")
                                .replace(/, ,/g, ",")
                                .replace(/^,/g, "")
                                .replace(/,$/g, "")}
                            </p>
                          </div>
                        )}
                        {Boolean(person.year_of_registration) && (
                          <div className="flex items-center justify-between bg-amber-50 dark:bg-amber-900/20 p-2 rounded">
                            <span className="text-xs font-semibold text-amber-700 dark:text-amber-400 uppercase">
                              Registration Year:
                            </span>
                            <span className="font-bold text-amber-700 dark:text-amber-400">
                              {person.year_of_registration}
                            </span>
                          </div>
                        )}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>

            {filteredResults.length === 0 && Boolean(clientSearchQuery) && (
              <Card className="p-8 text-center">
                <p className="text-slate-600 dark:text-slate-400">
                  No results match your filter &quot;{clientSearchQuery}&quot;
                </p>
              </Card>
            )}
          </>
        )}

        {/* Empty State */}
        {!loading && results.length === 0 && (
          <Card className="p-12 text-center">
            <Search className="h-12 w-12 mx-auto mb-4 text-slate-400" />
            <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-2">
              No Search Results
            </h3>
            <p className="text-slate-600 dark:text-slate-400">
              Enter your search criteria above and click Search Database
            </p>
          </Card>
        )}

        {/* Loading State */}
        {loading && (
          <Card className="p-12 text-center">
            <Spinner size="lg" className="mx-auto mb-4" />
            <p className="text-slate-600 dark:text-slate-400">
              Searching database...
            </p>
          </Card>
        )}
      </div>
    </div>
  );
}
