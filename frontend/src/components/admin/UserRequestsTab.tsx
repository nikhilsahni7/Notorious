import { Spinner } from "@/components/ui/spinner";
import { adminService, UserRequest } from "@/services/admin.service";
import { Check, X } from "lucide-react";
import { useEffect, useState } from "react";
import { ApproveRequestModal } from "./ApproveRequestModal";

interface UserRequestsTabProps {
  token: string;
}

export function UserRequestsTab({ token }: UserRequestsTabProps) {
  const [loading, setLoading] = useState(true);
  const [requests, setRequests] = useState<UserRequest[]>([]);
  const [filter, setFilter] = useState<"pending" | "approved" | "rejected">(
    "pending"
  );
  const [approvingRequest, setApprovingRequest] = useState<UserRequest | null>(
    null
  );

  useEffect(() => {
    loadRequests();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter]);

  const loadRequests = async () => {
    setLoading(true);
    try {
      const data = await adminService.listUserRequests(token, {
        status: filter,
        limit: 100,
      });
      setRequests(data);
    } catch (error) {
      console.error("Failed to load requests:", error);
      alert("Failed to load requests");
    } finally {
      setLoading(false);
    }
  };

  const handleReject = async (requestId: string, email: string) => {
    const reason = prompt(
      `Reject request from ${email}?\n\nReason (optional):`
    );
    if (reason === null) return;

    try {
      await adminService.rejectUserRequest(token, requestId, reason);
      await loadRequests();
    } catch (error) {
      console.error("Failed to reject request:", error);
      alert("Failed to reject request");
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
        <h2 className="text-xl font-bold text-white">User Access Requests</h2>
        <div className="flex gap-2">
          {(["pending", "approved", "rejected"] as const).map((status) => (
            <button
              key={status}
              onClick={() => setFilter(status)}
              className={`px-4 py-2 rounded text-sm font-medium transition-colors ${
                filter === status
                  ? "bg-pink-500 text-white"
                  : "bg-[#2D1B4E] text-gray-400 hover:text-white"
              }`}
            >
              {status.charAt(0).toUpperCase() + status.slice(1)}
            </button>
          ))}
        </div>
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
                  Phone
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Requested Limit
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Requested On
                </th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                  Status
                </th>
                {filter === "pending" && (
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">
                    Actions
                  </th>
                )}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-700">
              {requests.map((request) => (
                <tr
                  key={request.id}
                  className="hover:bg-[#1a0f2e] transition-colors"
                >
                  <td className="px-4 py-3 text-sm text-white">
                    {request.name}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    {request.email}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    {request.phone}
                  </td>
                  <td className="px-4 py-3 text-sm text-white">
                    {request.requested_searches_per_day}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-300">
                    {new Date(request.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <span
                      className={`px-2 py-1 rounded text-xs font-medium ${
                        request.status === "pending"
                          ? "bg-yellow-500/20 text-yellow-400"
                          : request.status === "approved"
                          ? "bg-green-500/20 text-green-400"
                          : "bg-red-500/20 text-red-400"
                      }`}
                    >
                      {request.status}
                    </span>
                  </td>
                  {filter === "pending" && (
                    <td className="px-4 py-3 text-sm">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => setApprovingRequest(request)}
                          className="text-green-400 hover:text-green-300 transition-colors"
                          title="Approve"
                        >
                          <Check className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() =>
                            handleReject(request.id, request.email)
                          }
                          className="text-red-400 hover:text-red-300 transition-colors"
                          title="Reject"
                        >
                          <X className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {requests.length === 0 && (
        <div className="text-center py-12 text-gray-400">
          No {filter} requests found
        </div>
      )}

      {approvingRequest && (
        <ApproveRequestModal
          token={token}
          request={approvingRequest}
          onClose={() => setApprovingRequest(null)}
          onSuccess={() => {
            setApprovingRequest(null);
            loadRequests();
          }}
        />
      )}
    </div>
  );
}
