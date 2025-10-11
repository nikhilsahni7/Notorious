"use client";

import { useState, useEffect } from "react";
import { RefreshCw } from "lucide-react";

interface CaptchaProps {
  onVerify: (isValid: boolean) => void;
}

export function Captcha({ onVerify }: CaptchaProps) {
  const [captchaText, setCaptchaText] = useState("");
  const [userInput, setUserInput] = useState("");

  const generateCaptcha = () => {
    const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789";
    let result = "";
    for (let i = 0; i < 6; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    setCaptchaText(result);
    setUserInput("");
    onVerify(false);
  };

  useEffect(() => {
    generateCaptcha();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (userInput.length === 6) {
      onVerify(userInput === captchaText);
    } else {
      onVerify(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userInput, captchaText]);

  return (
    <div className="space-y-2">
      <label className="block text-sm font-medium text-gray-300">
        Verification Code
      </label>
      <div className="flex items-center gap-2">
        <div className="flex-1 bg-[#2D1B4E] border border-gray-600 rounded-lg p-3 flex items-center justify-center select-none">
          <div
            className="text-2xl font-bold tracking-wider"
            style={{
              background: "linear-gradient(45deg, #ec4899, #8b5cf6, #06b6d4)",
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
              fontFamily: "monospace",
              letterSpacing: "0.3em",
              textShadow: "2px 2px 4px rgba(0,0,0,0.3)",
            }}
          >
            {captchaText}
          </div>
        </div>
        <button
          type="button"
          onClick={generateCaptcha}
          className="p-3 bg-[#2D1B4E] border border-gray-600 rounded-lg hover:bg-[#1a0f2e] transition-colors"
          title="Generate new code"
        >
          <RefreshCw className="w-5 h-5 text-gray-400" />
        </button>
      </div>
      <input
        type="text"
        value={userInput}
        onChange={(e) => setUserInput(e.target.value.slice(0, 6))}
        placeholder="Enter the code above"
        className="w-full px-4 py-2 bg-[#2D1B4E] border border-gray-600 rounded-lg text-white placeholder:text-gray-500 focus:border-pink-500 focus:outline-none focus:ring-1 focus:ring-pink-500"
        maxLength={6}
      />
      {userInput.length === 6 && userInput !== captchaText && (
        <p className="text-xs text-red-400">Incorrect verification code</p>
      )}
      {userInput === captchaText && userInput.length === 6 && (
        <p className="text-xs text-green-400">✓ Verification successful</p>
      )}
    </div>
  );
}

