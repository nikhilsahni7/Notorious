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
      const [users, requests, history] = await Promise.all([
        adminService.listUsers(token, { limit: 100 }).catch(() => []),
        adminService.listUserRequests(token, { status: "pending", limit: 100 }).catch(() => []),
        adminService.getSearchHistory(token, { limit: 1000 }).catch(() => []),
      ]);

      setStats({
        totalUsers: users?.length || 0,
        activeUsers: users?.filter((u) => u.is_active).length || 0,
        pendingRequests: requests?.length || 0,
        totalSearches: history?.length || 0,
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
    <div>
      <h2 className="text-xl font-bold text-white mb-4">Overview</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map((stat) => {
          const Icon = stat.icon;
          return (
            <div
              key={stat.label}
              className="bg-[#2D1B4E] p-6 rounded-lg border border-gray-700"
            >
              <div className="flex items-center justify-between mb-2">
                <p className="text-gray-400 text-sm">{stat.label}</p>
                <div className={`${stat.bgColor} p-2 rounded`}>
                  <Icon className={`h-5 w-5 ${stat.color}`} />
                </div>
              </div>
              <p className="text-3xl font-bold text-white">{stat.value}</p>
            </div>
          );
        })}
      </div>

      <div className="mt-6 bg-[#2D1B4E] p-6 rounded-lg border border-gray-700">
        <h3 className="text-lg font-semibold text-white mb-4">Quick Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
          <div className="text-gray-300">
            <p className="font-medium text-white mb-1">User Management</p>
            <p>View and manage all users, set search limits, activate/deactivate accounts</p>
          </div>
          <div className="text-gray-300">
            <p className="font-medium text-white mb-1">Access Requests</p>
            <p>Approve or reject user access requests, set initial search limits</p>
          </div>
          <div className="text-gray-300">
            <p className="font-medium text-white mb-1">Search History</p>
            <p>Monitor all searches, view user activity, track usage patterns</p>
          </div>
        </div>
      </div>
    </div>
  );
}

