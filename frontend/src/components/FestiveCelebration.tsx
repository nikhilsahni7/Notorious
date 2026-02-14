"use client";

import confetti from "canvas-confetti";
import { motion } from "framer-motion";
import { Heart } from "lucide-react";
import React, { useEffect } from "react";

export const triggerFestiveBlast = () => {
  const duration = 5 * 1000;
  const animationEnd = Date.now() + duration;
  const defaults = { startVelocity: 30, spread: 360, ticks: 60, zIndex: 0 };

  const randomInRange = (min: number, max: number) =>
    Math.random() * (max - min) + min;

  const interval: any = setInterval(function () {
    const timeLeft = animationEnd - Date.now();

    if (timeLeft <= 0) {
      return clearInterval(interval);
    }

    const particleCount = 50 * (timeLeft / duration);
    // Valentine's Day — reds, pinks, whites
    confetti({
      ...defaults,
      particleCount,
      origin: { x: randomInRange(0.1, 0.3), y: Math.random() - 0.2 },
      colors: ["#E11D48", "#FB7185", "#FFFFFF", "#FDA4AF"],
    });
    confetti({
      ...defaults,
      particleCount,
      origin: { x: randomInRange(0.7, 0.9), y: Math.random() - 0.2 },
      colors: ["#E11D48", "#FB7185", "#FFFFFF", "#FDA4AF"],
    });
  }, 250);
};

export const triggerFestiveQuickBlast = () => {
  confetti({
    particleCount: 150,
    spread: 70,
    origin: { y: 0.6 },
    colors: ["#E11D48", "#FB7185", "#FFFFFF", "#FDA4AF", "#FF1493"],
    ticks: 200,
    gravity: 1.2,
    scalar: 1.2,
    shapes: ["star", "circle"],
  });
};

const FloatingElement = ({ children, delay = 0, duration = 4 }: { children: React.ReactNode, delay?: number, duration?: number }) => (
  <motion.div
    initial={{ y: 0, opacity: 0 }}
    animate={{
      y: [0, -20, 0],
      opacity: [0.4, 0.8, 0.4],
      rotate: [0, 10, -10, 0]
    }}
    transition={{
      duration,
      repeat: Infinity,
      delay,
      ease: "easeInOut"
    }}
  >
    {children}
  </motion.div>
);

export const FestiveHeader = () => {
  return (
    <div className="hidden lg:flex items-center px-8 py-2.5 bg-gradient-to-r from-rose-600/20 via-pink-400/10 to-red-600/20 rounded-2xl border border-white/10 mx-6 flex-1 justify-center max-w-lg shadow-[0_0_30px_rgba(225,29,72,0.25)] relative overflow-hidden group">
      {/* Animated background glow - Valentine themed */}
      <motion.div
        animate={{
          opacity: [0.1, 0.2, 0.1],
          scale: [1, 1.1, 1],
        }}
        transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
        className="absolute inset-0 bg-gradient-to-r from-rose-500/10 via-pink-300/5 to-red-500/10 pointer-events-none"
      />

      <div className="flex items-center gap-6 relative z-10">
        <motion.div
          animate={{ rotate: [-10, 10, -10], scale: [1, 1.2, 1] }}
          transition={{ duration: 3, repeat: Infinity, ease: "easeInOut" }}
        >
          <Heart size={24} fill="#FB7185" className="text-rose-400 drop-shadow-[0_0_10px_rgba(251,113,133,0.8)]" />
        </motion.div>

        <div className="flex flex-col items-center">
          <motion.div
            initial={{ y: -5, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            className="flex items-center gap-3"
          >
            <span className="text-base font-black text-white tracking-[0.3em] uppercase drop-shadow-md">
              Happy Valentine's Day
            </span>
          </motion.div>

          <div className="h-[1.5px] w-full bg-gradient-to-r from-rose-500 via-pink-300 to-red-500 my-1.5 opacity-60" />

          <motion.span
            animate={{ opacity: [0.6, 1, 0.6] }}
            transition={{ duration: 2, repeat: Infinity }}
            className="text-[10px] text-white/80 font-black uppercase tracking-[0.4em]"
          >
            Spread Love & Joy 💕
          </motion.span>
        </div>

        <motion.div
          animate={{ rotate: [10, -10, 10], scale: [1, 1.2, 1] }}
          transition={{ duration: 3, repeat: Infinity, ease: "easeInOut" }}
        >
          <Heart size={24} fill="#FB7185" className="text-rose-400 drop-shadow-[0_0_10px_rgba(251,113,133,0.8)]" />
        </motion.div>
      </div>

      {/* Shimmer effect on hover */}
      <div className="absolute inset-0 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000 bg-gradient-to-r from-transparent via-pink-200/20 to-transparent" />
    </div>
  );
};

export const FestiveCelebration = () => {
  useEffect(() => {
    // Auto blast on mount for festive feel
    const blastTimer = setTimeout(() => {
        triggerFestiveQuickBlast();
    }, 1500);

    return () => clearTimeout(blastTimer);
  }, []);

  return null; // Side effects only
};
