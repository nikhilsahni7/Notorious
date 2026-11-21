import { KnotoriousBrand } from "@/components/KnotoriousBrand";
import { ArrowLeft, ArrowRight, MapPin } from "lucide-react";
import Link from "next/link";

export default function FindMyLocationPage() {
  return (
    <main className="min-h-screen bg-[#040f25] px-4 py-6 text-slate-100 md:px-6">
      <div className="mx-auto max-w-6xl">
        <header className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-sky-900/50 bg-[#0a1c38] px-5 py-4">
          <KnotoriousBrand subtitle="Location" />
          <Link href="/search" className="inline-flex items-center gap-2 rounded-lg border border-sky-800/70 px-3 py-2 text-sm text-sky-200 transition-colors hover:bg-sky-900/40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-300">
            <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to search
          </Link>
        </header>

        <section className="mt-6 overflow-hidden rounded-2xl border border-sky-900/50 bg-[#0a1c38]">
          <div className="grid items-center gap-6 px-6 py-10 md:grid-cols-2 md:gap-12 md:px-12 md:py-16">
            <div className="order-2 md:order-1">
              <span className="inline-flex items-center gap-2 rounded-full border border-sky-700/60 bg-sky-500/10 px-3 py-1 text-xs text-sky-200">
                <span className="h-1.5 w-1.5 rounded-full bg-sky-300" aria-hidden="true" /> Coming soon
              </span>
              <h1 className="mt-5 text-2xl tracking-tight text-white md:text-3xl">Find my location</h1>
              <p className="mt-4 max-w-md text-sm leading-6 text-slate-300">
                We’re working on a new location feature. It isn’t available yet, but you can continue searching while it’s being prepared.
              </p>
              <Link href="/search" className="mt-6 inline-flex items-center gap-2 rounded-lg bg-sky-500 px-4 py-3 text-sm text-slate-950 transition-colors hover:bg-sky-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-300">
                Back to search <ArrowRight className="h-4 w-4" aria-hidden="true" />
              </Link>
            </div>

            <div aria-hidden="true" className="order-1 flex min-h-56 items-center justify-center md:order-2 md:min-h-80">
              <div className="relative flex h-56 w-56 items-center justify-center rounded-full border border-sky-800/40 bg-[#071a33] md:h-72 md:w-72">
                <div className="absolute inset-6 rounded-full border border-sky-800/50" />
                <div className="absolute inset-12 rounded-full border border-sky-700/40 bg-sky-500/5" />
                <div className="relative flex h-20 w-20 items-center justify-center rounded-2xl border border-red-400/30 bg-[#281a31] text-red-400 shadow-lg shadow-black/20 motion-safe:animate-[location-pin_2.4s_ease-in-out_infinite]">
                  <MapPin className="h-10 w-10" />
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
