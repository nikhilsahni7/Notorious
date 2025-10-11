"use client";

import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { API_CONFIG } from "@/config/api";
import { useAuth } from "@/contexts/AuthContext";
import { apiRequest } from "@/lib/api-client";
import { format } from "date-fns";
import {
  ArrowLeft,
  Calendar,
  Globe,
  Mail,
  MapPin,
  Monitor,
  Phone,
  Shield,
  User,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

interface UserMetadata {
  ip_address?: string;
  country?: string;
  country_code?: string;
  city?: string;
  latitude?: number;
  longitude?: number;
  timezone?: string;
  device_type?: string;
  browser?: string;
  browser_version?: string;
  os?: string;
  os_version?: string;
  user_agent?: string;
  created_at: string;
}

export default function ProfilePage() {
  const { user, token, isLoading } = useAuth();
  const router = useRouter();
  const [metadata, setMetadata] = useState<UserMetadata | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!isLoading && !token) {
      router.push("/login");
    } else if (token && user) {
      loadMetadata();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, token, user, router]);

  const loadMetadata = async () => {
    try {
      const data = await apiRequest<UserMetadata>(
        API_CONFIG.ENDPOINTS.USER.METADATA,
        {
          method: "GET",
          token: token!,
        }
      );
      setMetadata(data);
    } catch (error) {
      console.error("Failed to load metadata:", error);
    } finally {
      setLoading(false);
    }
  };

  if (isLoading || loading) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-6">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              size="sm"
              className="bg-transparent border-gray-600 text-gray-300"
            >
              <ArrowLeft className="h-4 w-4 mr-1" />
              Back to Search
            </Button>
            <h1 className="text-2xl font-bold text-white">My Profile</h1>
          </div>
        </div>

        {/* User Info Card */}
        <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 p-6 mb-6">
          <h2 className="text-xl font-semibold text-white mb-4 flex items-center gap-2">
            <User className="h-5 w-5 text-purple-400" />
            Account Information
          </h2>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="text-sm text-gray-400 mb-1 block">Name</label>
              <div className="text-white font-medium">{user?.name}</div>
            </div>

            <div>
              <label className="text-sm text-gray-400 mb-1 block flex items-center gap-1">
                <Mail className="h-3 w-3" /> Email
              </label>
              <div className="text-white font-medium">{user?.email}</div>
            </div>

            {user?.phone && (
              <div>
                <label className="text-sm text-gray-400 mb-1 block flex items-center gap-1">
                  <Phone className="h-3 w-3" /> Phone
                </label>
                <div className="text-white font-medium">{user.phone}</div>
              </div>
            )}

            <div>
              <label className="text-sm text-gray-400 mb-1 block flex items-center gap-1">
                <Shield className="h-3 w-3" /> Role
              </label>
              <div className="text-white font-medium capitalize">
                {user?.role}
              </div>
            </div>

            <div>
              <label className="text-sm text-gray-400 mb-1 block">
                Daily Search Limit
              </label>
              <div className="text-white font-medium">
                {user?.searches_used_today} / {user?.daily_search_limit}{" "}
                searches used today
              </div>
            </div>

            <div>
              <label className="text-sm text-gray-400 mb-1 block">
                Account Status
              </label>
              <div>
                <span
                  className={`px-2 py-1 rounded text-xs font-medium ${
                    user?.is_active
                      ? "bg-green-500/20 text-green-400"
                      : "bg-red-500/20 text-red-400"
                  }`}
                >
                  {user?.is_active ? "Active" : "Inactive"}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Signup Metadata Card */}
        {metadata && (
          <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 p-6">
            <h2 className="text-xl font-semibold text-white mb-4 flex items-center gap-2">
              <Globe className="h-5 w-5 text-blue-400" />
              Signup Information
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="text-sm text-gray-400 mb-1 block flex items-center gap-1">
                  <Calendar className="h-3 w-3" /> Signup Date
                </label>
                <div className="text-white font-medium">
                  {format(new Date(metadata.created_at), "PPpp")}
                </div>
              </div>

              {metadata.ip_address && (
                <div>
                  <label className="text-sm text-gray-400 mb-1 block">
                    IP Address
                  </label>
                  <div className="text-white font-medium">
                    {metadata.ip_address === "127.0.0.1" ||
                    metadata.ip_address === "::1" ||
                    metadata.ip_address === "-:1"
                      ? "Localhost (Local Machine)"
                      : metadata.ip_address}
                  </div>
                </div>
              )}

              {(metadata.city || metadata.country) && (
                <div>
                  <label className="text-sm text-gray-400 mb-1 block flex items-center gap-1">
                    <MapPin className="h-3 w-3" /> Location
                  </label>
                  <div className="text-white font-medium">
                    {metadata.country === "Local" ||
                    metadata.city === "Local Network" ? (
                      <span className="text-gray-500 italic">
                        Local Network (No geolocation)
                      </span>
                    ) : (
                      <>
                        {metadata.city && `${metadata.city}, `}
                        {metadata.country}
                        {metadata.country_code && ` (${metadata.country_code})`}
                      </>
                    )}
                  </div>
                </div>
              )}

              {metadata.device_type && (
                <div>
                  <label className="text-sm text-gray-400 mb-1 block flex items-center gap-1">
                    <Monitor className="h-3 w-3" /> Device
                  </label>
                  <div className="text-white font-medium">
                    {metadata.device_type}
                  </div>
                </div>
              )}

              {metadata.browser && (
                <div>
                  <label className="text-sm text-gray-400 mb-1 block">
                    Browser
                  </label>
                  <div className="text-white font-medium">
                    {metadata.browser}
                    {metadata.browser_version && ` ${metadata.browser_version}`}
                  </div>
                </div>
              )}

              {metadata.os && (
                <div>
                  <label className="text-sm text-gray-400 mb-1 block">
                    Operating System
                  </label>
                  <div className="text-white font-medium">
                    {metadata.os}
                    {metadata.os_version && ` ${metadata.os_version}`}
                  </div>
                </div>
              )}

              {metadata.timezone && metadata.timezone !== "" && (
                <div>
                  <label className="text-sm text-gray-400 mb-1 block">
                    Timezone
                  </label>
                  <div className="text-white font-medium">
                    {metadata.timezone}
                  </div>
                </div>
              )}
            </div>

            {metadata.user_agent && (
              <div className="mt-6 pt-6 border-t border-gray-700">
                <label className="text-sm text-gray-400 mb-2 block">
                  User Agent
                </label>
                <div className="text-xs text-gray-500 font-mono bg-[#0f0820] p-3 rounded border border-gray-800 break-all">
                  {metadata.user_agent}
                </div>
              </div>
            )}
          </div>
        )}

        {!metadata && !loading && (
          <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 p-12 text-center">
            <p className="text-gray-400">No signup metadata available</p>
          </div>
        )}
      </div>
    </div>
  );
}
