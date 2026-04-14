import { Spinner } from "@/components/ui/spinner";
import { adminService, DashboardStats } from "@/services/admin.service";
import {
  Activity,
  Calendar,
  Clock,
  Key,
  Search,
  UserCheck,
  Users,
} from "lucide-react";
import { useEffect, useState } from "react";

interface StatsTabProps {
  token: string;
  onNavigate?: (tab: string) => void;
  initialStats?: DashboardStats | null;
}

export function StatsTab({ token, onNavigate, initialStats }: StatsTabProps) {
  const [loading, setLoading] = useState(!initialStats);
  const [stats, setStats] = useState({
    totalUsers: initialStats?.total_users || 0,
    activeUsers: initialStats?.active_users || 0,
    pendingRequests: initialStats?.pending_user_requests || 0,
    totalSearches: initialStats?.total_searches || 0,
  });

  useEffect(() => {
    if (!initialStats) {
      loadStats();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialStats]);

  const loadStats = async () => {
    try {
      const dbStats = await adminService.getDashboardStats(token);

      setStats({
        totalUsers: dbStats.total_users || 0,
        activeUsers: dbStats.active_users || 0,
        pendingRequests: dbStats.pending_user_requests || 0,
        totalSearches: dbStats.total_searches || 0,
      });
    } catch (error) {
      console.error("Failed to load stats:", error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Spinner size="lg" />
      </div>
    );
  }

  const statCards = [
    {
      label: "Total Users",
      value: stats.totalUsers,
      icon: Users,
      color: "text-blue-400",
      bgColor: "bg-blue-500/20",
      gradient: "from-blue-500/10 to-transparent",
    },
    {
      label: "Active Users",
      value: stats.activeUsers,
      icon: UserCheck,
      color: "text-emerald-400",
      bgColor: "bg-emerald-500/20",
      gradient: "from-emerald-500/10 to-transparent",
    },
    {
      label: "Pending Requests",
      value: stats.pendingRequests,
      icon: Clock,
      color: "text-amber-400",
      bgColor: "bg-amber-500/20",
      gradient: "from-amber-500/10 to-transparent",
    },
    {
      label: "Total Searches",
      value: stats.totalSearches,
      icon: Search,
      color: "text-pink-400",
      bgColor: "bg-pink-500/20",
      gradient: "from-pink-500/10 to-transparent",
    },
  ];

  return (
    <div className="space-y-10 animate-in fade-in slide-in-from-bottom-4 duration-200">
      <div>
        <h2 className="text-2xl font-bold text-white mb-6 tracking-tight flex items-center gap-2">
          <Activity className="h-6 w-6 text-pink-400" />
          Platform Overview
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {statCards.map((stat, i) => {
            const Icon = stat.icon;
            return (
              <div
                key={stat.label}
                className={`group relative overflow-hidden bg-[#1B1130] p-6 rounded-3xl border border-white/5 hover:border-white/10 transition-all duration-300 transform hover:-translate-y-1`}
                style={{ animationDelay: `${i * 30}ms` }}
              >
                {/* Background ambient gradient */}
                <div
                  className={`absolute inset-0 bg-gradient-to-br ${stat.gradient} opacity-50`}
                />

                <div className="relative z-10 flex items-center justify-between mb-6">
                  <div
                    className={`${stat.bgColor} p-3 rounded-2xl shadow-inner group-hover:scale-110 transition-transform duration-300`}
                  >
                    <Icon className={`h-6 w-6 ${stat.color}`} />
                  </div>
                </div>
                <div className="relative z-10">
                  <p className="text-4xl font-black text-white tracking-tight mb-1">
                    {stat.value.toLocaleString()}
                  </p>
                  <p className="text-gray-400 text-sm font-medium uppercase tracking-widest">
                    {stat.label}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="relative overflow-hidden bg-[#1B1130] p-8 rounded-3xl border border-white/5">
        <h3 className="text-lg font-bold text-white mb-8 tracking-tight flex items-center gap-2">
          <Calendar className="w-5 h-5 text-gray-400" />
          Quick Actions
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div
            onClick={() => onNavigate?.("users")}
            className="group cursor-pointer p-6 rounded-2xl bg-white/5 hover:bg-gradient-to-br hover:from-white/10 hover:to-transparent border border-white/5 transition-all duration-300"
          >
            <div className="flex items-center gap-3 mb-3">
              <div className="bg-white/10 p-2 rounded-xl group-hover:bg-blue-500/20 transition-colors">
                <Users className="h-5 w-5 text-gray-400 group-hover:text-blue-400 transition-colors" />
              </div>
              <p className="font-bold text-white group-hover:text-blue-400 transition-colors">
                User Management
              </p>
            </div>
            <p className="text-gray-400 text-sm leading-relaxed">
              View and manage all users, set search limits, and
              activate/deactivate accounts seamlessly.
            </p>
          </div>

          <div
            onClick={() => onNavigate?.("requests")}
            className="group cursor-pointer p-6 rounded-2xl bg-white/5 hover:bg-gradient-to-br hover:from-white/10 hover:to-transparent border border-white/5 transition-all duration-300"
          >
            <div className="flex items-center gap-3 mb-3">
              <div className="bg-white/10 p-2 rounded-xl group-hover:bg-amber-500/20 transition-colors">
                <Key className="h-5 w-5 text-gray-400 group-hover:text-amber-400 transition-colors" />
              </div>
              <p className="font-bold text-white group-hover:text-amber-400 transition-colors">
                Access Requests
              </p>
            </div>
            <p className="text-gray-400 text-sm leading-relaxed">
              Approve or reject user access requests instantly, and configure
              primary search limits.
            </p>
          </div>

          <div
            onClick={() => onNavigate?.("history")}
            className="group cursor-pointer p-6 rounded-2xl bg-white/5 hover:bg-gradient-to-br hover:from-white/10 hover:to-transparent border border-white/5 transition-all duration-300"
          >
            <div className="flex items-center gap-3 mb-3">
              <div className="bg-white/10 p-2 rounded-xl group-hover:bg-pink-500/20 transition-colors">
                <Search className="h-5 w-5 text-gray-400 group-hover:text-pink-400 transition-colors" />
              </div>
              <p className="font-bold text-white group-hover:text-pink-400 transition-colors">
                Search History
              </p>
            </div>
            <p className="text-gray-400 text-sm leading-relaxed">
              Monitor ongoing searches, examine user activity, and evaluate
              platform usage patterns.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
