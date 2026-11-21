"use client";

import { KnotoriousBrand } from "@/components/KnotoriousBrand";
import { ChatNotificationBadge } from "@/components/chat/ChatNotificationBadge";

import { PasswordRequestsTab } from "@/components/admin/PasswordRequestsTab";
import { SearchHistoryTab } from "@/components/admin/SearchHistoryTab";
import { SessionsTab } from "@/components/admin/SessionsTab";
import { StatsTab } from "@/components/admin/StatsTab";
import { SystemStatsTab } from "@/components/admin/SystemStatsTab";
import { UserRequestsTab } from "@/components/admin/UserRequestsTab";
import { UsersTab } from "@/components/admin/UsersTab";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useAuth } from "@/contexts/AuthContext";
import { adminService, DashboardStats } from "@/services/admin.service";
import {
  Activity,
  BarChart3,
  History,
  Key,
  LogOut,
  MessageCircle,
  Shield,
  UserPlus,
  Users,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

type Tab =
  | "users"
  | "requests"
  | "history"
  | "stats"
  | "password-requests"
  | "sessions"
  | "system-stats"
  | "messages";

export default function AdminDashboard() {
  const { user, token, logout, isLoading } = useAuth();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<Tab>("stats");
  const [requestCounts, setRequestCounts] = useState<DashboardStats | null>(
    null,
  );

  useEffect(() => {
    if (!isLoading) {
      if (!token) {
        router.push("/login");
      } else if (user?.role !== "admin") {
        router.push("/search");
      } else {
        loadRequestCounts();
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, token, user, router]);

  const loadRequestCounts = async () => {
    if (!token) return;
    try {
      const counts = await adminService.getDashboardStats(token);
      setRequestCounts(counts);
    } catch (error) {
      console.error("Failed to load request counts:", error);
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!token || user?.role !== "admin") {
    return null;
  }

  const tabs = [
    { id: "stats" as Tab, label: "Dashboard", icon: BarChart3 },
    { id: "users" as Tab, label: "Users", icon: Users },
    {
      id: "requests" as Tab,
      label: "Access Requests",
      icon: UserPlus,
      badge: requestCounts?.pending_user_requests,
    },
    {
      id: "password-requests" as Tab,
      label: "Password Requests",
      icon: Key,
      badge: requestCounts?.pending_password_requests,
    },
    { id: "sessions" as Tab, label: "Sessions", icon: Shield },
    { id: "history" as Tab, label: "Search History", icon: History },
    { id: "system-stats" as Tab, label: "System Stats", icon: Activity },
    { id: "messages" as Tab, label: "Messages", icon: MessageCircle },
  ];

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-3 md:p-6">
      <div className="max-w-[1800px] mx-auto">
        <div className="flex flex-col md:flex-row md:justify-between md:items-center gap-4 mb-6 md:mb-8">
          <div>
            <KnotoriousBrand subtitle="Administrator portal" size="sm" className="mb-4" />
            <h1 className="text-2xl md:text-3xl font-bold text-white tracking-tight">
              Admin Dashboard
            </h1>
            <p className="text-gray-400 text-sm mt-1">
              Manage users, requests, and view analytics for KNotorious
            </p>
          </div>

          <div className="flex items-center gap-3">
            <ChatNotificationBadge href="/admin/chat" size="md" />
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              className="bg-white/5 border-white/10 text-white hover:bg-white/10 hover:border-white/20 transition-all rounded-full px-6"
            >
              Go to Search
            </Button>
            <Button
              onClick={logout}
              variant="outline"
              className="bg-white/5 border-white/10 text-pink-400 hover:bg-pink-500/10 hover:border-pink-500/30 hover:text-pink-300 transition-all rounded-full px-6"
            >
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        </div>

        <div className="bg-[#1a0f2e] rounded-2xl border border-white/10 mb-8 shadow-xl overflow-hidden">
          <div className="flex overflow-x-auto hide-scrollbar border-b border-white/5 bg-black/20 p-2 gap-2">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-2 px-5 py-2.5 rounded-xl font-medium transition-all duration-200 whitespace-nowrap ${
                    isActive
                      ? "bg-pink-500/10 text-pink-400 border border-pink-500/20 shadow-[0_0_15px_rgba(236,72,153,0.1)]"
                      : "text-gray-400 hover:text-white hover:bg-white/5 border border-transparent"
                  }`}
                >
                  <Icon
                    className={`h-4 w-4 ${isActive ? "text-pink-400" : "opacity-70"}`}
                  />
                  {tab.label}
                  {typeof tab.badge === "number" && tab.badge > 0 && (
                    <span
                      className={`ml-1.5 px-2 py-0.5 text-[10px] uppercase tracking-wider font-bold rounded-full ${
                        isActive
                          ? "bg-pink-500/20 text-pink-300"
                          : "bg-red-500/20 text-red-400"
                      }`}
                    >
                      {tab.badge}
                    </span>
                  )}
                </button>
              );
            })}
          </div>

          <div className="p-3 md:p-6">
            {activeTab === "stats" && (
              <StatsTab
                token={token!}
                onNavigate={(tab) => setActiveTab(tab as Tab)}
                initialStats={requestCounts}
              />
            )}
            {activeTab === "users" && <UsersTab token={token!} />}
            {activeTab === "requests" && (
              <UserRequestsTab token={token!} onApprove={loadRequestCounts} />
            )}
            {activeTab === "password-requests" && (
              <PasswordRequestsTab token={token!} />
            )}
            {activeTab === "sessions" && <SessionsTab token={token!} />}
            {activeTab === "history" && <SearchHistoryTab token={token!} />}
            {activeTab === "system-stats" && <SystemStatsTab token={token!} />}
            {activeTab === "messages" && (
              <div className="flex flex-col items-center justify-center py-12">
                <MessageCircle size={48} className="text-gray-500 mb-4" />
                <h3 className="text-lg font-semibold text-white mb-2">Chat Messages</h3>
                <p className="text-gray-400 text-sm mb-4">Open the full chat interface to manage conversations</p>
                <Button
                  onClick={() => router.push("/admin/chat")}
                  className="bg-gradient-to-r from-indigo-500 to-purple-600 hover:from-indigo-600 hover:to-purple-700"
                >
                  <MessageCircle className="h-4 w-4 mr-2" />
                  Open Chat
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
