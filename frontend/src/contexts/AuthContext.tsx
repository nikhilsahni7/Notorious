"use client";

import { API_CONFIG } from "@/config/api";
import { authService } from "@/services/auth.service";
import { createContext, useCallback, useContext, useEffect, useRef, useState } from "react";

interface User {
  id: string;
  email: string;
  name: string;
  phone?: string;
  role: string;
  daily_search_limit: number;
  searches_used_today: number;
  is_active: boolean;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  isLoading: boolean;
  updateUser: (user: User) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const TOKEN_KEY = "knotorious_token";
const USER_KEY = "knotorious_user";
const HEARTBEAT_INTERVAL = 30000; // 30 seconds

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const heartbeatRef = useRef<NodeJS.Timeout | null>(null);

  // Heartbeat function to update presence
  const sendHeartbeat = useCallback(async () => {
    const storedToken = localStorage.getItem(TOKEN_KEY);
    if (!storedToken) return;

    try {
      await fetch(`${API_CONFIG.BASE_URL}${API_CONFIG.ENDPOINTS.USER.HEARTBEAT}`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${storedToken}`,
          "Content-Type": "application/json",
        },
      });
    } catch (error) {
      // Silently fail - heartbeat should not block user experience
      console.debug("Heartbeat failed:", error);
    }
  }, []);

  // Setup heartbeat interval
  useEffect(() => {
    const storedToken = localStorage.getItem(TOKEN_KEY);
    const storedUser = localStorage.getItem(USER_KEY);

    if (storedToken && storedUser) {
      try {
        setToken(storedToken);
        setUser(JSON.parse(storedUser));

        // Send immediate heartbeat on load
        sendHeartbeat();

        // Setup heartbeat interval
        heartbeatRef.current = setInterval(sendHeartbeat, HEARTBEAT_INTERVAL);
      } catch (error) {
        console.error("Failed to parse stored user data", error);
        localStorage.removeItem(TOKEN_KEY);
        localStorage.removeItem(USER_KEY);
      }
    }
    setIsLoading(false);

    // Handle visibility changes - pause heartbeat when tab is hidden
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        // Tab became visible - send heartbeat immediately and restart interval
        sendHeartbeat();
        if (!heartbeatRef.current && localStorage.getItem(TOKEN_KEY)) {
          heartbeatRef.current = setInterval(sendHeartbeat, HEARTBEAT_INTERVAL);
        }
      } else {
        // Tab hidden - stop heartbeat to save resources
        if (heartbeatRef.current) {
          clearInterval(heartbeatRef.current);
          heartbeatRef.current = null;
        }
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      if (heartbeatRef.current) {
        clearInterval(heartbeatRef.current);
      }
    };
  }, [sendHeartbeat]);

  const login = async (email: string, password: string) => {
    const data = await authService.login({ email, password });

    setToken(data.token);
    setUser(data.user);

    localStorage.setItem(TOKEN_KEY, data.token);
    localStorage.setItem(USER_KEY, JSON.stringify(data.user));

    // Start heartbeat after login
    sendHeartbeat();
    if (heartbeatRef.current) {
      clearInterval(heartbeatRef.current);
    }
    heartbeatRef.current = setInterval(sendHeartbeat, HEARTBEAT_INTERVAL);
  };

  const logout = async () => {
    // Stop heartbeat on logout
    if (heartbeatRef.current) {
      clearInterval(heartbeatRef.current);
      heartbeatRef.current = null;
    }

    if (token) {
      try {
        await authService.logout(token);
      } catch (error) {
        console.error("Failed to logout from server", error);
      }
    }
    setToken(null);
    setUser(null);
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
  };

  const updateUser = (updatedUser: User) => {
    setUser(updatedUser);
    localStorage.setItem(USER_KEY, JSON.stringify(updatedUser));
  };

  return (
    <AuthContext.Provider
      value={{ user, token, login, logout, isLoading, updateUser }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
