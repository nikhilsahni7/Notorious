"use client";

import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useAuth } from "@/contexts/AuthContext";
import {
  PasswordChangeRequest,
  passwordService,
} from "@/services/password.service";
import { format } from "date-fns";
import { ArrowLeft, CheckCircle, Clock, Send, XCircle } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function PasswordChangePage() {
  const { token, isLoading, user } = useAuth();
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [reason, setReason] = useState("");
  const [requests, setRequests] = useState<PasswordChangeRequest[]>([]);
  const [loadingRequests, setLoadingRequests] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    if (!isLoading && !token) {
      router.push("/login");
    }
  }, [isLoading, token, router]);

  useEffect(() => {
    if (token) {
      loadRequests();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const loadRequests = async () => {
    try {
      const data = await passwordService.getUserPasswordRequests(token!);
      setRequests(data);
    } catch (error) {
      console.error("Failed to load requests:", error);
    } finally {
      setLoadingRequests(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setLoading(true);

    try {
      await passwordService.requestPasswordChange(reason, token!);
      setSuccess("Password change request submitted successfully!");
      setReason("");
      loadRequests();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit request");
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "pending":
        return (
          <span className="flex items-center gap-1 text-xs bg-yellow-500/20 text-yellow-400 px-2 py-1 rounded">
            <Clock className="h-3 w-3" />
            Pending
          </span>
        );
      case "approved":
        return (
          <span className="flex items-center gap-1 text-xs bg-green-500/20 text-green-400 px-2 py-1 rounded">
            <CheckCircle className="h-3 w-3" />
            Approved
          </span>
        );
      case "rejected":
        return (
          <span className="flex items-center gap-1 text-xs bg-red-500/20 text-red-400 px-2 py-1 rounded">
            <XCircle className="h-3 w-3" />
            Rejected
          </span>
        );
      default:
        return null;
    }
  };

  if (isLoading || loadingRequests) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-4">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex justify-between items-center mb-4 bg-[#1a0f2e] p-4 rounded-lg border border-gray-700">
          <div className="flex items-center gap-4">
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              size="sm"
              className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E]"
            >
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
            <div>
              <h1 className="text-2xl font-bold text-white">Password Change</h1>
              <p className="text-sm text-gray-400">
                {user?.name} • {user?.email}
              </p>
            </div>
          </div>
        </div>

        {/* Request Form */}
        <div className="bg-[#1a0f2e] p-6 rounded-lg border border-gray-700 mb-4">
          <h2 className="text-lg font-semibold text-white mb-4">
            Request Password Change
          </h2>

          {error && (
            <div className="mb-4 bg-red-500/10 border border-red-500 text-red-400 p-3 rounded">
              {error}
            </div>
          )}

          {success && (
            <div className="mb-4 bg-green-500/10 border border-green-500 text-green-400 p-3 rounded">
              {success}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Reason for Password Change *
              </label>
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Please explain why you need to change your password..."
                className="w-full h-24 bg-[#2D1B4E] border border-gray-600 text-white placeholder:text-gray-500 rounded-md p-3 text-sm resize-none"
                required
              />
            </div>

            <Button
              type="submit"
              disabled={loading || !reason.trim()}
              className="w-full bg-pink-500 hover:bg-pink-600 text-white"
            >
              {loading ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Submitting...
                </>
              ) : (
                <>
                  <Send className="h-4 w-4 mr-2" />
                  Submit Request
                </>
              )}
            </Button>
          </form>
        </div>

        {/* Request History */}
        <div className="bg-[#1a0f2e] rounded-lg border border-gray-700">
          <div className="p-4 border-b border-gray-700">
            <h2 className="text-lg font-semibold text-white">
              Your Password Change Requests
            </h2>
            <p className="text-sm text-gray-400 mt-1">
              {requests.length} total requests
            </p>
          </div>

          {requests.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              <Clock className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p className="text-lg">No password change requests yet</p>
              <p className="text-sm mt-2">
                Submit a request above to get started
              </p>
            </div>
          ) : (
            <div className="divide-y divide-gray-700">
              {requests.map((request) => (
                <div
                  key={request.id}
                  className="p-4 hover:bg-[#2D1B4E] transition-colors"
                >
                  <div className="flex justify-between items-start mb-2">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        {getStatusBadge(request.status)}
                        <span className="text-xs text-gray-500">
                          {format(new Date(request.created_at), "PPP p")}
                        </span>
                      </div>
                      <p className="text-sm text-white mb-2">
                        <span className="text-gray-400">Reason:</span>{" "}
                        {request.reason}
                      </p>
                      {request.admin_notes && (
                        <p className="text-sm text-gray-300 bg-[#2D1B4E] p-2 rounded">
                          <span className="text-gray-500">Admin Response:</span>{" "}
                          {request.admin_notes}
                        </p>
                      )}
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
