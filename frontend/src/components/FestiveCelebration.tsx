"use client";

import confetti from "canvas-confetti";
import { motion } from "framer-motion";
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
    // Navratri celebration colors
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

const FloatingElement = ({
  children,
  delay = 0,
  duration = 4,
}: {
  children: React.ReactNode;
  delay?: number;
  duration?: number;
}) => (
  <motion.div
    initial={{ y: 0, opacity: 0 }}
    animate={{
      y: [0, -20, 0],
      opacity: [0.4, 0.8, 0.4],
      rotate: [0, 10, -10, 0],
    }}
    transition={{
      duration,
      repeat: Infinity,
      delay,
      ease: "easeInOut",
    }}
  >
    {children}
  </motion.div>
);

export const FestiveHeader = () => {
  const disclaimerText = "Disclaimer: for Maharashtra and Rajasthan data upcoming with live tracking process";

  return (
    <div className="hidden md:flex items-center mx-3 flex-1 max-w-3xl relative overflow-hidden rounded-full border border-amber-500/15 h-9 group hover:border-amber-400/30 transition-all duration-700"
      style={{
        background: "linear-gradient(135deg, rgba(120,53,15,0.25) 0%, rgba(154,52,18,0.15) 50%, rgba(120,53,15,0.25) 100%)",
        backdropFilter: "blur(16px)",
      }}
    >
      {/* Sweeping light line along the top edge */}
      <div className="absolute top-0 left-0 right-0 h-[1px] overflow-hidden">
        <div
          className="h-full w-[40%] bg-gradient-to-r from-transparent via-amber-400/70 to-transparent"
          style={{ animation: "sweep-line 4s ease-in-out infinite" }}
        />
      </div>

      {/* Bottom glow line */}
      <div className="absolute bottom-0 left-0 right-0 h-[1px] overflow-hidden opacity-40">
        <div
          className="h-full w-[30%] bg-gradient-to-r from-transparent via-orange-400/60 to-transparent"
          style={{ animation: "sweep-line 5s ease-in-out infinite", animationDelay: "2s" }}
        />
      </div>

      {/* Subtle inner glow */}
      <div className="absolute inset-0 bg-gradient-to-r from-amber-500/[0.03] via-transparent to-amber-500/[0.03] pointer-events-none" />

      {/* Pulsing indicator */}
      <div className="flex items-center gap-2 pl-4 pr-3 flex-shrink-0 z-10">
        <span className="relative flex h-[6px] w-[6px]">
          <span
            className="absolute inline-flex h-full w-full rounded-full bg-amber-400/50"
            style={{ animation: "glow-pulse 2s ease-in-out infinite" }}
          />
          <span className="relative inline-flex rounded-full h-[6px] w-[6px] bg-amber-400 shadow-[0_0_4px_rgba(251,191,36,0.6)]" />
        </span>
        <span className="text-[9px] font-semibold text-amber-400/80 uppercase tracking-[0.15em] whitespace-nowrap select-none">
          Live
        </span>
        <div className="w-[1px] h-4 bg-amber-500/15 ml-1" />
      </div>

      {/* Scrolling text area */}
      <div className="flex-1 overflow-hidden relative">
        {/* Smooth fade edges */}
        <div className="absolute left-0 top-0 bottom-0 w-12 z-10 pointer-events-none"
          style={{ background: "linear-gradient(to right, rgba(120,53,15,0.35), transparent)" }}
        />
        <div className="absolute right-0 top-0 bottom-0 w-12 z-10 pointer-events-none"
          style={{ background: "linear-gradient(to left, rgba(120,53,15,0.35), transparent)" }}
        />

        {/* The scrolling track — 4 copies for seamless loop */}
        <div
          className="flex whitespace-nowrap items-center"
          style={{
            animation: "marquee-scroll 24s linear infinite",
            willChange: "transform",
          }}
        >
          {[0, 1, 2, 3].map((i) => (
            <span key={i} className="inline-flex items-center gap-3 px-8 select-none">
              <span className="text-[11px] font-medium text-amber-200/80 tracking-[0.02em]"
                style={{ textShadow: "0 0 20px rgba(251,191,36,0.15)" }}
              >
                {disclaimerText}
              </span>
              <span className="text-amber-500/25 text-[8px]">◆</span>
            </span>
          ))}
        </div>
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
