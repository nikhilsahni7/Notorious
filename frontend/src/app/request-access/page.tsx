"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { useRouter } from "next/navigation";
import { useState } from "react";

export default function RequestAccessPage() {
  const [formData, setFormData] = useState({
    email: "",
    name: "",
    phone: "",
    requested_searches_per_day: "100",
  });
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const { authService } = await import("@/services/auth.service");

      await authService.requestAccess({
        ...formData,
        requested_searches_per_day: parseInt(formData.requested_searches_per_day),
      });

      setSuccess(true);
      setTimeout(() => router.push("/login"), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="min-h-screen bg-[#0a0515] flex items-center justify-center p-6 relative overflow-hidden">
        {/* Holi Background Elements */}
        <div className="absolute top-[-10%] left-[-10%] w-[50%] h-[50%] bg-pink-600/10 blur-[120px] rounded-full" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] bg-blue-600/10 blur-[120px] rounded-full" />

        <div className="w-full max-w-md relative z-10">
          <div className="bg-green-500/10 border border-green-500 p-8 rounded-lg text-center">
            <h2 className="text-2xl font-bold text-green-400 mb-4">Request Submitted!</h2>
            <p className="text-gray-300 mb-4">
              Your access request has been submitted successfully. An administrator will review your request and contact you via email.
            </p>
            <p className="text-gray-400 text-sm">
              Redirecting to login page...
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-black flex items-center justify-center p-6 relative overflow-hidden"
      style={{
        backgroundImage: 'url("/holi-auth.png")',
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        backgroundAttachment: 'fixed'
      }}
    >
      {/* Holi Background Elements */}
      <div className="absolute top-[-10%] left-[-10%] w-[50%] h-[50%] bg-pink-600/5 blur-[120px] rounded-full" />
      <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] bg-blue-600/5 blur-[120px] rounded-full" />

      <div className="w-full max-w-md relative z-10">
        <div className="bg-white/5 backdrop-blur-2xl p-8 rounded-[2.5rem] border border-white/20 shadow-2xl relative overflow-hidden">
          {/* Internal Festive Glow */}
          <div className="absolute -top-10 -right-10 w-32 h-32 bg-yellow-500/20 blur-[50px] pointer-events-none" />

          <h1 className="text-3xl font-black text-white mb-2 text-center tracking-tight">
            Request Access
          </h1>
          <p className="text-white/60 text-center mb-8 font-medium">
            Fill out the form to request database access
          </p>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Full Name *
              </label>
              <Input
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="John Doe"
                className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-400"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Email *
              </label>
              <Input
                type="email"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                placeholder="john@example.com"
                className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-400"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Phone Number *
              </label>
              <Input
                type="tel"
                value={formData.phone}
                onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                placeholder="+91 98765 43210"
                className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-400"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Requested Searches Per Day *
              </label>
              <Input
                type="number"
                value={formData.requested_searches_per_day}
                onChange={(e) => setFormData({ ...formData, requested_searches_per_day: e.target.value })}
                min="1"
                max="10000"
                className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-400"
                required
              />
              <p className="text-gray-500 text-xs mt-1">
                How many searches do you need per day? (1-10,000)
              </p>
            </div>

            {error && (
              <div className="bg-red-500/10 border border-red-500 text-red-400 p-3 rounded text-sm">
                {error}
              </div>
            )}

            <Button
              type="submit"
              disabled={loading}
              className="w-full bg-gradient-to-r from-pink-500 to-blue-500 hover:from-pink-600 hover:to-blue-600 text-white h-11 border-none shadow-[0_0_15px_rgba(236,72,153,0.3)] transition-all duration-300 font-bold"
            >
              {loading ? (
                <>
                  <Spinner size="sm" className="mr-2" />
                  Submitting...
                </>
              ) : (
                "Submit Request"
              )}
            </Button>
          </form>

          <div className="mt-8 text-center pt-6 border-t border-white/10">
            <p className="text-white/60 text-sm font-medium">
              Already have an account?{" "}
              <a href="/login" className="text-yellow-400 hover:text-yellow-300 font-bold transition-colors">
                Sign in
              </a>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
