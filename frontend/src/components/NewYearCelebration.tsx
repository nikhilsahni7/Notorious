"use client";

import confetti from "canvas-confetti";
import { motion } from "framer-motion";
import { PartyPopper, Sparkles } from "lucide-react";
import React, { useEffect } from "react";

export const triggerNewYearBlast = () => {
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
    // since particles fall down, start a bit higher than random
    confetti({
      ...defaults,
      particleCount,
      origin: { x: randomInRange(0.1, 0.3), y: Math.random() - 0.2 },
      colors: ["#FFD700", "#FF69B4", "#00CED1", "#FFFFFF"],
    });
    confetti({
      ...defaults,
      particleCount,
      origin: { x: randomInRange(0.7, 0.9), y: Math.random() - 0.2 },
      colors: ["#FFD700", "#FF69B4", "#00CED1", "#FFFFFF"],
    });
  }, 250);
};

export const triggerQuickBlast = () => {
  confetti({
    particleCount: 150,
    spread: 70,
    origin: { y: 0.6 },
    colors: ["#FFD700", "#FF69B4", "#00CED1", "#FFFFFF", "#FF4500"],
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

export const NewYearHeader = () => {
  return (
    <div className="hidden lg:flex items-center px-8 py-2.5 bg-gradient-to-r from-indigo-600/20 via-fuchsia-600/30 to-amber-500/20 rounded-2xl border border-white/10 mx-6 flex-1 justify-center max-w-lg shadow-[0_0_25px_rgba(168,85,247,0.15)] relative overflow-hidden group">
      {/* Animated background glow */}
      <motion.div
        animate={{
          opacity: [0.1, 0.3, 0.1],
          scale: [1, 1.2, 1],
        }}
        transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
        className="absolute inset-0 bg-gradient-to-r from-transparent via-white/5 to-transparent pointer-events-none"
      />

      <div className="flex items-center gap-4 relative z-10">
        <motion.div
          animate={{ rotate: [0, 15, -15, 0], scale: [1, 1.2, 1] }}
          transition={{ duration: 2, repeat: Infinity }}
        >
          <Sparkles size={22} className="text-yellow-300 drop-shadow-[0_0_8px_rgba(253,224,71,0.6)]" />
        </motion.div>

        <div className="flex flex-col items-center">
          <motion.div
            initial={{ y: -5, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            className="flex items-center gap-3"
          >
            <span className="text-sm font-black text-white tracking-[0.3em] uppercase drop-shadow-md">
              Happy New Year
            </span>
            <motion.span
              animate={{
                color: ["#FDE047", "#F0ABFC", "#67E8F9", "#FDE047"],
                scale: [1, 1.1, 1]
              }}
              transition={{ duration: 3, repeat: Infinity }}
              className="text-xl font-black italic tracking-tighter drop-shadow-[0_0_10px_rgba(253,224,71,0.4)]"
            >
              2026
            </motion.span>
          </motion.div>

          <div className="h-[1px] w-full bg-gradient-to-r from-transparent via-white/20 to-transparent my-1" />

          <motion.span
            animate={{ opacity: [0.4, 0.8, 0.4] }}
            transition={{ duration: 2, repeat: Infinity }}
            className="text-[10px] text-white/60 font-black uppercase tracking-[0.4em] italic leading-none"
          >
            A Year of New Victories
          </motion.span>
        </div>

        <motion.div
          animate={{ y: [0, -5, 0], rotate: [0, -10, 10, 0] }}
          transition={{ duration: 1.5, repeat: Infinity }}
        >
          <PartyPopper size={22} className="text-pink-400 drop-shadow-[0_0_8px_rgba(244,114,182,0.6)]" />
        </motion.div>
      </div>

      {/* Shimmer effect on hover */}
      <div className="absolute inset-0 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000 bg-gradient-to-r from-transparent via-white/10 to-transparent" />
    </div>
  );
};

export const NewYearCelebration = () => {
  useEffect(() => {
    // Auto blast on mount for festive feel
    const blastTimer = setTimeout(() => {
        triggerQuickBlast();
    }, 1500);

    return () => clearTimeout(blastTimer);
  }, []);

  return null; // Side effects only
};
