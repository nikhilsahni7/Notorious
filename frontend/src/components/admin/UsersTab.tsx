import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { adminService, User } from "@/services/admin.service";
import {
  BarChart3,
  CheckSquare,
  ChevronLeft,
  ChevronRight,
  Download,
  Edit,
  History,
  Key,
  Plus,
  Power,
  Smartphone,
  Square,
  Trash2,
  UserCheck,
  UserX,
  X,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { ChangePasswordModal } from "./ChangePasswordModal";
import { CreateUserModal } from "./CreateUserModal";
import { EditUserModal } from "./EditUserModal";
import { UserSessionsModal } from "./UserSessionsModal";

interface UsersTabProps {
  token: string;
}

export function UsersTab({ token }: UsersTabProps) {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [fetching, setFetching] = useState(false);
  const [hasLoadedInitially, setHasLoadedInitially] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [regionFilter, setRegionFilter] = useState<
    "all" | "pan-india" | "delhi-ncr"
  >("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalCount, setTotalCount] = useState(0);
  const [counts, setCounts] = useState({
    all: 0,
    "pan-india": 0,
    "delhi-ncr": 0,
  });
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [changingPasswordUser, setChangingPasswordUser] = useState<User | null>(
    null
  );
  const [viewingSessionsUser, setViewingSessionsUser] = useState<User | null>(
    null
  );
  const [selectedUserIds, setSelectedUserIds] = useState<Set<string>>(
    new Set()
  );
  const [bulkUpdating, setBulkUpdating] = useState(false);
  const [onlineUserIds, setOnlineUserIds] = useState<Set<string>>(new Set());

  const limit = 50;
  const totalPages = Math.max(1, Math.ceil(totalCount / limit));

  const loadOnlineUsers = async () => {
    try {
      const data = await adminService.getOnlineUsers(token);
      setOnlineUserIds(new Set(data.online_user_ids || []));
    } catch (error) {
      console.error("Failed to load online users:", error);
    }
  };

  useEffect(() => {
    loadOnlineUsers();
    const interval = setInterval(loadOnlineUsers, 30000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, regionFilter]);

  useEffect(() => {
    const isFirstLoad = !hasLoadedInitially;
    if (isFirstLoad) {
      setHasLoadedInitially(true);
    }

    const delayDebounceFn = setTimeout(
      () => {
        loadUsers(currentPage, searchQuery, regionFilter, isFirstLoad);
      },
      isFirstLoad ? 0 : 300
    );

    return () => clearTimeout(delayDebounceFn);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPage, searchQuery, regionFilter]);

  const loadUsers = async (
    page: number,
    query: string,
    region: "all" | "pan-india" | "delhi-ncr",
    isInitialLoad: boolean
  ) => {
    if (isInitialLoad) {
      setLoading(true);
    } else {
      setFetching(true);
    }
    try {
      const offset = (page - 1) * limit;
      const data = await adminService.listUsers(
        token,
        limit,
        offset,
        query,
        region
      );
      setUsers(data.users || []);
      setTotalCount(data.total || 0);
      setCounts({
        all: data.counts?.all ?? 0,
        "pan-india": data.counts?.["pan-india"] ?? 0,
        "delhi-ncr": data.counts?.["delhi-ncr"] ?? 0,
      });
      setSelectedUserIds(new Set());
    } catch (error) {
      console.error("Failed to load users:", error);
      alert("Failed to load users");
    } finally {
      setLoading(false);
      setFetching(false);
    }
  };

  const reloadCurrentPage = () => {
    loadUsers(currentPage, searchQuery, regionFilter, false);
  };

  const handlePageChange = (page: number) => {
    if (page >= 1 && page <= totalPages) {
      setCurrentPage(page);
    }
  };

  const getPageNumbers = () => {
    const pages: (number | string)[] = [];
    const siblingCount = 1;

    pages.push(1);

    const leftSiblingIndex = Math.max(currentPage - siblingCount, 2);
    const rightSiblingIndex = Math.min(
      currentPage + siblingCount,
      totalPages - 1
    );

    const showLeftSpill = leftSiblingIndex > 2;
    const showRightSpill = rightSiblingIndex < totalPages - 1;

    if (showLeftSpill) {
      pages.push("...");
    }

    for (let i = leftSiblingIndex; i <= rightSiblingIndex; i++) {
      pages.push(i);
    }

    if (showRightSpill) {
      pages.push("...");
    }

    if (totalPages > 1) {
      pages.push(totalPages);
    }

    return pages;
  };

  const handleDelete = async (userId: string, userName: string) => {
    if (!confirm(`Are you sure you want to delete user "${userName}"?`)) {
      return;
    }

    try {
      await adminService.deleteUser(userId, token);
      reloadCurrentPage();
    } catch (error) {
      console.error("Failed to delete user:", error);
      alert("Failed to delete user");
    }
  };

  const handleGenerateEOD = async (userId: string, userName: string) => {
    try {
      const blob = await adminService.generateUserEOD(userId, token);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${userName}_EOD_${
        new Date().toISOString().split("T")[0]
      }.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error("Failed to generate EOD report:", error);
      alert("Failed to generate EOD report");
    }
  };

  const toggleSelectAll = () => {
    if (selectedUserIds.size === users.length) {
      setSelectedUserIds(new Set());
    } else {
      setSelectedUserIds(new Set(users.map((u) => u.id)));
    }
  };

  const toggleSelectUser = (userId: string) => {
    const newSelected = new Set(selectedUserIds);
    if (newSelected.has(userId)) {
      newSelected.delete(userId);
    } else {
      newSelected.add(userId);
    }
    setSelectedUserIds(newSelected);
  };

  const clearSelection = () => {
    setSelectedUserIds(new Set());
  };

  const handleBulkUpdate = async (isActive: boolean) => {
    if (selectedUserIds.size === 0) return;

    const action = isActive ? "activate" : "deactivate";
    if (
      !confirm(
        `Are you sure you want to ${action} ${selectedUserIds.size} selected user(s)?`
      )
    ) {
      return;
    }

    setBulkUpdating(true);
    try {
      await adminService.bulkUpdateUsers(
        Array.from(selectedUserIds),
        isActive,
        token
      );
      clearSelection();
      reloadCurrentPage();
    } catch (error) {
      console.error(`Failed to ${action} users:`, error);
      alert(`Failed to ${action} users`);
    } finally {
      setBulkUpdating(false);
    }
  };

  const handleToggleStatus = async (user: User) => {
    try {
      await adminService.toggleUserStatus(user.id, !user.is_active, token);
      reloadCurrentPage();
    } catch (error) {
      console.error("Failed to toggle user status:", error);
      alert("Failed to toggle user status");
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
        <div>
          <h2 className="text-xl font-bold text-white">User Management</h2>
          <p className="text-sm text-gray-400 mt-1">
            Showing {users.length} of {totalCount} total users
          </p>
        </div>
        <Button
          onClick={() => setShowCreateModal(true)}
          className="bg-pink-500 hover:bg-pink-600 text-white"
        >
          <Plus className="h-4 w-4 mr-2" />
          Create User
        </Button>
      </div>

      <div className="mb-4">
        <Input
          placeholder="Search users by name, email, or phone..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-400"
        />
      </div>

      {/* Region Filter */}
      <div className="mb-4 flex gap-2">
        <button
          onClick={() => setRegionFilter("all")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            regionFilter === "all"
              ? "bg-pink-500 text-white"
              : "bg-[#2D1B4E] text-gray-300 hover:bg-[#3D2B5E]"
          }`}
        >
          All Users ({counts.all})
        </button>
        <button
          onClick={() => setRegionFilter("pan-india")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            regionFilter === "pan-india"
              ? "bg-blue-500 text-white"
              : "bg-[#2D1B4E] text-gray-300 hover:bg-[#3D2B5E]"
          }`}
        >
          🌏 Pan-India ({counts["pan-india"]})
        </button>
        <button
          onClick={() => setRegionFilter("delhi-ncr")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            regionFilter === "delhi-ncr"
              ? "bg-green-500 text-white"
              : "bg-[#2D1B4E] text-gray-300 hover:bg-[#3D2B5E]"
          }`}
        >
          📍 Delhi-NCR ({counts["delhi-ncr"]})
        </button>
      </div>

      {/* Bulk Actions Toolbar */}
      {selectedUserIds.size > 0 && (
        <div className="mb-4 bg-[#2D1B4E] border border-purple-500/30 rounded-lg p-3 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-white font-medium">
              <CheckSquare className="h-4 w-4 inline mr-2 text-purple-400" />
              {selectedUserIds.size} user(s) selected
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              onClick={() => handleBulkUpdate(true)}
              disabled={bulkUpdating}
              className="bg-green-600 hover:bg-green-700 text-white text-sm"
              size="sm"
            >
              <UserCheck className="h-4 w-4 mr-1" />
              Activate
            </Button>
            <Button
              onClick={() => handleBulkUpdate(false)}
              disabled={bulkUpdating}
              className="bg-red-600 hover:bg-red-700 text-white text-sm"
              size="sm"
            >
              <UserX className="h-4 w-4 mr-1" />
              Deactivate
            </Button>
            <Button
              onClick={clearSelection}
              variant="outline"
              className="border-gray-600 text-gray-300 hover:bg-gray-700 text-sm"
              size="sm"
            >
              <X className="h-4 w-4 mr-1" />
              Clear
            </Button>
          </div>
        </div>
      )}

      <div className="bg-[#2D1B4E] rounded-lg border border-gray-700 overflow-hidden relative">
        <div
          className={`overflow-x-auto transition-opacity duration-200 ${
            fetching ? "opacity-40" : "opacity-100"
          }`}
        >
          <table className="w-full">
            <thead className="bg-[#1a0f2e] border-b border-gray-700">
              <tr>
                <th className="px-3 py-3 text-left">
                  <button
                    onClick={toggleSelectAll}
                    className="text-gray-300 hover:text-white transition-colors"
                    title={
                      selectedUserIds.size === users.length
                        ? "Deselect all"
                        : "Select all"
                    }
                  >
                    {selectedUserIds.size === users.length &&
                    users.length > 0 ? (
                      <CheckSquare className="h-5 w-5 text-purple-400" />
                    ) : (
                      <Square className="h-5 w-5" />
                    )}
                  </button>
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Name
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Email
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Role
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Region
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Daily Limit
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Device Limit
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Used Today
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Total Searches
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Status
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700">
              {users.map((user) => (
                <tr
                  key={user.id}
                  className={`hover:bg-[#1a0f2e] transition-colors ${
                    selectedUserIds.has(user.id) ? "bg-purple-500/10" : ""
                  }`}
                >
                  <td className="px-3 py-3">
                    <button
                      onClick={() => toggleSelectUser(user.id)}
                      className="text-gray-300 hover:text-white transition-colors"
                    >
                      {selectedUserIds.has(user.id) ? (
                        <CheckSquare className="h-5 w-5 text-purple-400" />
                      ) : (
                        <Square className="h-5 w-5" />
                      )}
                    </button>
                  </td>
                  <td className="px-4 py-3 text-sm text-white">
                    <div className="flex items-center gap-2">
                      <span
                        className={`w-2 h-2 rounded-full flex-shrink-0 ${
                          onlineUserIds.has(user.id)
                            ? "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]"
                            : "bg-gray-500"
                        }`}
                        title={
                          onlineUserIds.has(user.id)
                            ? "Online now"
                            : "Offline"
                        }
                      />
                      {user.name}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    {user.email}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <span
                      className={`px-2 py-1 rounded text-xs font-medium ${
                        user.role === "admin"
                          ? "bg-purple-500/20 text-purple-400"
                          : "bg-blue-500/20 text-blue-400"
                      }`}
                    >
                      {user.role}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <span
                      className={`px-2 py-1 rounded text-xs font-medium ${
                        user.region === "delhi-ncr"
                          ? "bg-green-500/20 text-green-400"
                          : "bg-blue-500/20 text-blue-400"
                      }`}
                    >
                      {user.region === "delhi-ncr"
                        ? "📍 Delhi-NCR"
                        : "🌏 Pan-India"}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-white">
                    {user.daily_search_limit}
                  </td>
                  <td className="px-4 py-3 text-sm text-white">
                    {user.device_limit}
                  </td>
                  <td className="px-4 py-3 text-sm text-white">
                    {user.searches_used_today}
                  </td>
                  <td className="px-4 py-3 text-sm text-white">
                    <span className="font-semibold text-blue-400">
                      {user.total_searches || 0}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <button
                      onClick={() => handleToggleStatus(user)}
                      className={`flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-all hover:scale-105 ${
                        user.is_active
                          ? "bg-green-500/20 text-green-400 hover:bg-green-500/30"
                          : "bg-red-500/20 text-red-400 hover:bg-red-500/30"
                      }`}
                      title={
                        user.is_active
                          ? "Click to deactivate"
                          : "Click to activate"
                      }
                    >
                      <Power className="h-3 w-3" />
                      {user.is_active ? "Active" : "Inactive"}
                    </button>
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setViewingSessionsUser(user);
                        }}
                        className="text-cyan-400 hover:text-cyan-300 transition-colors"
                        title="View Active Sessions"
                      >
                        <Smartphone className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() =>
                          router.push(`/admin/users/${user.id}/history`)
                        }
                        className="text-green-400 hover:text-green-300 transition-colors"
                        title="View search history"
                      >
                        <History className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() =>
                          router.push(`/admin/users/${user.id}/stats`)
                        }
                        className="text-blue-400 hover:text-blue-300 transition-colors"
                        title="View stats"
                      >
                        <BarChart3 className="h-4 w-4" />
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setEditingUser(user);
                        }}
                        className="text-blue-400 hover:text-blue-300 transition-colors"
                        title="Edit user"
                      >
                        <Edit className="h-4 w-4" />
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setChangingPasswordUser(user);
                        }}
                        className="text-orange-400 hover:text-orange-300 transition-colors"
                        title="Change password"
                      >
                        <Key className="h-4 w-4" />
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleGenerateEOD(user.id, user.name);
                        }}
                        className="text-purple-400 hover:text-purple-300 transition-colors"
                        title="Generate EOD Report"
                      >
                        <Download className="h-4 w-4" />
                      </button>
                      {user.role !== "admin" && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDelete(user.id, user.name);
                          }}
                          className="text-red-400 hover:text-red-300 transition-colors"
                          title="Delete user"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      )}
                    </div>
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

      {users.length === 0 && (
        <div className="text-center py-12 text-gray-400">
          {searchQuery
            ? "No users found matching your search"
            : "No users found"}
        </div>
      )}

      {/* Pagination Controls */}
      {totalPages > 1 && (
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 mt-6 px-1">
          <div className="text-sm text-gray-400">
            Page <span className="text-white font-medium">{currentPage}</span> of{" "}
            <span className="text-white font-medium">{totalPages}</span>
          </div>

          <div className="flex items-center gap-1.5">
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

            <button
              onClick={() => handlePageChange(currentPage + 1)}
              disabled={currentPage === totalPages || fetching}
              className={`p-2 rounded-md border text-sm transition-all ${
                currentPage === totalPages || fetching
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

      {showCreateModal && (
        <CreateUserModal
          token={token}
          onClose={() => setShowCreateModal(false)}
          onSuccess={() => {
            setShowCreateModal(false);
            reloadCurrentPage();
          }}
        />
      )}

      {editingUser && (
        <EditUserModal
          token={token}
          user={editingUser}
          onClose={() => setEditingUser(null)}
          onSuccess={() => {
            setEditingUser(null);
            reloadCurrentPage();
          }}
        />
      )}

      {changingPasswordUser && (
        <ChangePasswordModal
          userId={changingPasswordUser.id}
          userName={changingPasswordUser.name}
          token={token}
          onClose={() => setChangingPasswordUser(null)}
          onSuccess={() => {
            setChangingPasswordUser(null);
            reloadCurrentPage();
          }}
        />
      )}

      {viewingSessionsUser && (
        <UserSessionsModal
          token={token}
          user={viewingSessionsUser}
          onClose={() => setViewingSessionsUser(null)}
        />
      )}
    </div>
  );
}
