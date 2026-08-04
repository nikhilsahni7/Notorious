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
import { useEffect, useState, type MouseEvent, type ReactNode } from "react";
import { ChangePasswordModal } from "./ChangePasswordModal";
import { CreateUserModal } from "./CreateUserModal";
import { EditUserModal } from "./EditUserModal";
import { UserSessionsModal } from "./UserSessionsModal";

interface UsersTabProps {
  token: string;
}

function ActionButton({
  title,
  onClick,
  className,
  children,
}: {
  title: string;
  onClick: (e: MouseEvent) => void;
  className: string;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      className={`inline-flex h-8 w-8 items-center justify-center rounded-md transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-pink-500/40 ${className}`}
    >
      {children}
    </button>
  );
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
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-xl font-semibold tracking-tight text-white">
            User Management
          </h2>
          <p className="mt-1 text-sm text-gray-400">
            Showing{" "}
            <span className="text-gray-300 tabular-nums">{users.length}</span> of{" "}
            <span className="text-gray-300 tabular-nums">{totalCount}</span>{" "}
            total users
          </p>
        </div>
        <Button
          onClick={() => setShowCreateModal(true)}
          className="bg-pink-500 hover:bg-pink-600 text-white shadow-sm shadow-pink-500/20"
        >
          <Plus className="h-4 w-4 mr-2" />
          Create User
        </Button>
      </div>

      <div className="space-y-3">
        <Input
          placeholder="Search users by name, email, or phone..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="bg-[#2D1B4E]/80 border-gray-600/80 text-white placeholder:text-gray-500 focus-visible:ring-pink-500/30"
        />

        <div className="flex flex-wrap gap-2">
          {(
            [
              {
                key: "all" as const,
                label: "All Users",
                count: counts.all,
                active: "bg-pink-500 text-white shadow-sm shadow-pink-500/25",
              },
              {
                key: "pan-india" as const,
                label: "Pan-India",
                count: counts["pan-india"],
                active: "bg-blue-500 text-white shadow-sm shadow-blue-500/25",
              },
              {
                key: "delhi-ncr" as const,
                label: "Delhi-NCR",
                count: counts["delhi-ncr"],
                active: "bg-emerald-500 text-white shadow-sm shadow-emerald-500/25",
              },
            ] as const
          ).map((filter) => (
            <button
              key={filter.key}
              type="button"
              onClick={() => setRegionFilter(filter.key)}
              className={`rounded-lg px-3.5 py-1.5 text-sm font-medium transition-colors duration-150 ${
                regionFilter === filter.key
                  ? filter.active
                  : "bg-[#2D1B4E]/80 text-gray-400 hover:bg-[#3D2B5E] hover:text-gray-200"
              }`}
            >
              {filter.label}
              <span
                className={`ml-1.5 tabular-nums ${
                  regionFilter === filter.key ? "text-white/80" : "text-gray-500"
                }`}
              >
                ({filter.count})
              </span>
            </button>
          ))}
        </div>
      </div>

      {selectedUserIds.size > 0 && (
        <div className="flex flex-col gap-3 rounded-xl border border-purple-500/25 bg-[#2D1B4E]/90 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
          <span className="flex items-center gap-2 text-sm font-medium text-white">
            <CheckSquare className="h-4 w-4 text-purple-400" />
            {selectedUserIds.size} user{selectedUserIds.size === 1 ? "" : "s"}{" "}
            selected
          </span>
          <div className="flex flex-wrap items-center gap-2">
            <Button
              onClick={() => handleBulkUpdate(true)}
              disabled={bulkUpdating}
              className="bg-emerald-600 hover:bg-emerald-700 text-white text-sm"
              size="sm"
            >
              <UserCheck className="h-4 w-4 mr-1.5" />
              Activate
            </Button>
            <Button
              onClick={() => handleBulkUpdate(false)}
              disabled={bulkUpdating}
              className="bg-red-600 hover:bg-red-700 text-white text-sm"
              size="sm"
            >
              <UserX className="h-4 w-4 mr-1.5" />
              Deactivate
            </Button>
            <Button
              onClick={clearSelection}
              variant="outline"
              className="border-gray-600/80 text-gray-300 hover:bg-gray-700/50 text-sm"
              size="sm"
            >
              <X className="h-4 w-4 mr-1.5" />
              Clear
            </Button>
          </div>
        </div>
      )}

      <div className="relative overflow-hidden rounded-xl border border-gray-700/80 bg-[#2D1B4E]/60">
        <div
          className={`overflow-x-auto transition-opacity duration-200 ${
            fetching ? "opacity-40" : "opacity-100"
          }`}
        >
          <table className="w-full min-w-[1100px]">
            <thead className="border-b border-gray-700/80 bg-[#1a0f2e]/90">
              <tr>
                <th className="w-12 px-3 py-3.5 text-left">
                  <button
                    type="button"
                    onClick={toggleSelectAll}
                    className="rounded-md p-1 text-gray-400 transition-colors hover:bg-white/5 hover:text-white"
                    title={
                      selectedUserIds.size === users.length
                        ? "Deselect all"
                        : "Select all"
                    }
                  >
                    {selectedUserIds.size === users.length &&
                    users.length > 0 ? (
                      <CheckSquare className="h-4 w-4 text-purple-400" />
                    ) : (
                      <Square className="h-4 w-4" />
                    )}
                  </button>
                </th>
                {[
                  "Name",
                  "Email",
                  "Role",
                  "Region",
                  "Daily Limit",
                  "Device Limit",
                  "Used Today",
                  "Total Searches",
                  "Status",
                  "Actions",
                ].map((label) => (
                  <th
                    key={label}
                    className={`px-3 py-3.5 text-left text-xs font-medium uppercase tracking-wider text-gray-500 ${
                      label === "Actions" ? "pr-4" : ""
                    }`}
                  >
                    {label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800/80">
              {users.map((user) => (
                <tr
                  key={user.id}
                  className={`transition-colors duration-150 hover:bg-white/[0.03] ${
                    selectedUserIds.has(user.id) ? "bg-purple-500/[0.08]" : ""
                  }`}
                >
                  <td className="px-3 py-2.5">
                    <button
                      type="button"
                      onClick={() => toggleSelectUser(user.id)}
                      className="rounded-md p-1 text-gray-400 transition-colors hover:bg-white/5 hover:text-white"
                    >
                      {selectedUserIds.has(user.id) ? (
                        <CheckSquare className="h-4 w-4 text-purple-400" />
                      ) : (
                        <Square className="h-4 w-4" />
                      )}
                    </button>
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-2.5">
                      <span
                        className={`h-2 w-2 flex-shrink-0 rounded-full ${
                          onlineUserIds.has(user.id)
                            ? "bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.55)]"
                            : "bg-gray-600"
                        }`}
                        title={
                          onlineUserIds.has(user.id) ? "Online now" : "Offline"
                        }
                      />
                      <span className="text-sm font-medium text-white">
                        {user.name}
                      </span>
                    </div>
                  </td>
                  <td className="px-3 py-2.5 text-sm text-gray-400">
                    {user.email}
                  </td>
                  <td className="px-3 py-2.5">
                    <span
                      className={`inline-flex rounded-md px-2 py-0.5 text-xs font-medium capitalize ${
                        user.role === "admin"
                          ? "bg-purple-500/15 text-purple-300"
                          : "bg-sky-500/15 text-sky-300"
                      }`}
                    >
                      {user.role}
                    </span>
                  </td>
                  <td className="px-3 py-2.5">
                    <span
                      className={`inline-flex rounded-md px-2 py-0.5 text-xs font-medium ${
                        user.region === "delhi-ncr"
                          ? "bg-emerald-500/15 text-emerald-300"
                          : "bg-blue-500/15 text-blue-300"
                      }`}
                    >
                      {user.region === "delhi-ncr" ? "Delhi-NCR" : "Pan-India"}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-sm tabular-nums text-gray-300">
                    {user.daily_search_limit}
                  </td>
                  <td className="px-3 py-2.5 text-sm tabular-nums text-gray-300">
                    {user.device_limit}
                  </td>
                  <td className="px-3 py-2.5 text-sm tabular-nums text-gray-300">
                    {user.searches_used_today}
                  </td>
                  <td className="px-3 py-2.5 text-sm tabular-nums font-medium text-sky-300">
                    {user.total_searches || 0}
                  </td>
                  <td className="px-3 py-2.5">
                    <button
                      type="button"
                      onClick={() => handleToggleStatus(user)}
                      className={`inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium transition-colors duration-150 ${
                        user.is_active
                          ? "bg-emerald-500/15 text-emerald-300 hover:bg-emerald-500/25"
                          : "bg-red-500/15 text-red-300 hover:bg-red-500/25"
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
                  <td className="px-3 py-2.5 pr-4">
                    <div className="flex items-center gap-0.5">
                      <ActionButton
                        title="View Active Sessions"
                        onClick={(e) => {
                          e.stopPropagation();
                          setViewingSessionsUser(user);
                        }}
                        className="text-cyan-400/90 hover:bg-cyan-500/10 hover:text-cyan-300"
                      >
                        <Smartphone className="h-3.5 w-3.5" />
                      </ActionButton>
                      <ActionButton
                        title="View search history"
                        onClick={() =>
                          router.push(`/admin/users/${user.id}/history`)
                        }
                        className="text-emerald-400/90 hover:bg-emerald-500/10 hover:text-emerald-300"
                      >
                        <History className="h-3.5 w-3.5" />
                      </ActionButton>
                      <ActionButton
                        title="View stats"
                        onClick={() =>
                          router.push(`/admin/users/${user.id}/stats`)
                        }
                        className="text-sky-400/90 hover:bg-sky-500/10 hover:text-sky-300"
                      >
                        <BarChart3 className="h-3.5 w-3.5" />
                      </ActionButton>
                      <ActionButton
                        title="Edit user"
                        onClick={(e) => {
                          e.stopPropagation();
                          setEditingUser(user);
                        }}
                        className="text-blue-400/90 hover:bg-blue-500/10 hover:text-blue-300"
                      >
                        <Edit className="h-3.5 w-3.5" />
                      </ActionButton>
                      <ActionButton
                        title="Change password"
                        onClick={(e) => {
                          e.stopPropagation();
                          setChangingPasswordUser(user);
                        }}
                        className="text-amber-400/90 hover:bg-amber-500/10 hover:text-amber-300"
                      >
                        <Key className="h-3.5 w-3.5" />
                      </ActionButton>
                      <ActionButton
                        title="Generate EOD Report"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleGenerateEOD(user.id, user.name);
                        }}
                        className="text-violet-400/90 hover:bg-violet-500/10 hover:text-violet-300"
                      >
                        <Download className="h-3.5 w-3.5" />
                      </ActionButton>
                      {user.role !== "admin" && (
                        <>
                          <span className="mx-0.5 h-4 w-px bg-gray-700" />
                          <ActionButton
                            title="Delete user"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDelete(user.id, user.name);
                            }}
                            className="text-red-400/80 hover:bg-red-500/10 hover:text-red-300"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </ActionButton>
                        </>
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
        <div className="py-12 text-center text-sm text-gray-500">
          {searchQuery
            ? "No users found matching your search"
            : "No users found"}
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex flex-col items-center justify-between gap-4 px-1 sm:flex-row">
          <div className="text-sm text-gray-500">
            Page{" "}
            <span className="font-medium tabular-nums text-gray-300">
              {currentPage}
            </span>{" "}
            of{" "}
            <span className="font-medium tabular-nums text-gray-300">
              {totalPages}
            </span>
          </div>

          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => handlePageChange(currentPage - 1)}
              disabled={currentPage === 1 || fetching}
              className={`rounded-lg border p-2 text-sm transition-colors ${
                currentPage === 1 || fetching
                  ? "cursor-not-allowed border-gray-800 bg-[#1a0f2e]/50 text-gray-600 opacity-50"
                  : "border-gray-700/80 bg-[#1a0f2e]/80 text-gray-300 hover:border-gray-600 hover:bg-[#2D1B4E] hover:text-white"
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
                    className="px-2 py-1.5 text-sm text-gray-600"
                  >
                    …
                  </span>
                );
              }

              const isCurrent = pageNum === currentPage;
              return (
                <button
                  key={`page-${pageNum}`}
                  type="button"
                  onClick={() => handlePageChange(pageNum as number)}
                  disabled={fetching}
                  className={`min-w-[2.25rem] rounded-lg px-2.5 py-1.5 text-sm font-medium transition-colors ${
                    isCurrent
                      ? "bg-pink-500 text-white shadow-sm shadow-pink-500/25"
                      : "border border-gray-700/80 bg-[#1a0f2e]/80 text-gray-300 hover:bg-[#2D1B4E] hover:text-white"
                  }`}
                >
                  {pageNum}
                </button>
              );
            })}

            <button
              type="button"
              onClick={() => handlePageChange(currentPage + 1)}
              disabled={currentPage === totalPages || fetching}
              className={`rounded-lg border p-2 text-sm transition-colors ${
                currentPage === totalPages || fetching
                  ? "cursor-not-allowed border-gray-800 bg-[#1a0f2e]/50 text-gray-600 opacity-50"
                  : "border-gray-700/80 bg-[#1a0f2e]/80 text-gray-300 hover:border-gray-600 hover:bg-[#2D1B4E] hover:text-white"
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
