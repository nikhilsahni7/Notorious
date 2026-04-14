"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { adminService, UserStats } from "@/services/admin.service";
import {
  ArrowLeft,
  User as UserIcon,
  Search,
  Clock,
  Shield,
  Smartphone,
  TrendingUp,
  BarChart3,
  Calendar,
  History,
  AlertCircle,
} from "lucide-react";

// ── Helper: format numbers with commas ────────────────────────────────────
function fmt(n: number): string {
  return n.toLocaleString();
}

// ── Helper: format date/time string ──────────────────────────────────────
function fmtDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("en-IN", {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

// ── CSS bar chart (horizontal percentage bars) ────────────────────────────
function BarList({
  data,
  maxCount,
  colorClass = "bg-pink-500",
}: {
  data: { label: string; count: number }[];
  maxCount: number;
  colorClass?: string;
}) {
  return (
    <div className="space-y-2">
      {data.map((item, i) => {
        const pct = maxCount > 0 ? (item.count / maxCount) * 100 : 0;
        return (
          <div key={i}>
            <div className="flex justify-between items-center mb-1">
              <span className="text-xs text-gray-300 truncate max-w-[200px]" title={item.label}>
                {item.label}
              </span>
              <span className="text-xs text-gray-400 ml-2 flex-shrink-0">{fmt(item.count)}</span>
            </div>
            <div className="h-1.5 bg-white/10 rounded-full overflow-hidden">
              <div
                className={`h-full ${colorClass} rounded-full transition-all duration-500`}
                style={{ width: `${pct}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ── Mini daily sparkline (vertical bars) ──────────────────────────────────
function DailySparkline({ data }: { data: { date: string; count: number }[] }) {
  if (data.length === 0) {
    return <p className="text-gray-500 text-sm">No search activity in last 30 days</p>;
  }
  const max = Math.max(...data.map((d) => d.count), 1);
  return (
    <div className="flex items-end gap-0.5 h-16 w-full">
      {data.map((d, i) => {
        const heightPct = Math.max((d.count / max) * 100, 4);
        return (
          <div
            key={i}
            className="flex-1 bg-pink-500/50 hover:bg-pink-400 rounded-t transition-colors cursor-default group relative"
            style={{ height: `${heightPct}%` }}
            title={`${d.date}: ${d.count}`}
          >
            <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 hidden group-hover:block bg-[#1a0f2e] border border-gray-600 text-white text-xs rounded px-1.5 py-0.5 whitespace-nowrap z-10">
              {d.date}: {fmt(d.count)}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ── Stat card ─────────────────────────────────────────────────────────────
function StatCard({
  label,
  value,
  sub,
  icon: Icon,
  color = "text-pink-400",
  bg = "bg-pink-500/10",
}: {
  label: string;
  value: string | number;
  sub?: string;
  icon: React.ElementType;
  color?: string;
  bg?: string;
}) {
  return (
    <div className="bg-white/5 backdrop-blur-sm rounded-xl border border-white/10 p-5 hover:border-white/20 transition-all duration-200">
      <div className="flex items-center justify-between mb-3">
        <p className="text-gray-400 text-xs font-medium uppercase tracking-wide">{label}</p>
        <div className={`${bg} p-2 rounded-lg`}>
          <Icon className={`h-4 w-4 ${color}`} />
        </div>
      </div>
      <p className="text-2xl font-bold text-white">{value}</p>
      {sub && <p className="text-gray-500 text-xs mt-1">{sub}</p>}
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
export default function UserStatsPage() {
  const { token, isLoading } = useAuth();
  const router = useRouter();
  const params = useParams();
  const userId = params.id as string;

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState<UserStats | null>(null);

  useEffect(() => {
    if (!isLoading && !token) router.push("/admin/login");
  }, [isLoading, token, router]);

  useEffect(() => {
    if (token && userId) {
      adminService
        .getUserStats(userId, token)
        .then(setStats)
        .catch(() => setError("Failed to load stats for this user."))
        .finally(() => setLoading(false));
    }
  }, [token, userId]);

  if (isLoading || loading) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error || !stats) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center flex-col gap-4">
        <AlertCircle className="h-12 w-12 text-red-400" />
        <p className="text-red-400 text-lg">{error || "Stats unavailable"}</p>
        <Button onClick={() => router.push("/admin")} variant="outline" className="border-gray-600 text-white">
          <ArrowLeft className="h-4 w-4 mr-2" /> Back to Admin
        </Button>
      </div>
    );
  }

  const { identity, sessions, search_behavior, security, engagement } = stats;
  const topTermsForBar = search_behavior.top_terms.map((t) => ({
    label: t.query,
    count: t.count,
  }));
  const maxTermCount = topTermsForBar.length > 0 ? topTermsForBar[0].count : 1;

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-4">
      <div className="max-w-6xl mx-auto space-y-6">

        {/* Header */}
        <div className="flex items-center gap-4 bg-[#1a0f2e] p-4 rounded-xl border border-gray-700">
          <Button
            onClick={() => router.push("/admin")}
            variant="outline"
            size="sm"
            className="bg-transparent border-gray-600 text-white hover:bg-[#2D1B4E]"
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Admin
          </Button>
          <Button
            onClick={() => router.push(`/admin/users/${userId}/history`)}
            variant="outline"
            size="sm"
            className="bg-transparent border-gray-600 text-gray-300 hover:bg-[#2D1B4E]"
          >
            <History className="h-4 w-4 mr-2" />
            Search History
          </Button>
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <UserIcon className="h-5 w-5 text-purple-400" />
              <h1 className="text-xl font-bold text-white">{identity.name}</h1>
              <span
                className={`text-xs px-2 py-0.5 rounded-full font-medium ${
                  identity.is_active
                    ? "bg-green-500/20 text-green-400"
                    : "bg-red-500/20 text-red-400"
                }`}
              >
                {identity.is_active ? "Active" : "Inactive"}
              </span>
            </div>
            <p className="text-gray-400 text-sm mt-0.5">{identity.email} · Stats Overview</p>
          </div>
        </div>

        {/* Identity & Account */}
        <section>
          <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-3 flex items-center gap-2">
            <UserIcon className="h-4 w-4" /> Identity &amp; Account
          </h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard label="Account Created" value={fmtDate(identity.created_at)} icon={Calendar} color="text-blue-400" bg="bg-blue-500/10" />
            <StatCard label="Last Login" value={fmtDate(sessions.last_login)} sub="via active session" icon={Clock} color="text-purple-400" bg="bg-purple-500/10" />
            <StatCard label="Daily Limit" value={fmt(identity.daily_search_limit)} sub="searches / day" icon={Search} color="text-yellow-400" bg="bg-yellow-500/10" />
            <StatCard label="Region" value={identity.region === "delhi-ncr" ? "📍 Delhi-NCR" : "🌏 Pan-India"} icon={UserIcon} color="text-cyan-400" bg="bg-cyan-500/10" />
          </div>
        </section>

        {/* Search Behavior KPIs */}
        <section>
          <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-3 flex items-center gap-2">
            <Search className="h-4 w-4" /> Search Behavior
          </h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard label="Total Searches" value={fmt(search_behavior.total_searches)} icon={Search} color="text-pink-400" bg="bg-pink-500/10" />
            <StatCard label="Avg / Day" value={search_behavior.avg_searches_per_day} sub="since account creation" icon={TrendingUp} color="text-green-400" bg="bg-green-500/10" />
            <StatCard label="Peak Hour" value={search_behavior.peak_hour_formatted} sub="most searches at this hour" icon={Clock} color="text-orange-400" bg="bg-orange-500/10" />
            <StatCard label="Zero-Result Rate" value={`${search_behavior.zero_result_pct}%`} sub={`${fmt(search_behavior.zero_result_searches)} empty searches`} icon={AlertCircle} color="text-red-400" bg="bg-red-500/10" />
          </div>
        </section>

        {/* Daily Volume + Top Terms */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Daily sparkline */}
          <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
            <h3 className="text-white font-semibold mb-1 flex items-center gap-2">
              <BarChart3 className="h-4 w-4 text-pink-400" /> Daily Volume
            </h3>
            <p className="text-gray-500 text-xs mb-4">Last 30 days</p>
            <DailySparkline data={search_behavior.daily_volume} />
            {search_behavior.daily_volume.length > 0 && (
              <div className="flex justify-between text-xs text-gray-600 mt-2">
                <span>{search_behavior.daily_volume[0].date}</span>
                <span>{search_behavior.daily_volume[search_behavior.daily_volume.length - 1].date}</span>
              </div>
            )}
          </div>

          {/* Top terms */}
          <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
            <h3 className="text-white font-semibold mb-1 flex items-center gap-2">
              <Search className="h-4 w-4 text-purple-400" /> Top Search Terms
            </h3>
            <p className="text-gray-500 text-xs mb-4">Top 10 by frequency</p>
            {topTermsForBar.length > 0 ? (
              <BarList data={topTermsForBar} maxCount={maxTermCount} colorClass="bg-purple-500" />
            ) : (
              <p className="text-gray-500 text-sm">No searches yet</p>
            )}
          </div>
        </div>

        {/* Engagement */}
        <section>
          <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-3 flex items-center gap-2">
            <TrendingUp className="h-4 w-4" /> Engagement
          </h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard label="First Search" value={fmtDate(engagement.first_search_at)} icon={Calendar} color="text-sky-400" bg="bg-sky-500/10" />
            <StatCard label="Most Recent Search" value={fmtDate(engagement.last_search_at)} icon={Clock} color="text-green-400" bg="bg-green-500/10" />
            <StatCard label="Longest Gap" value={`${fmt(engagement.longest_gap_days)} days`} sub="between consecutive searches" icon={Calendar} color="text-yellow-400" bg="bg-yellow-500/10" />
            <StatCard label="Searches Used Today" value={fmt(identity.searches_used_today)} sub={`of ${fmt(identity.daily_search_limit)} limit`} icon={Search} color="text-orange-400" bg="bg-orange-500/10" />
          </div>
        </section>

        {/* Security & Devices */}
        <section>
          <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-3 flex items-center gap-2">
            <Shield className="h-4 w-4" /> Security &amp; Devices
          </h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard
              label="Password Resets"
              value={fmt(security.total_password_reset_requests)}
              sub="all time"
              icon={Shield}
              color="text-red-400"
              bg="bg-red-500/10"
            />
            <StatCard
              label="Last Reset Request"
              value={fmtDate(security.last_password_reset_at)}
              icon={Clock}
              color="text-orange-400"
              bg="bg-orange-500/10"
            />
            <StatCard
              label="Devices Registered"
              value={`${fmt(sessions.devices_registered)} / ${fmt(security.device_limit)}`}
              sub={sessions.devices_registered > security.device_limit ? "⚠️ Limit exceeded" : "Within limit"}
              icon={Smartphone}
              color={sessions.devices_registered > security.device_limit ? "text-red-400" : "text-cyan-400"}
              bg={sessions.devices_registered > security.device_limit ? "bg-red-500/10" : "bg-cyan-500/10"}
            />
            <StatCard
              label="Device Limit"
              value={fmt(security.device_limit)}
              sub="configured maximum"
              icon={Smartphone}
              color="text-gray-400"
              bg="bg-gray-500/10"
            />
          </div>
        </section>

      </div>
    </div>
  );
}
