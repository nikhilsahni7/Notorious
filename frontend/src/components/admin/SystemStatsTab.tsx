"use client";

import { useEffect, useState } from "react";
import { adminService, SystemStats } from "@/services/admin.service";
import { Spinner } from "@/components/ui/spinner";
import {
  Search,
  Users,
  TrendingUp,
  Shield,
  Clock,
  BarChart3,
  AlertCircle,
  Smartphone,
  Calendar,
  UserCheck,
} from "lucide-react";

interface SystemStatsTabProps {
  token: string;
}

// ── Helpers ──────────────────────────────────────────────────────────────────
function fmt(n: number): string {
  return n.toLocaleString();
}

// ── CSS stat card ─────────────────────────────────────────────────────────────
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
    <div className="bg-white/5 backdrop-blur-sm rounded-xl border border-white/10 p-5 hover:border-white/20 transition-all duration-200 group">
      <div className="flex items-center justify-between mb-3">
        <p className="text-gray-400 text-xs font-medium uppercase tracking-wide">{label}</p>
        <div className={`${bg} p-2 rounded-lg group-hover:scale-105 transition-transform`}>
          <Icon className={`h-4 w-4 ${color}`} />
        </div>
      </div>
      <p className="text-2xl font-bold text-white">{value}</p>
      {sub && <p className="text-gray-500 text-xs mt-1">{sub}</p>}
    </div>
  );
}

