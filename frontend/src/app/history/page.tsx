"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { LogOut, ArrowLeft, Clock } from "lucide-react";
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
        <div className="flex justify-between items-center mb-4">
          <div className="flex items-center gap-4">
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              className="bg-transparent border-gray-600 text-white hover:bg-[#1a0f2e]"
            >
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
            <div>
              <h1 className="text-2xl font-bold text-white">Search History</h1>
              <p className="text-sm text-gray-400">{user?.name}</p>
            </div>
          </div>
          <Button
            onClick={logout}
            variant="outline"
            className="bg-transparent border-gray-600 text-white hover:bg-[#1a0f2e]"
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
              <p>No search history yet</p>
              <p className="text-sm mt-2">Your searches will appear here</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-700">
              {history.map((item, index) => (
                <div
                  key={item.id || index}
                  className="p-4 hover:bg-[#2D1B4E] transition-colors"
                >
                  <div className="flex justify-between items-start gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <span className="text-xs bg-pink-500/20 text-pink-400 px-2 py-1 rounded">
                          #{history.length - index}
                        </span>
                        <code className="text-sm text-white bg-[#2D1B4E] px-3 py-1 rounded">
                          {item.query}
                        </code>
                      </div>
                      <div className="flex items-center gap-4 text-sm">
                        <span className="text-gray-400">
                          Results:{" "}
                          <span className="text-white font-medium">
                            {item.total_results.toLocaleString()}
                          </span>
                        </span>
                        <span className="text-gray-400">
                          <Clock className="h-3 w-3 inline mr-1" />
                          {new Date(item.searched_at).toLocaleString()}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

