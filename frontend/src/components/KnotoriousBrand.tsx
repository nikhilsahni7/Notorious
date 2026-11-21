import Image from "next/image";

interface KnotoriousBrandProps {
  subtitle?: string;
  className?: string;
  size?: "sm" | "md" | "lg";
}

export function KnotoriousMark({ size = 36 }: { size?: number }) {
  return <Image src="/knotorious-mark.png" width={size} height={size} alt="" className="shrink-0 rounded-xl" />;
}

/** Shared product identity; the image is decorative alongside the wordmark. */
export function KnotoriousBrand({ subtitle, className = "", size = "md" }: KnotoriousBrandProps) {
  const dimensions = { sm: 36, md: 44, lg: 52 };
  const dimension = dimensions[size];
  return (
    <div className={`flex min-w-0 items-center gap-3 ${className}`}>
      <KnotoriousMark size={dimension} />
      <div className="min-w-0">
        <span className="block text-base leading-5 tracking-tight text-white">Knotorious</span>
        {subtitle && <span className="mt-1 block truncate text-xs text-slate-400">{subtitle}</span>}
      </div>
    </div>
  );
}
