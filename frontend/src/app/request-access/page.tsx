"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";

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
      <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center p-6">
        <div className="w-full max-w-md">
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
    <div className="min-h-screen bg-[#2D1B4E] flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="bg-[#1a0f2e] p-8 rounded-lg border border-gray-700">
          <h1 className="text-3xl font-bold text-white mb-2 text-center">
            Request Access
          </h1>
          <p className="text-gray-400 text-center mb-6">
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
              className="w-full bg-pink-500 hover:bg-pink-600 text-white"
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

          <div className="mt-6 text-center">
            <p className="text-gray-400 text-sm">
              Already have an account?{" "}
              <a href="/login" className="text-pink-500 hover:text-pink-400">
                Sign in
              </a>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

