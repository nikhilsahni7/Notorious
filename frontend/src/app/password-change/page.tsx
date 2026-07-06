"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [directLoading, setDirectLoading] = useState(false);
  const [directError, setDirectError] = useState<string | null>(null);
  const [directSuccess, setDirectSuccess] = useState<string | null>(null);

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

  const handleDirectSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setDirectError(null);
    setDirectSuccess(null);

    if (newPassword !== confirmPassword) {
      setDirectError("New passwords do not match");
      return;
    }

    if (newPassword.length < 6) {
      setDirectError("New password must be at least 6 characters");
      return;
    }

    setDirectLoading(true);

    try {
      await passwordService.changePassword(currentPassword, newPassword, token!);
      setDirectSuccess("Password changed successfully!");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      loadRequests();
    } catch (err) {
      setDirectError(err instanceof Error ? err.message : "Failed to change password");
    } finally {
      setDirectLoading(false);
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
      <div className="min-h-screen bg-[#0a0515] flex items-center justify-center">
        <Spinner size="lg" className="text-pink-500" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-4 relative overflow-hidden">
      {/* Navratri background elements */}
      <div className="absolute top-[-10%] left-[-10%] w-[50%] h-[50%] bg-pink-600/5 blur-[120px] rounded-full" />
      <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] bg-blue-600/5 blur-[120px] rounded-full" />

      <div className="max-w-4xl mx-auto relative z-10">
        {/* Header */}
        <div className="flex justify-between items-center mb-4 bg-white/5 backdrop-blur-xl p-4 rounded-xl border border-white/10 shadow-xl">
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

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          {/* Direct Password Change Form */}
          <div className="bg-white/5 backdrop-blur-xl p-6 rounded-xl border border-white/10 shadow-xl flex flex-col h-full">
            <h2 className="text-lg font-semibold text-white mb-4">
              Change Password Directly
            </h2>

            {directError && (
              <div className="mb-4 bg-red-500/10 border border-red-500/30 text-red-400 p-3 rounded-lg text-sm">
                {directError}
              </div>
            )}

            {directSuccess && (
              <div className="mb-4 bg-green-500/10 border border-green-500/30 text-green-400 p-3 rounded-lg text-sm">
                {directSuccess}
              </div>
            )}

            <form onSubmit={handleDirectSubmit} className="flex-1 flex flex-col justify-between">
              <div className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
                    Current Password *
                  </label>
                  <Input
                    type="password"
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                    placeholder="Enter current password"
                    className="bg-[#2D1B4E]/50 border-gray-600 text-white placeholder:text-gray-500 h-10 text-sm focus-visible:ring-pink-500/20 focus-visible:border-pink-500/50"
                    required
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
                    New Password *
                  </label>
                  <Input
                    type="password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    placeholder="Min 6 characters"
                    className="bg-[#2D1B4E]/50 border-gray-600 text-white placeholder:text-gray-500 h-10 text-sm focus-visible:ring-pink-500/20 focus-visible:border-pink-500/50"
                    required
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
                    Confirm New Password *
                  </label>
                  <Input
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="Confirm new password"
                    className="bg-[#2D1B4E]/50 border-gray-600 text-white placeholder:text-gray-500 h-10 text-sm focus-visible:ring-pink-500/20 focus-visible:border-pink-500/50"
                    required
                  />
                </div>
              </div>

              <Button
                type="submit"
                disabled={directLoading || !currentPassword || !newPassword || !confirmPassword}
                className="w-full bg-gradient-to-r from-pink-500 to-rose-600 hover:from-pink-600 hover:to-rose-700 text-white h-11 border-none shadow-[0_0_15px_rgba(236,72,153,0.3)] transition-all duration-300 font-bold mt-6"
              >
                {directLoading ? (
                  <>
                    <Spinner size="sm" className="mr-2" />
                    Updating...
                  </>
                ) : (
                  "Update Password"
                )}
              </Button>
            </form>
          </div>

          {/* Request Form */}
          <div className="bg-white/5 backdrop-blur-xl p-6 rounded-xl border border-white/10 shadow-xl flex flex-col h-full">
            <h2 className="text-lg font-semibold text-white mb-4">
              Request Password Change
            </h2>

            {error && (
              <div className="mb-4 bg-red-500/10 border border-red-500/30 text-red-400 p-3 rounded-lg text-sm">
                {error}
              </div>
            )}

            {success && (
              <div className="mb-4 bg-green-500/10 border border-green-500/30 text-green-400 p-3 rounded-lg text-sm">
                {success}
              </div>
            )}

            <form onSubmit={handleSubmit} className="flex-1 flex flex-col justify-between">
              <div className="flex-1 flex flex-col">
                <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
                  Reason for Password Change *
                </label>
                <textarea
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="Explain why you need admin to reset/change your password..."
                  className="flex-1 min-h-[180px] bg-[#2D1B4E]/50 border border-gray-600 text-white placeholder:text-gray-500 rounded-md p-3 text-sm resize-none focus:outline-none focus:ring-1 focus:ring-pink-500/20 focus:border-pink-500/50"
                  required
                />
              </div>

              <Button
                type="submit"
                disabled={loading || !reason.trim()}
                className="w-full bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 text-white h-11 border-none shadow-[0_0_15px_rgba(59,130,246,0.3)] transition-all duration-300 font-bold mt-6"
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
        </div>

        {/* Request History */}
        <div className="bg-white/5 backdrop-blur-xl rounded-xl border border-white/10 shadow-xl overflow-hidden">
          <div className="p-4 border-b border-white/10">
            <h2 className="text-lg font-semibold text-white">
              Your Password Activity Logs
            </h2>
            <p className="text-sm text-gray-400 mt-1">
              {requests.length} total logs
            </p>
          </div>

          {requests.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              <Clock className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p className="text-lg">No password activity logged yet</p>
              <p className="text-sm mt-2">
                Use the forms above to update your password
              </p>
            </div>
          ) : (
            <div className="divide-y divide-white/10">
              {requests.map((request) => {
                const isDirectChange = request.reason === "Self-service password change";
                return (
                  <div
                    key={request.id}
                    className="p-4 hover:bg-[#2D1B4E] transition-colors"
                  >
                    <div className="flex justify-between items-start mb-2">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-2">
                          {isDirectChange ? (
                            <span className="flex items-center gap-1 text-xs bg-green-500/20 text-green-400 px-2 py-1 rounded font-bold">
                              <CheckCircle className="h-3 w-3" />
                              Success (Direct)
                            </span>
                          ) : (
                            getStatusBadge(request.status)
                          )}
                          <span className="text-xs text-gray-500">
                            {format(new Date(request.created_at), "PPP p")}
                          </span>
                        </div>
                        <p className="text-sm text-white mb-2">
                          <span className="text-gray-400">Action:</span>{" "}
                          {isDirectChange ? (
                            <span className="text-green-300 font-medium">Direct self-service password update (Applied Instantly)</span>
                          ) : (
                            request.reason
                          )}
                        </p>
                        {!isDirectChange && request.admin_notes && (
                          <p className="text-sm text-gray-300 bg-[#2D1B4E] p-2 rounded">
                            <span className="text-gray-500">Admin Response:</span>{" "}
                            {request.admin_notes}
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
