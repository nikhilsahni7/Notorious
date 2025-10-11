"use client";

import { Spinner } from "@/components/ui/spinner";
import { useAuth } from "@/contexts/AuthContext";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function Home() {
  const router = useRouter();
  const { token, user, isLoading } = useAuth();

  useEffect(() => {
    if (!isLoading) {
      if (!token) {
        router.push("/login");
      } else if (user?.role === "admin") {
        router.push("/admin");
      } else if (user?.role === "user") {
        router.push("/search");
      }
    }
  }, [isLoading, token, user, router]);

  return (
    <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center">
      <Spinner size="lg" />
    </div>
  );
}
