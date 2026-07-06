/* eslint-disable @typescript-eslint/no-explicit-any */
"use client";

import { Button } from "@/components/ui/button";
import { Captcha } from "@/components/ui/captcha";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { useAuth } from "@/contexts/AuthContext";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";

// Wrap the main content in a separate component that uses useSearchParams
function LoginContent() {
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
    <div className="min-h-screen bg-[#05070d] p-3 md:p-5 overflow-hidden">
      <div
        className="relative h-[calc(100vh-1.5rem)] md:h-[calc(100vh-2.5rem)] rounded-2xl border border-cyan-300/25 shadow-[0_0_40px_rgba(34,211,238,0.18)] overflow-hidden"
        style={{
          backgroundImage: 'url("/third_eye_login_hero.png")',
          backgroundSize: "112% auto",
          backgroundPosition: "34% center",
          backgroundRepeat: "no-repeat",
          backgroundColor: "#070d1d",
        }}
      >
        <div className="absolute inset-0 bg-gradient-to-r from-[#080e1f]/40 via-[#070c1a]/20 to-[#050812]/78 pointer-events-none" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_22%_30%,rgba(56,189,248,0.18),transparent_45%),radial-gradient(circle_at_8%_10%,rgba(236,72,153,0.12),transparent_30%)] pointer-events-none" />

        <div className="absolute left-[6%] md:left-[10%] top-[67%] md:top-[69%] -translate-y-1/2 z-10 w-[min(88vw,38rem)] md:w-[min(46vw,40rem)] text-center md:text-left">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-amber-500/10 border border-amber-500/25 backdrop-blur-md mb-4 shadow-[0_0_15px_rgba(245,158,11,0.1)]">
            <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
            <span className="text-amber-400 font-bold uppercase tracking-wider text-[0.75rem]">
              System Notice
            </span>
          </div>
          <p className="text-white font-black leading-[1.25] text-[clamp(1.2rem,2.15vw,2.15rem)] drop-shadow-[0_6px_18px_rgba(0,0,0,0.9)]">
            Disclaimer: for Maharashtra and Rajasthan data
            <br />
            upcoming with live tracking process
          </p>
        </div>

        <div className="w-full h-full flex items-center justify-end px-4 md:px-10">
          <div className="w-full max-w-md relative z-10 animate-in fade-in slide-in-from-right-10 duration-1000">
        <div className="text-center mb-10 relative">
          <h1 className="text-4xl font-black text-white mb-2 tracking-tighter drop-shadow-[0_4px_12px_rgba(0,0,0,0.9)]">
            Welcome
          </h1>
          <p className="text-white/60 font-medium text-sm tracking-wide">
            Sign in to access your search dashboard
          </p>
        </div>

        <div className="bg-white/5 backdrop-blur-2xl rounded-[2.5rem] border border-white/20 p-10 shadow-[0_25px_50px_-12px_rgba(0,0,0,0.5)] relative overflow-hidden">
          {/* Internal festive glows - Navratri themed */}
          <div className="absolute -top-10 -right-10 w-40 h-40 bg-yellow-500/20 blur-[60px] pointer-events-none" />
          <div className="absolute -bottom-10 -left-10 w-40 h-40 bg-blue-500/20 blur-[60px] pointer-events-none" />
          <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-60 h-60 bg-pink-500/10 blur-[80px] pointer-events-none" />
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
                className="w-full bg-gradient-to-r from-pink-500 to-blue-500 hover:from-pink-600 hover:to-blue-600 text-white h-11 border-none shadow-[0_0_15px_rgba(236,72,153,0.3)] transition-all duration-300"
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
                  className="text-yellow-400 hover:text-yellow-300 font-medium transition-colors"
                >
                  Request Access
                </Link>
              </p>
            </div>
          )}
        </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// Default export with Suspense boundary for useSearchParams
export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <div
          className="min-h-screen relative flex items-center justify-center bg-[#0a0515]"
          style={{
            background:
              "radial-gradient(circle at center, #1a0f2e 0%, #0a0515 100%)",
          }}
        >
          <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-80 h-80 bg-pink-500/20 blur-[100px]" />
          <div className="absolute inset-0 bg-[#0a0515]/60" />
          <Spinner size="lg" className="relative z-10" />
        </div>
      }
    >
      <LoginContent />
    </Suspense>
  );
}
