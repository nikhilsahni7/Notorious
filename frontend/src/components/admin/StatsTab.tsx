import { useEffect, useState } from "react";
import { adminService } from "@/services/admin.service";
import { Spinner } from "@/components/ui/spinner";
import { Users, Search, Clock, UserCheck } from "lucide-react";

interface StatsTabProps {
  token: string;
}

export function StatsTab({ token }: StatsTabProps) {
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    totalUsers: 0,
    activeUsers: 0,
    pendingRequests: 0,
    totalSearches: 0,
  });

  useEffect(() => {
    loadStats();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
      <div className="flex items-center justify-center py-12">
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
      bgColor: "bg-blue-500/10",
    },
    {
      label: "Active Users",
      value: stats.activeUsers,
      icon: UserCheck,
      color: "text-green-400",
      bgColor: "bg-green-500/10",
    },
    {
      label: "Pending Requests",
      value: stats.pendingRequests,
      icon: Clock,
      color: "text-yellow-400",
      bgColor: "bg-yellow-500/10",
    },
    {
      label: "Total Searches",
      value: stats.totalSearches,
      icon: Search,
      color: "text-pink-400",
      bgColor: "bg-pink-500/10",
    },
  ];

  return (
    <div className="space-y-8 animate-in fade-in zoom-in-95 duration-500">
      <div>
        <h2 className="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-white to-gray-400 mb-6 drop-shadow-sm">
          Overview
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {statCards.map((stat, i) => {
            const Icon = stat.icon;
            return (
              <div
                key={stat.label}
                className="group relative overflow-hidden bg-white/5 backdrop-blur-xl p-6 rounded-2xl border border-white/10 shadow-xl hover:shadow-2xl hover:border-white/20 transition-all duration-300 transform hover:-translate-y-1"
                style={{ animationDelay: `${i * 100}ms` }}
              >
                {/* Background ambient glow */}
                <div className={`absolute -inset-4 opacity-0 group-hover:opacity-20 blur-2xl transition-opacity duration-500 ${stat.bgColor.replace("/10", "")}`} />
                
                <div className="relative z-10 flex items-center justify-between mb-4">
                  <p className="text-gray-400 text-sm font-medium tracking-wide uppercase">{stat.label}</p>
                  <div className={`${stat.bgColor} p-3 rounded-xl shadow-inner group-hover:scale-110 transition-transform duration-300`}>
                    <Icon className={`h-5 w-5 ${stat.color} filter drop-shadow-md`} />
                  </div>
                </div>
                <div className="relative z-10 flex items-baseline gap-2">
                  <p className="text-4xl font-extrabold text-white tracking-tight">
                    {stat.value.toLocaleString()}
                  </p>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="relative overflow-hidden bg-gradient-to-br from-[#2D1B4E] to-[#1A0B2E] p-8 rounded-2xl border border-white/10 shadow-2xl">
        <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-pink-500 to-transparent opacity-50" />
        <h3 className="text-xl font-bold text-white mb-6 flex items-center gap-2">
          <Clock className="w-5 h-5 text-gray-400" />
          Quick Actions
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="group cursor-pointer p-4 rounded-xl hover:bg-white/5 transition-colors duration-300">
            <p className="font-semibold text-white mb-2 group-hover:text-pink-400 transition-colors">User Management</p>
            <p className="text-gray-400 text-sm leading-relaxed">View and manage all users, set search limits, activate/deactivate accounts seamlessly.</p>
          </div>
          <div className="group cursor-pointer p-4 rounded-xl hover:bg-white/5 transition-colors duration-300">
            <p className="font-semibold text-white mb-2 group-hover:text-yellow-400 transition-colors">Access Requests</p>
            <p className="text-gray-400 text-sm leading-relaxed">Approve or reject user access requests instantly, and set primary search limits.</p>
          </div>
          <div className="group cursor-pointer p-4 rounded-xl hover:bg-white/5 transition-colors duration-300">
            <p className="font-semibold text-white mb-2 group-hover:text-blue-400 transition-colors">Search History</p>
            <p className="text-gray-400 text-sm leading-relaxed">Monitor all ongoing searches, examine user activity, and evaluate platform usage patterns.</p>
          </div>
        </div>
      </div>
    </div>
  );
}

