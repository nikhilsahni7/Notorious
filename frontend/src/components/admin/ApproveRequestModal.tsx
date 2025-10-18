import { Button } from "@/components/ui/button";
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
  const [adminNote, setAdminNote] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      await adminService.approveUserRequest(
        request.id,
        token,
        adminNote || undefined
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
            <p className="text-gray-300 text-sm mt-2">
              <span className="text-gray-400">Requested searches/day:</span>{" "}
              {request.requested_searches_per_day}
            </p>
          </div>

          <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-3">
            <p className="text-sm text-blue-300">
              ℹ️ <strong>Note:</strong> Approving this request marks it as
              approved. You can create the user account later using the
              &quot;Create User&quot; button.
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-1">
              Admin Note (Optional)
            </label>
            <textarea
              value={adminNote}
              onChange={(e) => setAdminNote(e.target.value)}
              placeholder="Why are you approving this request? (e.g., Verified company email, Known user, etc.)"
              rows={4}
              className="w-full px-3 py-2 bg-[#2D1B4E] border border-gray-600 rounded-md text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-pink-500"
            />
            <p className="text-xs text-gray-400 mt-1">
              This note will help you remember why you approved this request
              later.
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
                "Approve Request"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