// ── Horizontal bar list ───────────────────────────────────────────────────────
function BarList({
  data,
  maxCount,
  colorClass = "bg-pink-500",
  limit,
}: {
  data: { label: string; count: number }[];
  maxCount: number;
  colorClass?: string;
  limit?: number;
}) {
  const visible = limit ? data.slice(0, limit) : data;
  return (
    <div className="space-y-2">
      {visible.map((item, i) => {
        const pct = maxCount > 0 ? (item.count / maxCount) * 100 : 0;
        return (
          <div key={i}>
            <div className="flex justify-between items-center mb-1">
              <span className="text-xs text-gray-300 truncate max-w-[55%]" title={item.label}>
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

// ── Vertical bar chart (for hours 0-23 and day-of-week) ──────────────────────
function VerticalBarChart({
  data,
  colorClass = "bg-pink-500/60 hover:bg-pink-400",
  height = "h-20",
}: {
  data: { label: string; count: number }[];
  colorClass?: string;
  height?: string;
}) {
  if (data.length === 0) return <p className="text-gray-500 text-sm">No data</p>;
  const max = Math.max(...data.map((d) => d.count), 1);
  return (
    <div className={`flex items-end gap-0.5 ${height} w-full`}>
      {data.map((d, i) => {
        const heightPct = Math.max((d.count / max) * 100, 3);
        return (
          <div
            key={i}
            className={`flex-1 ${colorClass} rounded-t transition-colors cursor-default group relative`}
            style={{ height: `${heightPct}%` }}
          >
            <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 hidden group-hover:block bg-[#1a0f2e] border border-gray-600 text-white text-xs rounded px-1.5 py-0.5 whitespace-nowrap z-10">
              {d.label}: {fmt(d.count)}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ── Section wrapper ───────────────────────────────────────────────────────────
function Section({
  title,
  icon: Icon,
  children,
}: {
  title: string;
  icon: React.ElementType;
  children: React.ReactNode;
}) {
  return (
    <section>
      <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-3 flex items-center gap-2">
        <Icon className="h-4 w-4" />
        {title}
      </h2>
      {children}
    </section>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
export function SystemStatsTab({ token }: SystemStatsTabProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState<SystemStats | null>(null);

  useEffect(() => {
    adminService
      .getSystemStats(token)
      .then(setStats)
      .catch(() => setError("Failed to load system stats."))
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error || !stats) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-3">
        <AlertCircle className="h-10 w-10 text-red-400" />
        <p className="text-red-400">{error || "Stats unavailable"}</p>
      </div>
    );
  }

  const sv = stats.search_volume;
  const up = stats.user_patterns;
  const sp = stats.search_patterns;
  const sec = stats.security;
  const td = stats.time_distributions;

  // Prepare bar list data for top terms
  const topTermsBars = sp.top_terms.map((t) => ({ label: t.query, count: t.count }));
  const maxTermCount = topTermsBars.length > 0 ? topTermsBars[0].count : 1;

  // Most active users bar data
  const topUserBars = up.most_active_users.map((u) => ({
    label: u.name,
    count: u.search_count,
  }));
  const maxUserSearches = topUserBars.length > 0 ? topUserBars[0].count : 1;

  // Hour distribution for vertical chart
  // Build all 24 hours, filling gaps with 0
  const hourMap = new Map(td.by_hour.map((h) => [h.hour, h.count]));
  const allHours = Array.from({ length: 24 }, (_, i) => ({
    label: i % 6 === 0 ? (i === 0 ? "12 AM" : i === 12 ? "12 PM" : `${i < 12 ? i : i - 12} ${i < 12 ? "AM" : "PM"}`) : "",
    count: hourMap.get(i) ?? 0,
    fullLabel: i === 0 ? "12 AM" : i === 12 ? "12 PM" : `${i < 12 ? i : i - 12} ${i < 12 ? "AM" : "PM"}`,
  })).map(h => ({ label: h.fullLabel, count: h.count }));

  // DoW chart
  const dowMap = new Map(td.by_day_of_week.map((d) => [d.dow, d]));
  const dowNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const allDow = Array.from({ length: 7 }, (_, i) => ({
    label: dowNames[i],
    count: dowMap.get(i)?.count ?? 0,
  }));

  return (
    <div className="space-y-8 animate-in fade-in duration-300">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-white">System Stats</h2>
          <p className="text-gray-400 text-sm mt-1">Aggregated analytics across all users and searches</p>
        </div>
        <div className="text-right">
          <p className="text-gray-500 text-xs">Peak system hour</p>
          <p className="text-white font-bold text-lg">{sv.peak_hour_formatted}</p>
        </div>
      </div>

      {/* ── Search Volume ───────────────────────────────────────────────── */}
      <Section title="Search Volume" icon={Search}>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <StatCard label="Total Searches" value={fmt(sv.total_all_time)} sub="all time" icon={Search} color="text-pink-400" bg="bg-pink-500/10" />
          <StatCard label="Last 30 Days" value={fmt(sv.total_last_30_days)} icon={Calendar} color="text-blue-400" bg="bg-blue-500/10" />
          <StatCard label="Avg / Day" value={sv.avg_daily} sub="since first search" icon={TrendingUp} color="text-green-400" bg="bg-green-500/10" />
          <StatCard label="Avg / User / Day" value={sv.avg_per_user_per_day} icon={UserCheck} color="text-purple-400" bg="bg-purple-500/10" />
        </div>

        {/* Daily trend – last 90 days vertical bars */}
        <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
          <h3 className="text-white font-semibold mb-1">Search Trend — Last 90 Days</h3>
          <p className="text-gray-500 text-xs mb-4">Each bar = one day</p>
          <VerticalBarChart
            data={sv.daily_trend.map((d) => ({ label: d.date, count: d.count }))}
            colorClass="bg-blue-500/50 hover:bg-blue-400"
            height="h-24"
          />
          {sv.daily_trend.length > 0 && (
            <div className="flex justify-between text-xs text-gray-600 mt-2">
              <span>{sv.daily_trend[0].date}</span>
              <span>{sv.daily_trend[sv.daily_trend.length - 1].date}</span>
            </div>
          )}
        </div>
      </Section>

      {/* ── User Patterns ───────────────────────────────────────────────── */}
      <Section title="User Patterns" icon={Users}>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-6">
          <StatCard label="Total Users" value={fmt(up.total_users)} icon={Users} color="text-blue-400" bg="bg-blue-500/10" />
          <StatCard label="Active (Last 30d)" value={fmt(up.active_users_last_30d)} sub="searched at least once" icon={UserCheck} color="text-green-400" bg="bg-green-500/10" />
          <StatCard label="Avg Searches / User" value={up.avg_searches_per_user} sub="all time" icon={Search} color="text-yellow-400" bg="bg-yellow-500/10" />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Most active users */}
          <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
            <h3 className="text-white font-semibold mb-1 flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-green-400" /> Most Active Users
            </h3>
            <p className="text-gray-500 text-xs mb-4">Top 10 by search count</p>
            {topUserBars.length > 0 ? (
              <BarList data={topUserBars} maxCount={maxUserSearches} colorClass="bg-green-500" limit={10} />
            ) : (
              <p className="text-gray-500 text-sm">No data</p>
            )}
          </div>

          {/* Device distribution */}
          <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
            <h3 className="text-white font-semibold mb-1 flex items-center gap-2">
              <Smartphone className="h-4 w-4 text-cyan-400" /> Device Count Distribution
            </h3>
            <p className="text-gray-500 text-xs mb-4">Users grouped by number of registered devices</p>
            {up.device_distribution.length > 0 ? (
              <div className="space-y-3">
                {up.device_distribution.map((d, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <span className="text-gray-400 text-xs w-20 flex-shrink-0">
                      {d.device_count} device{d.device_count !== 1 ? "s" : ""}
                    </span>
                    <div className="flex-1 h-2 bg-white/10 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-cyan-500 rounded-full"
                        style={{ width: `${(d.user_count / Math.max(...up.device_distribution.map(x => x.user_count))) * 100}%` }}
                      />
                    </div>
                    <span className="text-gray-300 text-xs w-12 text-right">{fmt(d.user_count)} users</span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-gray-500 text-sm">No session data</p>
            )}
          </div>
        </div>
      </Section>

      {/* ── Search Patterns ─────────────────────────────────────────────── */}
      <Section title="Search Patterns" icon={Search}>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
            <h3 className="text-white font-semibold mb-1 flex items-center gap-2">
              <BarChart3 className="h-4 w-4 text-pink-400" /> Top 20 Search Terms
            </h3>
            <p className="text-gray-500 text-xs mb-4">System-wide frequency</p>
            {topTermsBars.length > 0 ? (
              <BarList data={topTermsBars} maxCount={maxTermCount} colorClass="bg-pink-500" limit={20} />
            ) : (
              <p className="text-gray-500 text-sm">No searches yet</p>
            )}
          </div>
          <div className="flex flex-col gap-4">
            <StatCard
              label="Zero-Result Searches"
              value={fmt(sp.zero_result_count)}
              sub={`${sp.zero_result_pct}% of total searches`}
              icon={AlertCircle}
              color="text-red-400"
              bg="bg-red-500/10"
            />
            <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
              <p className="text-gray-400 text-xs uppercase tracking-wide mb-2">Zero-Result Rate</p>
              <div className="h-2 bg-white/10 rounded-full overflow-hidden">
                <div
                  className="h-full bg-red-500 rounded-full transition-all duration-700"
                  style={{ width: `${Math.min(parseFloat(sp.zero_result_pct), 100)}%` }}
                />
              </div>
              <div className="flex justify-between text-xs text-gray-500 mt-1">
                <span>0%</span>
                <span className="text-white font-medium">{sp.zero_result_pct}%</span>
                <span>100%</span>
              </div>
            </div>
          </div>
        </div>
      </Section>

      {/* ── Time Distributions ──────────────────────────────────────────── */}
      <Section title="Time Distributions" icon={Clock}>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* By hour */}
          <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
            <h3 className="text-white font-semibold mb-1">Searches by Hour of Day</h3>
            <p className="text-gray-500 text-xs mb-4">IST timezone · peak at {sv.peak_hour_formatted}</p>
            <VerticalBarChart data={allHours} colorClass="bg-purple-500/50 hover:bg-purple-400" height="h-20" />
            <div className="flex justify-between text-xs text-gray-600 mt-2">
              <span>12 AM</span>
              <span>6 AM</span>
              <span>12 PM</span>
              <span>6 PM</span>
              <span>11 PM</span>
            </div>
          </div>

          {/* By day of week */}
          <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
            <h3 className="text-white font-semibold mb-1">Searches by Day of Week</h3>
            <p className="text-gray-500 text-xs mb-4">IST timezone</p>
            <VerticalBarChart data={allDow} colorClass="bg-yellow-500/50 hover:bg-yellow-400" height="h-20" />
            <div className="flex justify-between text-xs text-gray-600 mt-2">
              {["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map((d) => (
                <span key={d}>{d}</span>
              ))}
            </div>
          </div>
        </div>

        {/* By month */}
        {td.by_month.length > 0 && (
          <div className="mt-6 bg-[#1a0f2e] border border-gray-700 rounded-xl p-5">
            <h3 className="text-white font-semibold mb-1">Searches by Month — Last 12 Months</h3>
            <p className="text-gray-500 text-xs mb-4">IST timezone</p>
            <VerticalBarChart
              data={td.by_month.map((m) => ({ label: m.month, count: m.count }))}
              colorClass="bg-green-500/50 hover:bg-green-400"
              height="h-20"
            />
            <div className="flex justify-between text-xs text-gray-600 mt-2 overflow-hidden">
              {td.by_month.length > 0 && <span>{td.by_month[0].month}</span>}
              {td.by_month.length > 1 && <span>{td.by_month[td.by_month.length - 1].month}</span>}
            </div>
          </div>
        )}
      </Section>

      {/* ── Security ────────────────────────────────────────────────────── */}
      <Section title="Security" icon={Shield}>
        <div className="grid grid-cols-2 md:grid-cols-2 gap-4 mb-6">
          <StatCard label="Total Password Resets" value={fmt(sec.total_password_resets)} sub="all time" icon={Shield} color="text-orange-400" bg="bg-orange-500/10" />
          <StatCard label="Resets (Last 30d)" value={fmt(sec.password_resets_last_30_days)} icon={Calendar} color="text-yellow-400" bg="bg-yellow-500/10" />
        </div>

        {sec.users_exceeding_device_limit.length > 0 && (
          <div className="bg-red-500/5 border border-red-500/20 rounded-xl p-5">
            <h3 className="text-red-400 font-semibold mb-3 flex items-center gap-2">
              <AlertCircle className="h-4 w-4" /> Users Exceeding Device Limit
            </h3>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-gray-400 border-b border-gray-700">
                    <th className="pb-2 pr-4">Name</th>
                    <th className="pb-2 pr-4">Email</th>
                    <th className="pb-2 pr-4">Limit</th>
                    <th className="pb-2">Sessions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-700/50">
                  {sec.users_exceeding_device_limit.map((u) => (
                    <tr key={u.id} className="text-gray-300">
                      <td className="py-2 pr-4 text-white font-medium">{u.name}</td>
                      <td className="py-2 pr-4 text-gray-400">{u.email}</td>
                      <td className="py-2 pr-4">{u.device_limit}</td>
                      <td className="py-2 text-red-400 font-semibold">{u.session_count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {sec.users_exceeding_device_limit.length === 0 && (
          <div className="bg-green-500/5 border border-green-500/20 rounded-xl p-4 flex items-center gap-3">
            <UserCheck className="h-5 w-5 text-green-400" />
            <p className="text-green-400 text-sm">No users are exceeding their device limit.</p>
          </div>
        )}
      </Section>

      {/* Most active users table */}
      {up.most_active_users.length > 0 && (
        <Section title="Top 10 Most Active Users" icon={TrendingUp}>
          <div className="bg-[#1a0f2e] border border-gray-700 rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-[#2D1B4E] border-b border-gray-700">
                <tr className="text-left text-gray-400">
                  <th className="px-4 py-3">#</th>
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Email</th>
                  <th className="px-4 py-3 text-right">Searches</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700/50">
                {up.most_active_users.map((u, i) => (
                  <tr key={u.id} className="hover:bg-[#2D1B4E]/50 transition-colors">
                    <td className="px-4 py-3 text-gray-500">{i + 1}</td>
                    <td className="px-4 py-3 text-white font-medium">{u.name}</td>
                    <td className="px-4 py-3 text-gray-400">{u.email}</td>
                    <td className="px-4 py-3 text-right font-semibold text-pink-400">{fmt(u.search_count)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Section>
      )}
    </div>
  );
}
