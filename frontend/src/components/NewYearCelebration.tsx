"use client";

import confetti from "canvas-confetti";
import { motion } from "framer-motion";
import { Flag, Flame, Wind } from "lucide-react";
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
    // Use Tricolor + Bonfire Gold/Orange
    confetti({
      ...defaults,
      particleCount,
      origin: { x: randomInRange(0.1, 0.3), y: Math.random() - 0.2 },
      colors: ["#F97316", "#FFFFFF", "#22C55E", "#FDE047"],
    });
    confetti({
      ...defaults,
      particleCount,
      origin: { x: randomInRange(0.7, 0.9), y: Math.random() - 0.2 },
      colors: ["#F97316", "#FFFFFF", "#22C55E", "#FDE047"],
    });
  }, 250);
};

export const triggerFestiveQuickBlast = () => {
  confetti({
    particleCount: 150,
    spread: 70,
    origin: { y: 0.6 },
    colors: ["#F97316", "#FFFFFF", "#22C55E", "#FFD700", "#FF4500"],
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
    <div className="hidden lg:flex items-center px-8 py-2.5 bg-gradient-to-r from-orange-600/20 via-white/10 to-green-600/20 rounded-2xl border border-white/10 mx-6 flex-1 justify-center max-w-lg shadow-[0_0_25px_rgba(255,165,0,0.15)] relative overflow-hidden group">
      {/* Animated background glow */}
      <motion.div
        animate={{
          opacity: [0.1, 0.2, 0.1],
          scale: [1, 1.1, 1],
        }}
        transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
        className="absolute inset-0 bg-gradient-to-r from-orange-500/5 via-transparent to-green-500/5 pointer-events-none"
      />

      <div className="flex items-center gap-4 relative z-10">
        <motion.div
          animate={{ rotate: [0, 10, -10, 0], scale: [1, 1.1, 1] }}
          transition={{ duration: 3, repeat: Infinity }}
        >
          <Flame size={20} className="text-orange-500 drop-shadow-[0_0_8px_rgba(249,115,22,0.6)]" />
        </motion.div>

        <div className="flex flex-col items-center">
          <motion.div
            initial={{ y: -5, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            className="flex items-center gap-3"
          >
            <span className="text-sm font-black text-white tracking-[0.2em] uppercase drop-shadow-md">
              Happy Lohri + Makar Sankranti
            </span>
            <motion.div
              animate={{
                scale: [1, 1.1, 1],
                filter: ["brightness(1)", "brightness(1.2)", "brightness(1)"]
              }}
              transition={{ duration: 2, repeat: Infinity }}
              className="flex items-center gap-1 bg-white/10 px-2 py-0.5 rounded-md border border-white/10"
            >
              <Flag size={12} className="text-white" />
              <span className="text-sm font-black text-white italic tracking-tighter">
                26 JAN
              </span>
            </motion.div>
          </motion.div>

          <div className="h-[1px] w-full bg-gradient-to-r from-orange-500/40 via-white/40 to-green-500/40 my-1" />

          <motion.span
            animate={{ opacity: [0.5, 0.9, 0.5] }}
            transition={{ duration: 2, repeat: Infinity }}
            className="text-[9px] text-white/70 font-black uppercase tracking-[0.3em]"
          >
            Celebrating India • Harmony • Prosperity
          </motion.span>
        </div>

        <motion.div
          animate={{ y: [0, -5, 0], rotate: [0, -15, 15, 0] }}
          transition={{ duration: 2, repeat: Infinity }}
        >
          <Wind size={20} className="text-blue-400 drop-shadow-[0_0_8px_rgba(96,165,250,0.6)]" />
        </motion.div>
      </div>

      {/* Shimmer effect on hover */}
      <div className="absolute inset-0 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000 bg-gradient-to-r from-transparent via-white/10 to-transparent" />
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
