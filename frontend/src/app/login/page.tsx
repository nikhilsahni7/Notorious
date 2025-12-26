/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";

import { Button } from "@/components/ui/button";
import { Captcha } from "@/components/ui/captcha";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { useAuth } from "@/contexts/AuthContext";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [captchaValid, setCaptchaValid] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);
  const { login } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [deviceLimitData, setDeviceLimitData] = useState<{
    limit: number;
    active_devices: any[];
  } | null>(null);

  // Check if redirected due to session expiry
  useEffect(() => {
    if (searchParams.get("session") === "expired") {
      setSessionExpired(true);
    }
  }, [searchParams]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setDeviceLimitData(null);

    if (!captchaValid) {
      setError("Please complete the verification code");
      return;
    }

    setLoading(true);

    try {
      await login(email, password);
      router.push("/search");
    } catch (err: any) {
      // Check if it's a device limit error
      // ApiError has status and data properties
      if (err.status === 409 && err.data?.error === "device_limit_exceeded") {
        setDeviceLimitData(err.data);
      } else {
        setError(err instanceof Error ? err.message : "Login failed");
      }
    } finally {
      setLoading(false);
    }
  };

  const handleRevoke = async (sessionId: string) => {
    // No confirmation needed for smoother UX, or maybe just a quick one?
    // User said "they can logout from other device and then it logins automatically"
    // Let's keep confirmation to avoid accidental clicks, but make it auto-login after.
    if (!confirm("Are you sure you want to logout this device?")) return;

    try {
      setLoading(true);
      const { authService } = await import("@/services/auth.service");
      await authService.revokeSession({
        email,
        password,
        session_id: sessionId,
      });

      // Auto-login after revocation
      await login(email, password);
      router.push("/search");
    } catch (err: any) {
      alert(err.message || "Failed to revoke session");
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-[#1a0f2e] via-[#2D1B4E] to-[#1a0f2e] flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">Welcome Back</h1>
          <p className="text-gray-400">
            Sign in to access your search dashboard
          </p>
        </div>

        <div className="bg-[#1a0f2e] rounded-lg border border-gray-700 p-8 shadow-xl relative">
          {/* Session Expired Warning */}
          {sessionExpired && !deviceLimitData && (
            <div className="bg-yellow-500/10 border border-yellow-500 text-yellow-400 p-4 rounded-lg mb-6">
              <h3 className="font-bold text-lg mb-1">Session Expired</h3>
              <p className="text-sm">
                Your session has expired. Please sign in again to continue.
              </p>
            </div>
          )}

          {deviceLimitData ? (
            <div className="space-y-4">
              <div className="bg-red-500/10 border border-red-500 text-red-400 p-4 rounded-lg">
                <h3 className="font-bold text-lg mb-1">Device Limit Reached</h3>
                <p className="text-sm">
                  You have reached your device limit of {deviceLimitData.limit}.
                  Please logout from one of your active devices to continue.
                </p>
              </div>

              <div className="space-y-3">
                <h4 className="text-sm font-medium text-gray-300">
                  Active Devices:
                </h4>
                {deviceLimitData.active_devices.map((device: any) => (
                  <div
                    key={device.id}
                    className="bg-[#2D1B4E] p-3 rounded border border-gray-600 flex justify-between items-center"
                  >
                    <div>
                      <div className="text-white text-sm font-medium">
                        {device.device_name}
                      </div>
                      <div className="text-xs text-gray-400">
                        {device.device_os} •{" "}
                        {device.location || "Unknown Location"}
                      </div>
                      <div className="text-xs text-gray-500 mt-0.5">
                        Last active:{" "}
                        {new Date(device.last_active).toLocaleDateString()}
                      </div>
                    </div>
                    <Button
                      onClick={() => handleRevoke(device.id)}
                      variant="destructive"
                      size="sm"
                      className="h-8 text-xs"
                      disabled={loading}
                    >
                      Logout
                    </Button>
                  </div>
                ))}
              </div>

              <Button
                onClick={() => setDeviceLimitData(null)}
                variant="outline"
                className="w-full mt-4 bg-transparent border-gray-600 text-white hover:bg-gray-700"
              >
                Cancel
              </Button>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-5">
              {error && (
                <div className="bg-red-500/10 border border-red-500 text-red-400 p-3 rounded-lg text-sm">
                  {error}
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Email
                </label>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="bg-[#2D1B4E] border-gray-600 text-white"
                  placeholder="your@email.com"
                  required
                  disabled={loading}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Password
                </label>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="bg-[#2D1B4E] border-gray-600 text-white"
                  placeholder="••••••••"
                  required
                  disabled={loading}
                />
              </div>

              <Captcha onVerify={setCaptchaValid} />

              <Button
                type="submit"
                className="w-full bg-pink-500 hover:bg-pink-600 text-white h-11"
                disabled={loading || !captchaValid}
              >
                {loading ? (
                  <>
                    <Spinner size="sm" className="mr-2" />
                    Signing in...
                  </>
                ) : (
                  "Sign In"
                )}
              </Button>
            </form>
          )}

          {!deviceLimitData && (
            <div className="mt-6 pt-6 border-t border-gray-700 space-y-3">
              <p className="text-center text-sm text-gray-400">
                Don&apos;t have an account?{" "}
                <Link
                  href="/request-access"
                  className="text-pink-400 hover:text-pink-300 font-medium"
                >
                  Request Access
                </Link>
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
