import { useEffect, useState } from "react";
import { passwordService, PasswordChangeRequestWithUser } from "@/services/password.service";
import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { Check, X, Clock, CheckCircle, XCircle } from "lucide-react";
import { format } from "date-fns";

interface PasswordRequestsTabProps {
  token: string;
}

export function PasswordRequestsTab({ token }: PasswordRequestsTabProps) {
  const [loading, setLoading] = useState(true);
  const [requests, setRequests] = useState<PasswordChangeRequestWithUser[]>([]);
  const [filter, setFilter] = useState<"pending" | "approved" | "rejected">("pending");
  const [processingId, setProcessingId] = useState<string | null>(null);

  useEffect(() => {
    loadRequests();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter]);

  const loadRequests = async () => {
    setLoading(true);
    try {
      const data = await passwordService.getPasswordChangeRequests(token, filter);
      setRequests(data || []);
    } catch (error) {
      console.error("Failed to load password requests:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (requestId: string, userName: string) => {
    const newPassword = prompt(`Enter new password for ${userName} (min 6 characters):`);
    if (!newPassword) return;

    if (newPassword.length < 6) {
      alert("Password must be at least 6 characters");
      return;
    }

    const adminNotes = prompt("Admin notes (optional):");

    setProcessingId(requestId);
    try {
      await passwordService.approvePasswordRequest(requestId, newPassword, adminNotes || undefined, token);
      alert("Password change approved successfully!");
      loadRequests();
    } catch (error) {
      console.error("Failed to approve request:", error);
      alert("Failed to approve request");
    } finally {
      setProcessingId(null);
    }
  };

  const handleReject = async (requestId: string) => {
    const reason = prompt("Reason for rejection:");
    if (!reason) return;

    setProcessingId(requestId);
    try {
      await passwordService.rejectPasswordRequest(requestId, reason, token);
      alert("Password change rejected");
      loadRequests();
    } catch (error) {
      console.error("Failed to reject request:", error);
      alert("Failed to reject request");
    } finally {
      setProcessingId(null);
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
        <h2 className="text-xl font-bold text-white">Password Change Requests</h2>
        <div className="flex gap-2">
          <Button
            onClick={() => setFilter("pending")}
            variant={filter === "pending" ? "default" : "outline"}
            size="sm"
            className={filter === "pending" ? "bg-yellow-500 text-black" : "bg-transparent border-gray-600 text-white"}
          >
            <Clock className="h-4 w-4 mr-2" />
            Pending
          </Button>
          <Button
            onClick={() => setFilter("approved")}
            variant={filter === "approved" ? "default" : "outline"}
            size="sm"
            className={filter === "approved" ? "bg-green-500 text-black" : "bg-transparent border-gray-600 text-white"}
          >
            <CheckCircle className="h-4 w-4 mr-2" />
            Approved
          </Button>
          <Button
            onClick={() => setFilter("rejected")}
            variant={filter === "rejected" ? "default" : "outline"}
            size="sm"
            className={filter === "rejected" ? "bg-red-500 text-black" : "bg-transparent border-gray-600 text-white"}
          >
            <XCircle className="h-4 w-4 mr-2" />
            Rejected
          </Button>
        </div>
      </div>

      {requests.length === 0 ? (
        <div className="text-center py-12 text-gray-400 bg-[#2D1B4E] rounded-lg border border-gray-700">
          <Clock className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p className="text-lg">No {filter} requests</p>
        </div>
      ) : (
        <div className="bg-[#2D1B4E] rounded-lg border border-gray-700 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-[#1a0f2e] border-b border-gray-700">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">User</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Reason</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Requested</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Status</th>
                  {filter === "pending" && (
                    <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Actions</th>
                  )}
                  {(filter === "approved" || filter === "rejected") && (
                    <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Admin Notes</th>
                  )}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700">
                {requests.map((request) => (
                  <tr key={request.id} className="hover:bg-[#1a0f2e] transition-colors">
                    <td className="px-4 py-3">
                      <div className="text-sm">
                        <div className="font-medium text-white">{request.user_name}</div>
                        <div className="text-gray-400 text-xs">{request.user_email}</div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300 max-w-xs">
                      {request.reason}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300">
                      {format(new Date(request.created_at), "PPP")}
                      <div className="text-xs text-gray-500">
                        {format(new Date(request.created_at), "p")}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      {request.status === "pending" && (
                        <span className="flex items-center gap-1 text-xs bg-yellow-500/20 text-yellow-400 px-2 py-1 rounded w-fit">
                          <Clock className="h-3 w-3" />
                          Pending
                        </span>
                      )}
                      {request.status === "approved" && (
                        <span className="flex items-center gap-1 text-xs bg-green-500/20 text-green-400 px-2 py-1 rounded w-fit">
                          <CheckCircle className="h-3 w-3" />
                          Approved
                        </span>
                      )}
                      {request.status === "rejected" && (
                        <span className="flex items-center gap-1 text-xs bg-red-500/20 text-red-400 px-2 py-1 rounded w-fit">
                          <XCircle className="h-3 w-3" />
                          Rejected
                        </span>
                      )}
                    </td>
                    {filter === "pending" && (
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => handleApprove(request.id, request.user_name)}
                            disabled={processingId === request.id}
                            className="text-green-400 hover:text-green-300 transition-colors disabled:opacity-50"
                            title="Approve"
                          >
                            <Check className="h-5 w-5" />
                          </button>
                          <button
                            onClick={() => handleReject(request.id)}
                            disabled={processingId === request.id}
                            className="text-red-400 hover:text-red-300 transition-colors disabled:opacity-50"
                            title="Reject"
                          >
                            <X className="h-5 w-5" />
                          </button>
                        </div>
                      </td>
                    )}
                    {(filter === "approved" || filter === "rejected") && (
                      <td className="px-4 py-3 text-sm text-gray-300">
                        {request.admin_notes || "-"}
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

