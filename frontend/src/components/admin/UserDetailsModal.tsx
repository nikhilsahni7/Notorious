import { useEffect, useState } from "react";
import { adminService, UserWithMetadata } from "@/services/admin.service";
import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { X, User, Globe, MapPin, Monitor, Calendar } from "lucide-react";
import { format } from "date-fns";

interface UserDetailsModalProps {
  userId: string;
  token: string;
  onClose: () => void;
}

export function UserDetailsModal({ userId, token, onClose }: UserDetailsModalProps) {
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<UserWithMetadata | null>(null);

  useEffect(() => {
    loadUserDetails();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userId]);

  const loadUserDetails = async () => {
    setLoading(true);
    try {
      const result = await adminService.getUserDetails(userId, token);
      setData(result);
    } catch (error) {
      console.error("Failed to load user details:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex justify-between items-center p-4 border-b border-gray-700 sticky top-0 bg-[#1a0f2e]">
          <div className="flex items-center gap-2">
            <User className="h-5 w-5 text-blue-400" />
            <h2 className="text-lg font-bold text-white">User Details</h2>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body */}
        <div className="p-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Spinner size="lg" />
            </div>
          ) : data ? (
            <div className="space-y-6">
              {/* User Information */}
              <div>
                <h3 className="text-md font-semibold text-white mb-3 flex items-center gap-2">
                  <User className="h-4 w-4" />
                  User Information
                </h3>
                <div className="bg-[#2D1B4E] rounded-lg p-4 space-y-2">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="text-xs text-gray-500">Name</label>
                      <div className="text-sm text-white">{data.user.name}</div>
                    </div>
                    <div>
                      <label className="text-xs text-gray-500">Email</label>
                      <div className="text-sm text-white">{data.user.email}</div>
                    </div>
                    <div>
                      <label className="text-xs text-gray-500">Phone</label>
                      <div className="text-sm text-white">{data.user.phone || "N/A"}</div>
                    </div>
                    <div>
                      <label className="text-xs text-gray-500">Role</label>
                      <div className="text-sm text-white capitalize">{data.user.role}</div>
                    </div>
                    <div>
                      <label className="text-xs text-gray-500">Daily Search Limit</label>
                      <div className="text-sm text-white">{data.user.daily_search_limit}</div>
                    </div>
                    <div>
                      <label className="text-xs text-gray-500">Searches Used Today</label>
                      <div className="text-sm text-white">{data.user.searches_used_today}</div>
                    </div>
                    <div>
                      <label className="text-xs text-gray-500">Status</label>
                      <div className="text-sm">
                        {data.user.is_active ? (
                          <span className="text-green-400">Active</span>
                        ) : (
                          <span className="text-red-400">Inactive</span>
                        )}
                      </div>
                    </div>
                    <div>
                      <label className="text-xs text-gray-500">Created</label>
                      <div className="text-sm text-white">
                        {format(new Date(data.user.created_at), "PPp")}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Signup Metadata */}
              {data.metadata ? (
                <div>
                  <h3 className="text-md font-semibold text-white mb-3 flex items-center gap-2">
                    <Globe className="h-4 w-4" />
                    Signup Information
                  </h3>
                  <div className="bg-[#2D1B4E] rounded-lg p-4 space-y-3">
                    {data.metadata.ip_address && (
                      <div className="flex items-start gap-2">
                        <Globe className="h-4 w-4 text-gray-500 mt-0.5" />
                        <div className="flex-1">
                          <label className="text-xs text-gray-500">IP Address</label>
                          <div className="text-sm text-white font-mono">{data.metadata.ip_address}</div>
                        </div>
                      </div>
                    )}

                    {(data.metadata.city || data.metadata.country) && (
                      <div className="flex items-start gap-2">
                        <MapPin className="h-4 w-4 text-gray-500 mt-0.5" />
                        <div className="flex-1">
                          <label className="text-xs text-gray-500">Location</label>
                          <div className="text-sm text-white">
                            {data.metadata.city && `${data.metadata.city}, `}
                            {data.metadata.country}
                            {data.metadata.country_code && ` (${data.metadata.country_code})`}
                          </div>
                          {data.metadata.latitude && data.metadata.longitude && (
                            <div className="text-xs text-gray-400 mt-1">
                              Coordinates: {data.metadata.latitude.toFixed(4)}, {data.metadata.longitude.toFixed(4)}
                            </div>
                          )}
                          {data.metadata.timezone && (
                            <div className="text-xs text-gray-400">
                              Timezone: {data.metadata.timezone}
                            </div>
                          )}
                        </div>
                      </div>
                    )}

                    {(data.metadata.device_type || data.metadata.browser || data.metadata.os) && (
                      <div className="flex items-start gap-2">
                        <Monitor className="h-4 w-4 text-gray-500 mt-0.5" />
                        <div className="flex-1">
                          <label className="text-xs text-gray-500">Device Information</label>
                          <div className="text-sm text-white">
                            {data.metadata.device_type && (
                              <div>{data.metadata.device_type}</div>
                            )}
                            {data.metadata.browser && (
                              <div>
                                Browser: {data.metadata.browser}
                                {data.metadata.browser_version && ` ${data.metadata.browser_version}`}
                              </div>
                            )}
                            {data.metadata.os && (
                              <div>
                                OS: {data.metadata.os}
                                {data.metadata.os_version && ` ${data.metadata.os_version}`}
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    )}

                    <div className="flex items-start gap-2">
                      <Calendar className="h-4 w-4 text-gray-500 mt-0.5" />
                      <div className="flex-1">
                        <label className="text-xs text-gray-500">Signed Up</label>
                        <div className="text-sm text-white">
                          {format(new Date(data.metadata.created_at), "PPpp")}
                        </div>
                      </div>
                    </div>

                    {data.metadata.user_agent && (
                      <div className="pt-2 border-t border-gray-700">
                        <label className="text-xs text-gray-500">User Agent</label>
                        <div className="text-xs text-gray-400 font-mono break-all mt-1">
                          {data.metadata.user_agent}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              ) : (
                <div className="bg-[#2D1B4E] rounded-lg p-6 text-center">
                  <Globe className="h-12 w-12 mx-auto mb-3 text-gray-600" />
                  <p className="text-gray-400">No signup metadata available</p>
                  <p className="text-sm text-gray-500 mt-1">
                    This user was created before metadata tracking was enabled
                  </p>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center py-12 text-gray-400">
              Failed to load user details
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-gray-700 sticky bottom-0 bg-[#1a0f2e]">
          <Button
            onClick={onClose}
            variant="outline"
            className="w-full bg-transparent border-gray-600 text-white hover:bg-gray-700"
          >
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}

