import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { adminService, SearchHistoryItem } from "@/services/admin.service";
import { useEffect, useState } from "react";

interface SearchHistoryTabProps {
  token: string;
}

export function SearchHistoryTab({ token }: SearchHistoryTabProps) {
  const [loading, setLoading] = useState(true);
  const [history, setHistory] = useState<SearchHistoryItem[]>([]);
  const [filteredHistory, setFilteredHistory] = useState<SearchHistoryItem[]>(
    []
  );
  const [searchQuery, setSearchQuery] = useState("");

  useEffect(() => {
    loadHistory();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      setFilteredHistory(
        history.filter(
          (item) =>
            item.user_email?.toLowerCase().includes(query) ||
            item.user_name?.toLowerCase().includes(query) ||
            item.query.toLowerCase().includes(query)
        )
      );
    } else {
      setFilteredHistory(history);
    }
  }, [searchQuery, history]);

  const loadHistory = async () => {
    try {
      const data = await adminService.getSearchHistory(token, { limit: 100 });
      setHistory(data);
      setFilteredHistory(data);
    } catch (error) {
      console.error("Failed to load history:", error);
      alert("Failed to load search history");
    } finally {
      setLoading(false);
    }
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
        <h2 className="text-xl font-bold text-white">Search History</h2>
      </div>

      <div className="mb-4">
        <Input
          placeholder="Search by user, email, or query..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-400"
        />
      </div>

      <div className="bg-[#2D1B4E] rounded-lg border border-gray-700 overflow-hidden">
        <div className="overflow-x-auto">
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
              {filteredHistory.map((item) => (
                <tr
                  key={item.id}
                  className="hover:bg-[#1a0f2e] transition-colors"
                >
                  <td className="px-4 py-3 text-sm">
                    <div>
                      <p className="text-white">{item.user_name}</p>
                      <p className="text-gray-400 text-xs">{item.user_email}</p>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    <code className="bg-[#1a0f2e] px-2 py-1 rounded text-xs">
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
      </div>

      {filteredHistory.length === 0 && (
        <div className="text-center py-12 text-gray-400">
          {searchQuery
            ? "No search history found matching your search"
            : "No search history found"}
        </div>
      )}
    </div>
  );
}
