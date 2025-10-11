"use client";

import { AuthProvider as Auth } from "@/contexts/AuthContext";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  return <Auth>{children}</Auth>;
}

