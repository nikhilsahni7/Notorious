import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { adminService, UserRequest } from "@/services/admin.service";
import { X } from "lucide-react";
import { useState } from "react";

interface ApproveRequestModalProps {
  token: string;
  request: UserRequest;
  onClose: () => void;
  onSuccess: () => void;
}

export function ApproveRequestModal({
  token,
  request,
  onClose,
  onSuccess,
}: ApproveRequestModalProps) {
  const [loading, setLoading] = useState(false);
  const [password, setPassword] = useState("");
  const [region, setRegion] = useState("pan-india");
  const [dailyLimit, setDailyLimit] = useState(
    request.requested_searches_per_day.toString()
  );

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      await adminService.approveUserRequest(
        request.id,
        password,
        region,
        parseInt(dailyLimit),
        token
      );
      onSuccess();
    } catch (error) {
      console.error("Failed to approve request:", error);
      alert(
        error instanceof Error ? error.message : "Failed to approve request"
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 w-full max-w-md">
        <div className="flex justify-between items-center p-4 border-b border-gray-700">
          <h3 className="text-lg font-semibold text-white">
            Approve Access Request
          </h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          <div className="bg-[#2D1B4E] p-3 rounded border border-gray-700">
            <p className="text-sm text-gray-400">Requesting User</p>
            <p className="text-white font-medium">{request.name}</p>
            <p className="text-gray-300 text-sm">{request.email}</p>
            <p className="text-gray-300 text-sm">{request.phone}</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Set Password *
            </label>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Create password for user"
              className="bg-[#2D1B4E] border-gray-600 text-white"
              minLength={6}
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              Minimum 6 characters. Share this with the user.
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Region *
            </label>
            <select
              value={region}
              onChange={(e) => setRegion(e.target.value)}
              className="w-full px-3 py-2 bg-[#2D1B4E] border border-gray-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-pink-500"
              required
            >
              <option value="pan-india">🌏 Pan-India (Access All Data)</option>
              <option value="delhi-ncr">
                📍 Delhi-NCR (Restricted Access)
              </option>
            </select>
            <p className="text-xs text-gray-400 mt-1">
              Choose access level for this user
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Daily Search Limit *
            </label>
            <Input
              type="number"
              value={dailyLimit}
              onChange={(e) => setDailyLimit(e.target.value)}
              className="bg-[#2D1B4E] border-gray-600 text-white"
              min="1"
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              Requested: {request.requested_searches_per_day} searches/day
            </p>
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
              className="flex-1 bg-green-500 hover:bg-green-600 text-white"
              disabled={loading}
            >
              {loading ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Approving...
                </>
              ) : (
                "Approve & Create User"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
