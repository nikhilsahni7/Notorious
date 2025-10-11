import { useEffect, useState } from "react";
import { adminService, AdminSession } from "@/services/admin.service";
import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { Monitor, Smartphone, Tablet, Globe, MapPin, Trash2 } from "lucide-react";
import { format } from "date-fns";

interface SessionsTabProps {
  token: string;
}

export function SessionsTab({ token }: SessionsTabProps) {
  const [loading, setLoading] = useState(true);
  const [sessions, setSessions] = useState<AdminSession[]>([]);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    loadSessions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadSessions = async () => {
    setLoading(true);
    try {
      const data = await adminService.getAdminSessions(token);
      setSessions(data || []);
    } catch (error) {
      console.error("Failed to load sessions:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleInvalidate = async (sessionId: string) => {
    if (!confirm("Are you sure you want to invalidate this session? The admin will be logged out.")) {
      return;
    }

    setDeletingId(sessionId);
    try {
      await adminService.invalidateSession(sessionId, token);
      await loadSessions();
    } catch (error) {
      console.error("Failed to invalidate session:", error);
      alert("Failed to invalidate session");
    } finally {
      setDeletingId(null);
    }
  };

  const getDeviceIcon = (deviceType?: string) => {
    switch (deviceType?.toLowerCase()) {
      case "mobile":
        return <Smartphone className="h-5 w-5" />;
      case "tablet":
        return <Tablet className="h-5 w-5" />;
      default:
        return <Monitor className="h-5 w-5" />;
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
        <div>
          <h2 className="text-xl font-bold text-white">Active Admin Sessions</h2>
          <p className="text-sm text-gray-400 mt-1">
            {sessions.length} active session{sessions.length !== 1 ? "s" : ""}
          </p>
        </div>
        <Button
          onClick={loadSessions}
          variant="outline"
          size="sm"
          className="bg-transparent border-gray-600 text-white hover:bg-gray-700"
        >
          Refresh
        </Button>
      </div>

      {sessions.length === 0 ? (
        <div className="text-center py-12 text-gray-400 bg-[#2D1B4E] rounded-lg border border-gray-700">
          <Monitor className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p className="text-lg">No active sessions</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {sessions.map((session) => (
            <div
              key={session.id}
              className="bg-[#2D1B4E] rounded-lg border border-gray-700 p-4 hover:border-gray-600 transition-colors"
            >
              {/* Header */}
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className="text-blue-400">
                    {getDeviceIcon(session.device_type)}
                  </div>
                  <div>
                    <div className="font-semibold text-white">{session.admin_name}</div>
                    <div className="text-sm text-gray-400">{session.admin_email}</div>
                  </div>
                </div>
                <Button
                  onClick={() => handleInvalidate(session.id)}
                  disabled={deletingId === session.id}
                  variant="outline"
                  size="sm"
                  className="bg-transparent border-red-500 text-red-400 hover:bg-red-500/10 h-8 w-8 p-0"
                >
                  {deletingId === session.id ? (
                    <Spinner size="sm" />
                  ) : (
                    <Trash2 className="h-4 w-4" />
                  )}
                </Button>
              </div>

              {/* Device Info */}
              <div className="space-y-2 mb-3">
                <div className="flex items-center gap-2 text-sm">
                  <Monitor className="h-4 w-4 text-gray-500" />
                  <span className="text-gray-300">
                    {session.browser} {session.browser_version && `(${session.browser_version})`}
                    {" on "}
                    {session.os} {session.os_version && session.os_version}
                  </span>
                </div>

                {session.ip_address && (
                  <div className="flex items-center gap-2 text-sm">
                    <Globe className="h-4 w-4 text-gray-500" />
                    <span className="text-gray-300">
                      {session.ip_address === "127.0.0.1" || session.ip_address === "::1" || session.ip_address === "-:1"
                        ? "Localhost (Local Machine)"
                        : session.ip_address}
                    </span>
                  </div>
                )}

                {(session.city || session.country) && (
                  <div className="flex items-center gap-2 text-sm">
                    <MapPin className="h-4 w-4 text-gray-500" />
                    <span className="text-gray-300">
                      {session.country === "Local" || session.city === "Local Network" ? (
                        <span className="text-gray-500 italic">Local Network (No geolocation)</span>
                      ) : (
                        <>
                          {session.city && `${session.city}, `}
                          {session.country}
                          {session.country_code && ` (${session.country_code})`}
                        </>
                      )}
                    </span>
                  </div>
                )}
              </div>

              {/* Timestamps */}
              <div className="pt-3 border-t border-gray-700 space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-gray-500">Created:</span>
                  <span className="text-gray-400">
                    {format(new Date(session.created_at), "PPp")}
                  </span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-gray-500">Last Used:</span>
                  <span className="text-gray-400">
                    {format(new Date(session.last_used_at), "PPp")}
                  </span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-gray-500">Expires:</span>
                  <span className="text-gray-400">
                    {format(new Date(session.expires_at), "PPp")}
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

