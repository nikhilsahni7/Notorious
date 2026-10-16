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
    <div className="min-h-screen bg-[#2D1B4E] p-3 md:p-6 relative overflow-hidden">
      {/* Navratri background elements */}
      <div className="absolute top-[-10%] left-[-10%] w-[50%] h-[50%] bg-pink-600/5 blur-[120px] rounded-full" />
      <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] bg-blue-600/5 blur-[120px] rounded-full" />

      <div className="max-w-4xl mx-auto relative z-10">
        {/* Header */}
        <div className="flex items-center justify-between mb-6 bg-white/5 backdrop-blur-xl p-4 rounded-2xl border border-white/10 shadow-xl">
          <div className="flex items-center gap-4">
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              size="sm"
              className="h-8 px-3 bg-transparent border-purple-500/50 text-purple-300 hover:bg-purple-500/20 hover:border-purple-400 hover:text-white transition-all duration-300 hover:-translate-y-[1px]"
            >
              <ArrowLeft className="h-4 w-4 mr-1.5" />
              Back to Search
            </Button>
            <h1 className="text-xl font-bold text-white">My Profile</h1>
          </div>
        </div>

        {/* User Info Card */}
        <div className="bg-white/5 backdrop-blur-xl border border-white/10 p-4 md:p-6 rounded-2xl shadow-xl mb-6">
          <h2 className="text-lg font-semibold text-white mb-6 flex items-center gap-2 border-b border-white/10 pb-3">
            <User className="h-5 w-5 text-purple-400" />
            Account Information
          </h2>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block">Name</label>
              <div className="text-white text-base font-bold">{user?.name}</div>
            </div>

            <div>
              <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block flex items-center gap-1">
                <Mail className="h-3 w-3" /> Email
              </label>
              <div className="text-white text-base font-bold">{user?.email}</div>
            </div>

            {user?.phone && (
              <div>
                <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block flex items-center gap-1">
                  <Phone className="h-3 w-3" /> Phone
                </label>
                <div className="text-white text-base font-bold">{user.phone}</div>
              </div>
            )}

            <div>
              <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block flex items-center gap-1">
                <Shield className="h-3 w-3" /> Role
              </label>
              <div className="text-white text-base font-bold capitalize">
                {user?.role}
              </div>
            </div>

            {/* Member Since (User CreatedAt Date) */}
            {user?.created_at ? (
              <div>
                <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block flex items-center gap-1">
                  <Calendar className="h-3 w-3" /> Member Since
                </label>
                <div className="text-white text-base font-bold">
                  {format(new Date(user.created_at), "PPP")}
                </div>
              </div>
            ) : metadata?.created_at ? (
              <div>
                <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block flex items-center gap-1">
                  <Calendar className="h-3 w-3" /> Member Since
                </label>
                <div className="text-white text-base font-bold">
                  {format(new Date(metadata.created_at), "PPP")}
                </div>
              </div>
            ) : null}

            <div>
              <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block">
                Daily Search Limit
              </label>
              <div className="text-white text-base font-bold">
                {user?.searches_used_today} / {user?.daily_search_limit}{" "}
                <span className="text-xs text-gray-400 font-medium">used today</span>
              </div>
            </div>

            <div>
              <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-2 block">
                Account Status
              </label>
              <div className="flex items-center">
                <span
                  className={`px-2.5 py-0.5 rounded-full text-xs font-bold ${
                    user?.is_active
                      ? "bg-green-500/10 border border-green-500/35 text-green-400 shadow-[0_0_10px_rgba(34,197,94,0.1)]"
                      : "bg-red-500/10 border border-red-500/35 text-red-400 shadow-[0_0_10px_rgba(239,68,68,0.1)]"
                  }`}
                >
                  {user?.is_active ? "Active" : "Inactive"}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Signup Metadata Card */}
        {metadata ? (
          <div className="bg-white/5 backdrop-blur-xl border border-white/10 p-4 md:p-6 rounded-2xl shadow-xl">
            <h2 className="text-lg font-semibold text-white mb-6 flex items-center gap-2 border-b border-white/10 pb-3">
              <Globe className="h-5 w-5 text-blue-400" />
              Signup Information
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block flex items-center gap-1">
                  <Calendar className="h-3 w-3" /> Signup Date
                </label>
                <div className="text-white text-base font-bold">
                  {format(new Date(metadata.created_at), "PPpp")}
                </div>
              </div>

              {metadata.ip_address && (
                <div>
                  <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block">
                    IP Address
                  </label>
                  <div className="text-white text-base font-bold">
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
                  <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block flex items-center gap-1">
                    <MapPin className="h-3 w-3" /> Location
                  </label>
                  <div className="text-white text-base font-bold">
                    {metadata.country === "Local" ||
                    metadata.city === "Local Network" ? (
                      <span className="text-gray-500 italic font-medium">
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
                  <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block flex items-center gap-1">
                    <Monitor className="h-3 w-3" /> Device
                  </label>
                  <div className="text-white text-base font-bold">
                    {metadata.device_type}
                  </div>
                </div>
              )}

              {metadata.browser && (
                <div>
                  <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block">
                    Browser
                  </label>
                  <div className="text-white text-base font-bold">
                    {metadata.browser}
                    {metadata.browser_version && ` ${metadata.browser_version}`}
                  </div>
                </div>
              )}

              {metadata.os && (
                <div>
                  <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block">
                    Operating System
                  </label>
                  <div className="text-white text-base font-bold">
                    {metadata.os}
                    {metadata.os_version && ` ${metadata.os_version}`}
                  </div>
                </div>
              )}

              {metadata.timezone && metadata.timezone !== "" && (
                <div>
                  <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-1 block">
                    Timezone
                  </label>
                  <div className="text-white text-base font-bold">
                    {metadata.timezone}
                  </div>
                </div>
              )}
            </div>

            {metadata.user_agent && (
              <div className="mt-6 pt-6 border-t border-white/10">
                <label className="text-[10px] text-gray-400 uppercase font-black tracking-wider mb-2 block">
                  User Agent
                </label>
                <div className="text-xs text-gray-400 font-mono bg-[#0f0820]/45 p-3 rounded-xl border border-white/5 break-all">
                  {metadata.user_agent}
                </div>
              </div>
            )}
          </div>
        ) : (
          !loading && (
            <div className="bg-white/5 backdrop-blur-xl rounded-2xl border border-white/10 p-12 text-center shadow-lg">
              <Globe className="h-12 w-12 mx-auto mb-4 text-white/20" />
              <p className="text-gray-400 text-sm font-semibold">No signup metadata available for this account</p>
            </div>
          )
        )}
      </div>
    </div>
  );
}
