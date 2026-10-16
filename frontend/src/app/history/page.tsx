"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { LogOut, ArrowLeft, Clock, Search } from "lucide-react";
import { searchService } from "@/services/search.service";

interface HistoryItem {
  id: string;
  query: string;
  total_results: number;
  searched_at: string;
}

export default function HistoryPage() {
  const { user, token, logout, isLoading } = useAuth();
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [history, setHistory] = useState<HistoryItem[]>([]);

  useEffect(() => {
    if (!isLoading && !token) {
      router.push("/login");
    }
  }, [isLoading, token, router]);

  useEffect(() => {
    if (token) {
      loadHistory();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const loadHistory = async () => {
    try {
      const data = await searchService.getHistory(token!);
      setHistory(data || []);
    } catch (error) {
      console.error("Failed to load history:", error);
    } finally {
      setLoading(false);
    }
  };

  const parseOperator = (query: string): string => {
    if (query.includes(" AND ")) return "AND";
    if (query.includes(" OR ")) return "OR";
    return "AND";
  };

  if (isLoading || !token) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-4">
      <div className="max-w-5xl mx-auto">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3 mb-4 bg-[#1a0f2e] p-4 rounded-lg border border-gray-700">
          <div className="flex items-center gap-3 md:gap-4 min-w-0">
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              size="sm"
              className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E] flex-shrink-0"
            >
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
            <div className="min-w-0">
              <h1 className="text-xl md:text-2xl font-bold text-white">Search History</h1>
              <p className="text-sm text-gray-400 truncate">{user?.name} • {history.length} searches</p>
            </div>
          </div>
          <Button
            onClick={logout}
            variant="outline"
            size="sm"
            className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E] self-start sm:self-auto flex-shrink-0"
          >
            <LogOut className="h-4 w-4 mr-2" />
            Logout
          </Button>
        </div>

        {/* History List */}
        <div className="bg-[#1a0f2e] rounded-lg border border-gray-700">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Spinner size="lg" />
            </div>
          ) : history.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              <Clock className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p className="text-lg">No search history yet</p>
              <p className="text-sm mt-2">Your searches will appear here</p>
              <Button
                onClick={() => router.push("/search")}
                className="mt-4 bg-pink-500 hover:bg-pink-600 text-white"
              >
                <Search className="h-4 w-4 mr-2" />
                Start Searching
              </Button>
            </div>
          ) : (
            <div className="divide-y divide-gray-700">
              {history.map((item, index) => {
                const operator = parseOperator(item.query);
                return (
                  <div
                    key={item.id || index}
                    className="p-4 hover:bg-[#2D1B4E] transition-colors"
                  >
                    <div className="flex justify-between items-start gap-4">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-2 flex-wrap">
                          <span className="text-xs bg-pink-500/20 text-pink-400 px-2 py-1 rounded font-medium">
                            #{history.length - index}
                          </span>
                          <span className={`text-xs px-2 py-1 rounded font-medium ${
                            operator === "AND" 
                              ? "bg-blue-500/20 text-blue-400" 
                              : "bg-purple-500/20 text-purple-400"
                          }`}>
                            {operator}
                          </span>
                          <code className="text-sm text-white bg-[#2D1B4E] px-3 py-1 rounded flex-1">
                            {item.query}
                          </code>
                        </div>
                        <div className="flex items-center gap-4 text-sm">
                          <span className="text-gray-400">
                            Results:{" "}
                            <span className={`font-medium ${
                              item.total_results > 0 ? "text-green-400" : "text-gray-500"
                            }`}>
                              {item.total_results.toLocaleString()}
                            </span>
                          </span>
                          <span className="text-gray-400">
                            <Clock className="h-3 w-3 inline mr-1" />
                            {new Date(item.searched_at).toLocaleString("en-IN", {
                              dateStyle: "medium",
                              timeStyle: "short",
                            })}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
