import { Input } from "@/components/ui/input";
import { SearchFields, SearchOperator } from "@/types/person";
import { AtSign, Fingerprint, Hash, MapPin, Phone, Smartphone, UserRound, UsersRound, type LucideIcon } from "lucide-react";

interface SearchFormProps {
  searchFields: SearchFields;
  operator: SearchOperator;
  onFieldChange: (field: keyof SearchFields, value: string) => void;
  onOperatorChange: (operator: SearchOperator) => void;
  onSearch?: () => void;
}

const FIELDS: { key: keyof SearchFields; label: string; placeholder: string; icon: LucideIcon }[] = [
  { key: "id", label: "Master ID", placeholder: "Enter master ID", icon: Fingerprint },
  { key: "oid", label: "ID", placeholder: "Enter ID", icon: Hash },
  { key: "name", label: "Name", placeholder: "Enter name", icon: UserRound },
  { key: "fname", label: "Father’s name", placeholder: "Enter father’s name", icon: UsersRound },
  { key: "mobile", label: "Mobile", placeholder: "Enter mobile number", icon: Smartphone },
  { key: "alt", label: "Alternate number", placeholder: "Enter alternate number", icon: Phone },
  { key: "email", label: "Email", placeholder: "Enter email address", icon: AtSign },
  { key: "address", label: "Address", placeholder: "Enter address", icon: MapPin },
];

export function SearchForm({ searchFields, operator, onFieldChange, onOperatorChange, onSearch }: SearchFormProps) {
  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-sky-900/50 pb-5">
        <div>
          <h2 className="text-base text-white">Search records</h2>
          <p className="mt-1 text-sm text-slate-400">Enter one or more details to find a record.</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-slate-300">Match</span>
          <div role="group" aria-label="Search match mode" className="flex rounded-lg border border-sky-800/70 bg-[#07172e] p-1">
            {(["AND", "OR"] as const).map((mode) => (
              <button key={mode} type="button" aria-pressed={operator === mode} onClick={() => onOperatorChange(mode)} className={`rounded-md px-4 py-1.5 text-sm transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-300 ${operator === mode ? "bg-sky-500 text-[#041525]" : "text-slate-300 hover:bg-sky-900/40 hover:text-white"}`}>
                {mode}
              </button>
            ))}
          </div>
        </div>
      </div>

      <p className="mt-4 text-xs text-slate-400" aria-live="polite">
        {operator === "AND" ? "AND: results must match all entered details." : "OR: results can match any entered detail."}
      </p>
      <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {FIELDS.map(({ key, label, placeholder, icon: Icon }) => (
          <div key={key} className="min-w-0">
            <label htmlFor={key} className="mb-2 block text-sm text-slate-200">{label}</label>
            <div className="relative">
              <Icon aria-hidden="true" className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <Input id={key} placeholder={placeholder} value={searchFields[key] ?? ""} onChange={(event) => onFieldChange(key, event.target.value)} onKeyDown={(event) => { if (event.key === "Enter" && onSearch) onSearch(); }} className="h-11 rounded-lg border-sky-900/70 bg-[#0b203e] pl-10 text-white placeholder:text-slate-500 hover:border-sky-700 focus-visible:border-sky-400 focus-visible:ring-sky-400/30" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
