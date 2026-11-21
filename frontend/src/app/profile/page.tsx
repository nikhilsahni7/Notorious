"use client";

import { KnotoriousBrand } from "@/components/KnotoriousBrand";
import { Button } from "@/components/ui/button";
import { NotificationBell } from "@/components/NotificationBell";
import { Spinner } from "@/components/ui/spinner";
import { API_CONFIG } from "@/config/api";
import { useAuth } from "@/contexts/AuthContext";
import { notificationService, NotificationPrefs } from "@/services/notification.service";
import { optInToPush, optOutFromPush, readPushStatus } from "@/lib/push";
import { apiRequest } from "@/lib/api-client";
import { format } from "date-fns";
import {
  ArrowLeft,
  Bell,
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
import { useCallback, useEffect, useState } from "react";

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

// Per-trigger switch rows. Types marked as such are deferred backend-side
// (no trigger writes them yet) and shown disabled so users tune only real ones.
const NOTIFICATION_TRIGGERS: { key: string; label: string; description: string; disabled: boolean }[] = [
  {
    key: "quota_reset",
    label: "Daily quota reset",
    description: "One notification at 12:00 AM IST when your daily searches reset",
    disabled: false,
  },
  {
    key: "quota_warning",
    label: "Quota nearly used",
    description: "Heads-up when you've used 90% of today's searches (max one per day)",
    disabled: false,
  },
  {
    key: "watchlist_match",
    label: "Saved search matches",
    description: "Coming soon — saved searches aren't in the app yet",
    disabled: true,
  },
  {
    key: "export_ready",
    label: "Export ready",
    description: "Coming soon — exports are downloadable immediately today",
    disabled: true,
  },
];

export default function ProfilePage() {
  const { user, token, isLoading } = useAuth();
  const router = useRouter();
  const [metadata, setMetadata] = useState<UserMetadata | null>(null);
  const [loading, setLoading] = useState(true);

  // Notifications settings state
  const [prefs, setPrefs] = useState<NotificationPrefs>({});
  const [prefsLoading, setPrefsLoading] = useState(true);
  const [permission, setPermission] = useState<NotificationPermission | "unsupported">("default");
  const [subscribed, setSubscribed] = useState(false);

  const refreshPermission = useCallback(async () => {
    const status = await readPushStatus();
    setPermission(status.permission);
    setSubscribed(status.subscribed);
  }, []);

  useEffect(() => {
    if (!isLoading && !token) {
      router.push("/login");
    } else if (token && user) {
      loadMetadata();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, token, user, router]);

  useEffect(() => {
    if (!token) return;
    setPrefsLoading(true);
    refreshPermission();
    notificationService
      .getPrefs(token)
      .then((data) => setPrefs(data.prefs || {}))
      .catch((error) => console.error("Failed to load notification prefs:", error))
      .finally(() => setPrefsLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const setTriggerPref = async (key: string, value: boolean) => {
    if (!token || NOTIFICATION_TRIGGERS.find((t) => t.key === key)?.disabled) return;
    const previous = prefs;
    setPrefs({ ...prefs, [key]: value });
    try {
      const data = await notificationService.updatePrefs({ [key]: value }, token);
      setPrefs(data.prefs || {});
    } catch (error) {
      console.error("Failed to update notification prefs:", error);
      setPrefs(previous); // revert silently
    }
  };

  // Browser push opt-in/out from this explicit user gesture only
  const handlePushToggle = async () => {
    if (!token) return;
    if (permission === "granted" && subscribed) {
      await optOutFromPush(token);
    } else {
      const ok = await optInToPush(token); // degrades silently on denial
      if (!ok) {
        await refreshPermission();
        return;
      }
    }
    await refreshPermission();
  };

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
      <div className="min-h-screen bg-[#040f25] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#040f25] p-3 md:p-6 relative overflow-hidden">
      <div className="max-w-4xl mx-auto relative z-10">
        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-4 mb-6 bg-[#0a1c38] p-4 rounded-2xl border border-sky-900/50">
          <div className="flex flex-wrap items-center gap-4">
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              size="sm"
              className="h-8 px-3 bg-transparent border-sky-800 text-sky-200 hover:bg-sky-900/50 hover:border-sky-500 hover:text-white transition-all duration-300 hover:-translate-y-[1px]"
            >
              <ArrowLeft className="h-4 w-4 mr-1.5" />
              Back to Search
            </Button>
            <KnotoriousBrand subtitle="Account and notification settings" size="sm" /><h1 className="sr-only">My profile</h1>
          </div>
          <NotificationBell />
        </div>

        {/* User Info Card */}
        <div className="bg-[#0a1c38] border border-sky-900/50 p-5 md:p-6 rounded-2xl mb-6">
          <h2 className="text-base text-white mb-6 flex items-center gap-3 border-b border-sky-900/50 pb-4">
            <User className="h-5 w-5 text-sky-300" />
            Account Information
          </h2>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="text-xs text-slate-400 mb-2 block">Name</label>
              <div className="text-slate-100 text-sm break-words">{user?.name}</div>
            </div>

            <div>
              <label className="text-xs text-slate-400 mb-2 flex items-center gap-2">
                <Mail className="h-3 w-3" /> Email
              </label>
              <div className="text-slate-100 text-sm break-words">{user?.email}</div>
            </div>

            {user?.phone && (
              <div>
                <label className="text-xs text-slate-400 mb-2 flex items-center gap-2">
                  <Phone className="h-3 w-3" /> Phone
                </label>
                <div className="text-slate-100 text-sm break-words">{user.phone}</div>
              </div>
            )}

            <div>
              <label className="text-xs text-slate-400 mb-2 flex items-center gap-2">
                <Shield className="h-3 w-3" /> Role
              </label>
              <div className="text-slate-100 text-sm break-words capitalize">
                {user?.role}
              </div>
            </div>

            {/* Member Since (User CreatedAt Date) */}
            {user?.created_at ? (
              <div>
                <label className="text-xs text-slate-400 mb-2 flex items-center gap-2">
                  <Calendar className="h-3 w-3" /> Member Since
                </label>
                <div className="text-slate-100 text-sm break-words">
                  {format(new Date(user.created_at), "PPP")}
                </div>
              </div>
            ) : metadata?.created_at ? (
              <div>
                <label className="text-xs text-slate-400 mb-2 flex items-center gap-2">
                  <Calendar className="h-3 w-3" /> Member Since
                </label>
                <div className="text-slate-100 text-sm break-words">
                  {format(new Date(metadata.created_at), "PPP")}
                </div>
              </div>
            ) : null}

            <div>
              <label className="text-xs text-slate-400 mb-2 block">
                Daily Search Limit
              </label>
              <div className="text-slate-100 text-sm break-words">
                {user?.searches_used_today} / {user?.daily_search_limit}{" "}
                <span className="text-xs text-slate-400">used today</span>
              </div>
            </div>

            <div>
              <label className="text-xs text-slate-400 mb-2 block">
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

        {/* Notifications Settings Card */}
        <div className="bg-[#0a1c38] border border-sky-900/50 p-5 md:p-6 rounded-2xl mb-6">
          <h2 className="text-base text-white mb-6 flex items-center gap-3 border-b border-sky-900/50 pb-4">
            <Bell className="h-5 w-5 text-sky-300" />
            Notifications
          </h2>

          <div className="space-y-2">
            {prefsLoading ? (
              <div className="py-4 text-center">
                <Spinner size="sm" />
              </div>
            ) : (
              NOTIFICATION_TRIGGERS.map((trigger) => {
                const checked = prefs[trigger.key] === true;
                return (
                  <label
                    key={trigger.key}
                    className={`flex items-center justify-between gap-4 p-3 rounded-xl border transition-colors ${
                      trigger.disabled
                        ? "border-white/5 opacity-50 cursor-not-allowed"
                        : "border-sky-900/50 hover:bg-sky-900/20 cursor-pointer"
                    }`}
                  >
                    <div className="min-w-0">
                      <p className="text-sm text-white">{trigger.label}</p>
                      <p className="text-xs text-gray-400 break-words">{trigger.description}</p>
                    </div>
                    <button
                      type="button"
                      role="switch"
                      aria-label={trigger.label}
                      aria-checked={checked}
                      disabled={trigger.disabled}
                      onClick={() => setTriggerPref(trigger.key, !checked)}
                      className={`relative flex-shrink-0 w-10 h-6 rounded-full transition-colors ${
                        checked ? "bg-sky-500" : "bg-white/15"
                      } ${trigger.disabled ? "opacity-60" : ""}`}
                    >
                      <span
                        className={`absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-all ${
                          checked ? "left-[1.125rem]" : "left-0.5"
                        }`}
                      />
                    </button>
                  </label>
                );
              })
            )}

            {/* Browser-level push control (OS notifications) */}
            <div className="flex items-center justify-between gap-4 p-3 rounded-xl border border-sky-900/50">
              <div className="min-w-0">
                <p className="text-sm text-white">Browser push (OS notifications)</p>
                <p className="text-xs text-gray-400 break-words">
                  {permission === "unsupported"
                    ? "Not supported in this browser — in-app bell still works"
                    : permission === "denied"
                    ? "Blocked in your browser — in-app bell still works"
                    : subscribed
                    ? "Enrolled ✅ — OS notifications arrive while Knotorious tabs are closed"
                    : permission === "granted"
                    ? "Permission granted but not enrolled — tap Enable to subscribe"
                    : "Off — get OS notifications even when the tab is closed"}
                </p>
              </div>
              <Button
                onClick={handlePushToggle}
                variant="outline"
                size="sm"
                disabled={permission === "unsupported"}
                className="flex-shrink-0 h-8 bg-transparent border-white/10 text-white hover:bg-white/10 hover:border-white/20"
              >
                {permission === "granted" && subscribed ? "Disable" : "Enable"}
              </Button>
            </div>
          </div>
        </div>

        {/* Signup Metadata Card */}
        {metadata ? (
          <div className="bg-[#0a1c38] border border-sky-900/50 p-5 md:p-6 rounded-2xl">
            <h2 className="text-base text-white mb-6 flex items-center gap-3 border-b border-sky-900/50 pb-4">
              <Globe className="h-5 w-5 text-sky-300" />
              Signup Information
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="text-xs text-slate-400 mb-2 flex items-center gap-2">
                  <Calendar className="h-3 w-3" /> Signup Date
                </label>
                <div className="text-slate-100 text-sm break-words">
                  {format(new Date(metadata.created_at), "PPpp")}
                </div>
              </div>

              {metadata.ip_address && (
                <div>
                  <label className="text-xs text-slate-400 mb-2 block">
                    IP Address
                  </label>
                  <div className="text-slate-100 text-sm break-words">
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
                  <label className="text-xs text-slate-400 mb-2 flex items-center gap-2">
                    <MapPin className="h-3 w-3" /> Location
                  </label>
                  <div className="text-slate-100 text-sm break-words">
                    {metadata.country === "Local" ||
                    metadata.city === "Local Network" ? (
                      <span className="text-slate-400">
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
                  <label className="text-xs text-slate-400 mb-2 flex items-center gap-2">
                    <Monitor className="h-3 w-3" /> Device
                  </label>
                  <div className="text-slate-100 text-sm break-words">
                    {metadata.device_type}
                  </div>
                </div>
              )}

              {metadata.browser && (
                <div>
                  <label className="text-xs text-slate-400 mb-2 block">
                    Browser
                  </label>
                  <div className="text-slate-100 text-sm break-words">
                    {metadata.browser}
                    {metadata.browser_version && ` ${metadata.browser_version}`}
                  </div>
                </div>
              )}

              {metadata.os && (
                <div>
                  <label className="text-xs text-slate-400 mb-2 block">
                    Operating System
                  </label>
                  <div className="text-slate-100 text-sm break-words">
                    {metadata.os}
                    {metadata.os_version && ` ${metadata.os_version}`}
                  </div>
                </div>
              )}

              {metadata.timezone && metadata.timezone !== "" && (
                <div>
                  <label className="text-xs text-slate-400 mb-2 block">
                    Timezone
                  </label>
                  <div className="text-slate-100 text-sm break-words">
                    {metadata.timezone}
                  </div>
                </div>
              )}
            </div>

            {metadata.user_agent && (
              <div className="mt-6 pt-6 border-t border-sky-900/50">
                <label className="text-xs text-slate-400 mb-2 block">
                  User Agent
                </label>
                <div className="text-xs text-gray-400 font-mono bg-[#06152d] p-3 rounded-xl border border-white/5 break-all">
                  {metadata.user_agent}
                </div>
              </div>
            )}
          </div>
        ) : (
          !loading && (
            <div className="bg-[#0a1c38] rounded-2xl border border-sky-900/50 p-12 text-center">
              <Globe className="h-12 w-12 mx-auto mb-4 text-white/20" />
              <p className="text-gray-400 text-sm font-semibold">No signup metadata available for this account</p>
            </div>
          )
        )}
      </div>
    </div>
  );
}
