import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { adminService, User } from "@/services/admin.service";
import {
  Check,
  Download,
  Edit,
  History,
  Key,
  Plus,
  Trash2,
  X,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { ChangePasswordModal } from "./ChangePasswordModal";
import { CreateUserModal } from "./CreateUserModal";
import { EditUserModal } from "./EditUserModal";

interface UsersTabProps {
  token: string;
}

export function UsersTab({ token }: UsersTabProps) {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [users, setUsers] = useState<User[]>([]);
  const [filteredUsers, setFilteredUsers] = useState<User[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [regionFilter, setRegionFilter] = useState<
    "all" | "pan-india" | "delhi-ncr"
  >("all");
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [changingPasswordUser, setChangingPasswordUser] = useState<User | null>(
    null
  );

  useEffect(() => {
    loadUsers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    let filtered = users;

    // Apply region filter
    if (regionFilter !== "all") {
      filtered = filtered.filter((user) => user.region === regionFilter);
    }

    // Apply search query
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (user) =>
          user.name.toLowerCase().includes(query) ||
          user.email.toLowerCase().includes(query) ||
          user.phone?.toLowerCase().includes(query)
      );
    }

    setFilteredUsers(filtered);
  }, [searchQuery, regionFilter, users]);

  const loadUsers = async () => {
    try {
      const data = await adminService.listUsers(token, 100);
      setUsers(data);
      setFilteredUsers(data);
    } catch (error) {
      console.error("Failed to load users:", error);
      alert("Failed to load users");
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (userId: string, userName: string) => {
    if (!confirm(`Are you sure you want to delete user "${userName}"?`)) {
      return;
    }

    try {
      await adminService.deleteUser(userId, token);
      await loadUsers();
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
        <h2 className="text-xl font-bold text-white">User Management</h2>
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
          All Users ({users.length})
        </button>
        <button
          onClick={() => setRegionFilter("pan-india")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            regionFilter === "pan-india"
              ? "bg-blue-500 text-white"
              : "bg-[#2D1B4E] text-gray-300 hover:bg-[#3D2B5E]"
          }`}
        >
          🌏 Pan-India ({users.filter((u) => u.region === "pan-india").length})
        </button>
        <button
          onClick={() => setRegionFilter("delhi-ncr")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            regionFilter === "delhi-ncr"
              ? "bg-green-500 text-white"
              : "bg-[#2D1B4E] text-gray-300 hover:bg-[#3D2B5E]"
          }`}
        >
          📍 Delhi-NCR ({users.filter((u) => u.region === "delhi-ncr").length})
        </button>
      </div>

      <div className="bg-[#2D1B4E] rounded-lg border border-gray-700 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-[#1a0f2e] border-b border-gray-700">
              <tr>
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
              {filteredUsers.map((user) => (
                <tr
                  key={user.id}
                  className="hover:bg-[#1a0f2e] transition-colors"
                >
                  <td className="px-4 py-3 text-sm text-white">{user.name}</td>
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
                    {user.searches_used_today}
                  </td>
                  <td className="px-4 py-3 text-sm text-white">
                    <span className="font-semibold text-blue-400">
                      {user.total_searches || 0}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm">
                    {user.is_active ? (
                      <span className="flex items-center text-green-400">
                        <Check className="h-4 w-4 mr-1" />
                        Active
                      </span>
                    ) : (
                      <span className="flex items-center text-red-400">
                        <X className="h-4 w-4 mr-1" />
                        Inactive
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <div className="flex items-center gap-2">
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
      </div>

      {filteredUsers.length === 0 && (
        <div className="text-center py-12 text-gray-400">
          {searchQuery
            ? "No users found matching your search"
            : "No users found"}
        </div>
      )}

      {showCreateModal && (
        <CreateUserModal
          token={token}
          onClose={() => setShowCreateModal(false)}
          onSuccess={() => {
            setShowCreateModal(false);
            loadUsers();
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
            loadUsers();
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
            loadUsers();
          }}
        />
      )}
    </div>
  );
}
