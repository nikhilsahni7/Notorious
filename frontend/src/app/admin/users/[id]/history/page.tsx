"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Clock, User as UserIcon } from "lucide-react";
import { adminService } from "@/services/admin.service";

interface HistoryItem {
  id: string;
  query: string;
  total_results: number;
  searched_at: string;
}

interface UserDetails {
  id: string;
  name: string;
  email: string;
}

export default function UserHistoryPage() {
  const { token, isLoading } = useAuth();
  const router = useRouter();
  const params = useParams();
  const userId = params.id as string;

  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [user, setUser] = useState<UserDetails | null>(null);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const limit = 50;

  useEffect(() => {
    if (!isLoading && !token) {
      router.push("/admin/login");
    }
  }, [isLoading, token, router]);

  useEffect(() => {
    if (token && userId) {
      loadData(page);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, userId, page]);

  const loadData = async (currentPage: number) => {
    setLoading(true);
    try {
      const offset = (currentPage - 1) * limit;
      const [historyData, userData] = await Promise.all([
        adminService.getUserSearchHistory(userId, token!, limit, offset),
        user ? Promise.resolve(user) : adminService.getUser(userId, token!),
      ]);
      setHistory(historyData || []);
      setHasMore((historyData || []).length === limit);
      setUser(userData);
    } catch (error) {
      console.error("Failed to load data:", error);
    } finally {
      setLoading(false);
      setInitialLoading(false);
    }
  };

  const parseOperator = (query: string): string => {
    if (query.includes(" AND ")) return "AND";
    if (query.includes(" OR ")) return "OR";
    return "AND";
  };

  if (isLoading || !token || initialLoading) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-4">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="flex justify-between items-center mb-4 bg-[#1a0f2e] p-4 rounded-lg border border-gray-700">
          <div className="flex items-center gap-4">
            <Button
              onClick={() => router.push("/admin")}
              variant="outline"
              size="sm"
              className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E]"
            >
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Admin
            </Button>
            <div>
              <div className="flex items-center gap-2 mb-1">
                <UserIcon className="h-5 w-5 text-purple-400" />
                <h1 className="text-2xl font-bold text-white">{user?.name}</h1>
              </div>
              <p className="text-sm text-gray-400">
                {user?.email} • Search History
              </p>
            </div>
          </div>
        </div>

        {/* History List */}
        <div className={`bg-[#1a0f2e] rounded-lg border border-gray-700 transition-opacity duration-300 ${loading ? "opacity-50 pointer-events-none" : "opacity-100"}`}>
          {history.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              <Clock className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p className="text-lg">No search history</p>
              <p className="text-sm mt-2">This user hasn&apos;t performed any searches yet</p>
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
                          <span className="text-xs bg-purple-500/20 text-purple-400 px-2 py-1 rounded font-medium">
                            #{(page - 1) * limit + index + 1}
                          </span>
                          <span className={`text-xs px-2 py-1 rounded font-medium ${
                            operator === "AND" 
                              ? "bg-blue-500/20 text-blue-400" 
                              : "bg-pink-500/20 text-pink-400"
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

        {/* Pagination */
        (history.length > 0 || page > 1) && (
          <div className="flex justify-between items-center mt-6 p-4 bg-[#1a0f2e] rounded-lg border border-gray-700">
            <Button
              disabled={page === 1 || loading}
              onClick={() => setPage((p) => p - 1)}
              variant="outline"
              className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E]"
            >
              Previous
            </Button>
            <span className="text-gray-400 font-medium">Page {page}</span>
            <Button
              disabled={!hasMore || loading}
              onClick={() => setPage((p) => p + 1)}
              variant="outline"
              className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E]"
            >
              Next
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

