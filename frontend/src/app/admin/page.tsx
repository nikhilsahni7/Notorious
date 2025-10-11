"use client";

import { SearchHistoryTab } from "@/components/admin/SearchHistoryTab";
import { StatsTab } from "@/components/admin/StatsTab";
import { UserRequestsTab } from "@/components/admin/UserRequestsTab";
import { UsersTab } from "@/components/admin/UsersTab";
import { PasswordRequestsTab } from "@/components/admin/PasswordRequestsTab";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useAuth } from "@/contexts/AuthContext";
import { BarChart3, History, LogOut, UserPlus, Users, Key } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

type Tab = "users" | "requests" | "history" | "stats" | "password-requests";

export default function AdminDashboard() {
  const { user, token, logout, isLoading } = useAuth();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<Tab>("stats");

  useEffect(() => {
    if (!isLoading) {
      if (!token) {
        router.push("/login");
      } else if (user?.role !== "admin") {
        router.push("/search");
      }
    }
  }, [isLoading, token, user, router]);

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
    { id: "requests" as Tab, label: "Access Requests", icon: UserPlus },
    { id: "password-requests" as Tab, label: "Password Requests", icon: Key },
    { id: "history" as Tab, label: "Search History", icon: History },
  ];

  return (
    <div className="min-h-screen bg-[#2D1B4E] p-6">
      <div className="max-w-[1800px] mx-auto">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-3xl font-bold text-white">Admin Dashboard</h1>
            <p className="text-gray-400 text-sm">
              Manage users, requests, and view analytics
            </p>
          </div>
          <div className="flex items-center gap-4">
            <Button
              onClick={() => router.push("/search")}
              variant="outline"
              className="bg-transparent border-gray-600 text-white hover:bg-[#1a0f2e]"
            >
              Go to Search
            </Button>
            <Button
              onClick={logout}
              variant="outline"
              className="bg-transparent border-gray-600 text-white hover:bg-[#1a0f2e]"
            >
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        </div>

        <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 mb-6">
          <div className="flex border-b border-gray-700">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-2 px-6 py-4 font-medium transition-colors ${
                    activeTab === tab.id
                      ? "text-pink-500 border-b-2 border-pink-500 bg-[#2D1B4E]"
                      : "text-gray-400 hover:text-white hover:bg-[#2D1B4E]/50"
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  {tab.label}
                </button>
              );
            })}
          </div>

          <div className="p-6">
            {activeTab === "stats" && <StatsTab token={token!} />}
            {activeTab === "users" && <UsersTab token={token!} />}
            {activeTab === "requests" && <UserRequestsTab token={token!} />}
            {activeTab === "password-requests" && <PasswordRequestsTab token={token!} />}
            {activeTab === "history" && <SearchHistoryTab token={token!} />}
          </div>
        </div>
      </div>
    </div>
  );
}
