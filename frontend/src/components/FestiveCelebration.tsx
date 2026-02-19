"use client";

import confetti from "canvas-confetti";
import { motion } from "framer-motion";
import { Palette } from "lucide-react";
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
    // Holi — vibrant multi-colors
    confetti({
      ...defaults,
      particleCount,
      origin: { x: randomInRange(0.1, 0.3), y: Math.random() - 0.2 },
      colors: ["#FF1493", "#FFFF00", "#00BFFF", "#FF8C00", "#32CD32"],
    });
    confetti({
      ...defaults,
      particleCount,
      origin: { x: randomInRange(0.7, 0.9), y: Math.random() - 0.2 },
      colors: ["#FF1493", "#FFFF00", "#00BFFF", "#FF8C00", "#32CD32"],
    });
  }, 250);
};

export const triggerFestiveQuickBlast = () => {
  confetti({
    particleCount: 150,
    spread: 70,
    origin: { y: 0.6 },
    colors: ["#FF1493", "#FFFF00", "#00BFFF", "#FF8C00", "#32CD32"],
    ticks: 200,
    gravity: 1.2,
    scalar: 1.2,
    shapes: ["circle"],
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
    <div
      className="hidden lg:flex items-center px-10 py-3 rounded-2xl border border-white/20 mx-6 flex-1 justify-center max-w-xl shadow-[0_0_40px_rgba(255,20,147,0.3)] relative overflow-hidden group transition-all duration-500 hover:shadow-[0_0_50px_rgba(255,20,147,0.5)]"
      style={{
        backgroundImage: 'url("/holi-header.png")',
        backgroundSize: 'cover',
        backgroundPosition: 'center',
      }}
    >
      {/* Dark overlay for better readability */}
      <div className="absolute inset-0 bg-black/30 group-hover:bg-black/20 transition-colors duration-500" />

      {/* Animated light sweep */}
      <motion.div
        animate={{
          x: ["-100%", "200%"],
        }}
        transition={{
          duration: 3,
          repeat: Infinity,
          ease: "linear",
          delay: 1
        }}
        className="absolute inset-0 bg-gradient-to-r from-transparent via-white/10 to-transparent skew-x-12 pointer-events-none"
      />

      <div className="flex items-center gap-8 relative z-10">
        <motion.div
          animate={{ rotate: [-20, 20, -20], scale: [1, 1.2, 1] }}
          transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
          className="bg-white/10 backdrop-blur-md p-2 rounded-full border border-white/20 shadow-lg"
        >
          <Palette size={22} className="text-pink-400 drop-shadow-[0_0_8px_rgba(255,20,147,0.8)]" />
        </motion.div>

        <div className="flex flex-col items-center bg-black/40 backdrop-blur-md px-6 py-2 rounded-xl border border-white/10">
          <motion.div
            initial={{ y: -5, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            className="flex items-center"
          >
            <span className="text-lg font-black text-white tracking-[0.4em] uppercase drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">
              Happy Holi
            </span>
          </motion.div>

          <div className="h-[2px] w-full bg-gradient-to-r from-transparent via-pink-500 via-yellow-400 via-blue-500 to-transparent my-1.5 opacity-80" />

          <motion.span
            animate={{ opacity: [0.7, 1, 0.7], y: [0, -1, 0] }}
            transition={{ duration: 2.5, repeat: Infinity }}
            className="text-[11px] text-white/90 font-black uppercase tracking-[0.5em] space-x-1"
          >
            Celebrate with Colors 🌈
          </motion.span>
        </div>

        <motion.div
          animate={{ rotate: [20, -20, 20], scale: [1, 1.2, 1] }}
          transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
          className="bg-white/10 backdrop-blur-md p-2 rounded-full border border-white/20 shadow-lg"
        >
          <Palette size={22} className="text-blue-400 drop-shadow-[0_0_8px_rgba(0,191,255,0.8)]" />
        </motion.div>
      </div>
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
