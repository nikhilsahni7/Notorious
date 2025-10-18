import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { adminService, User } from "@/services/admin.service";
import { X } from "lucide-react";
import { useState } from "react";

interface EditUserModalProps {
  token: string;
  user: User;
  onClose: () => void;
  onSuccess: () => void;
}

export function EditUserModal({
  token,
  user,
  onClose,
  onSuccess,
}: EditUserModalProps) {
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    name: user.name,
    phone: user.phone || "",
    region: user.region || "pan-india",
    daily_search_limit: user.daily_search_limit.toString(),
    is_active: user.is_active,
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      await adminService.updateUser(
        user.id,
        {
          ...formData,
          daily_search_limit: parseInt(formData.daily_search_limit),
        },
        token
      );
      onSuccess();
    } catch (error) {
      console.error("Failed to update user:", error);
      alert(error instanceof Error ? error.message : "Failed to update user");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 w-full max-w-md">
        <div className="flex justify-between items-center p-4 border-b border-gray-700">
          <h3 className="text-lg font-semibold text-white">Edit User</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Email
            </label>
            <Input
              value={user.email}
              className="bg-[#2D1B4E] border-gray-600 text-gray-500"
              disabled
            />
            <p className="text-xs text-gray-500 mt-1">
              Email cannot be changed
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Name *
            </label>
            <Input
              value={formData.name}
              onChange={(e) =>
                setFormData({ ...formData, name: e.target.value })
              }
              className="bg-[#2D1B4E] border-gray-600 text-white"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Phone
            </label>
            <Input
              value={formData.phone}
              onChange={(e) =>
                setFormData({ ...formData, phone: e.target.value })
              }
              className="bg-[#2D1B4E] border-gray-600 text-white"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Region *
            </label>
            <select
              value={formData.region}
              onChange={(e) =>
                setFormData({ ...formData, region: e.target.value })
              }
              className="w-full px-3 py-2 bg-[#2D1B4E] border border-gray-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-pink-500"
              required
            >
              <option value="pan-india">🌏 Pan-India (Access All Data)</option>
              <option value="delhi-ncr">
                📍 Delhi-NCR (Restricted Access)
              </option>
            </select>
            <p className="text-xs text-gray-400 mt-1">
              Pan-India users can search all data. Delhi-NCR users only see
              Delhi data.
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Daily Search Limit *
            </label>
            <Input
              type="number"
              value={formData.daily_search_limit}
              onChange={(e) =>
                setFormData({ ...formData, daily_search_limit: e.target.value })
              }
              className="bg-[#2D1B4E] border-gray-600 text-white"
              min="1"
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              Current usage: {user.searches_used_today} searches
            </p>
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="is_active"
              checked={formData.is_active}
              onChange={(e) =>
                setFormData({ ...formData, is_active: e.target.checked })
              }
              className="rounded"
            />
            <label htmlFor="is_active" className="text-sm text-gray-300">
              Active Account
            </label>
          </div>

          <div className="flex gap-2 pt-4">
            <Button
              type="button"
              onClick={onClose}
              variant="outline"
              className="flex-1 bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E]"
              disabled={loading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              className="flex-1 bg-pink-500 hover:bg-pink-600 text-white"
              disabled={loading}
            >
              {loading ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Saving...
                </>
              ) : (
                "Save Changes"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
