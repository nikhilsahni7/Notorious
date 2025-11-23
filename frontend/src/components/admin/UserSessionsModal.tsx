import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { adminService, User, UserWithMetadata } from "@/services/admin.service";
import { Laptop, Monitor, Smartphone, X } from "lucide-react";
import { useEffect, useState } from "react";

interface UserSessionsModalProps {
  token: string;
  user: User;
  onClose: () => void;
}

export function UserSessionsModal({
  token,
  user,
  onClose,
}: UserSessionsModalProps) {
  const [loading, setLoading] = useState(true);
  const [details, setDetails] = useState<UserWithMetadata | null>(null);

  useEffect(() => {
    loadDetails();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadDetails = async () => {
    try {
      const data = await adminService.getUserDetails(user.id, token);
      setDetails(data);
    } catch (error) {
      console.error("Failed to load user details:", error);
    } finally {
      setLoading(false);
    }
  };

  const getDeviceIcon = (type: string) => {
    switch (type?.toLowerCase()) {
      case "mobile":
        return <Smartphone className="h-5 w-5" />;
      case "desktop":
        return <Monitor className="h-5 w-5" />;
      default:
        return <Laptop className="h-5 w-5" />;
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString("en-IN", {
      timeZone: "Asia/Kolkata",
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 w-full max-w-4xl max-h-[90vh] overflow-y-auto shadow-xl">
        <div className="p-6 border-b border-gray-700 flex justify-between items-center sticky top-0 bg-[#1a0f2e] z-10">
          <div>
            <h2 className="text-xl font-bold text-white flex items-center gap-2">
              Active Sessions
              <span className="text-sm font-normal text-gray-400 bg-gray-800 px-2 py-0.5 rounded-full border border-gray-700">
                {details?.sessions?.length || 0} / {user.device_limit}
              </span>
            </h2>
            <p className="text-sm text-gray-400">
              Managing devices for <span className="text-white">{user.name}</span>
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            <X className="h-6 w-6" />
          </button>
        </div>

        <div className="p-6">
          {loading ? (
            <div className="flex justify-center py-12">
              <Spinner size="lg" />
            </div>
          ) : !details?.sessions || details.sessions.length === 0 ? (
            <div className="text-center py-12 text-gray-400 bg-[#2D1B4E]/30 rounded-lg border border-gray-700 border-dashed">
              <Laptop className="h-12 w-12 mx-auto mb-3 opacity-20" />
              <p>No active sessions found for this user.</p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {details.sessions.map((session, index) => (
                <div
                  key={session.id}
                  className="bg-[#2D1B4E] rounded-lg border border-gray-600 p-4 relative group hover:border-pink-500/50 transition-colors"
                >
                  <div className="absolute top-3 right-3 text-xs font-mono text-gray-500 bg-[#1a0f2e] px-1.5 py-0.5 rounded border border-gray-700">
                    #{index + 1}
                  </div>

                  <div className="flex items-start gap-3 mb-4">
                    <div className="bg-pink-500/10 p-2 rounded-lg text-pink-400 border border-pink-500/20">
                      {getDeviceIcon(session.device_type)}
                    </div>
                    <div>
                      <h3 className="font-medium text-white text-sm leading-tight mb-1">
                        {session.device_name || "Unknown Device"}
                      </h3>
                      <p className="text-xs text-gray-400">
                        {session.device_os} • {session.device_type}
                      </p>
                    </div>
                  </div>

                  <div className="space-y-2 text-xs">
                    <div className="flex justify-between py-1 border-b border-gray-700/50">
                      <span className="text-gray-500">IP Address</span>
                      <span className="text-gray-300 font-mono">
                        {session.ip_address}
                      </span>
                    </div>
                    <div className="flex justify-between py-1 border-b border-gray-700/50">
                      <span className="text-gray-500">Location</span>
                      <span className="text-gray-300 text-right truncate max-w-[60%]">
                        {session.location || "Unknown"}
                      </span>
                    </div>
                    <div className="flex justify-between py-1 border-b border-gray-700/50">
                      <span className="text-gray-500">Last Active</span>
                      <span className="text-gray-300 text-right">
                        {formatDate(session.last_active)}
                      </span>
                    </div>
                    <div className="flex justify-between py-1">
                      <span className="text-gray-500">Logged In</span>
                      <span className="text-gray-300 text-right">
                        {formatDate(session.created_at)}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="p-6 border-t border-gray-700 bg-[#1a0f2e] rounded-b-lg flex justify-end">
          <Button onClick={onClose} variant="outline">
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}
