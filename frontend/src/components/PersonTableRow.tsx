import { Person } from "@/types/person";
import { Check, Copy } from "lucide-react";

interface PersonTableRowProps {
  readonly person: Person;
  readonly index: number;
  readonly isCopied: boolean;
  readonly onCopy: (person: Person, index: number) => void;
}

const formatAddress = (address: string | undefined) => {
  if (!address) return "-";
  return address
    .replace(/!/g, ", ")
    .replace(/, ,/g, ",")
    .replace(/^,/g, "")
    .replace(/,$/g, "");
};

export function PersonTableRow({
  person,
  index,
  isCopied,
  onCopy,
}: PersonTableRowProps) {
  return (
    <>
      {/* Mobile Card View */}
      <div className="md:hidden mb-4 rounded-lg overflow-hidden border border-gray-700 shadow-lg bg-[#2D1B4E]">
        {/* Header: Name and Action */}
        <div className="bg-[#2D3748] p-3 flex justify-between items-start">
          <div>
            <div className="text-sm font-bold text-white">
              {person.name || "-"}
            </div>
            <div className="text-xs text-gray-400 mt-1">
              Father: {person.fname || "-"}
            </div>
          </div>
          <button
            onClick={() => onCopy(person, index)}
            className="bg-gray-700 hover:bg-gray-600 p-2 rounded transition-colors ml-2"
            title="Copy all data"
          >
            {isCopied ? (
              <Check className="h-4 w-4 text-green-400" />
            ) : (
              <Copy className="h-4 w-4 text-white" />
            )}
          </button>
        </div>

        {/* IDs Grid */}
        <div className="grid grid-cols-3 gap-1 bg-[#1a0f2e] p-2 border-b border-gray-700">
          <div className="text-center">
            <div className="text-[10px] text-gray-500 uppercase">Master ID</div>
            <div className="text-xs text-white font-mono break-all">
              {person.id || "-"}
            </div>
          </div>
          <div className="text-center border-l border-gray-700">
            <div className="text-[10px] text-gray-500 uppercase">ID</div>
            <div className="text-xs text-white font-mono break-all">
              {person.oid || "-"}
            </div>
          </div>
          <div className="text-center border-l border-gray-700">
            <div className="text-[10px] text-gray-500 uppercase">Year</div>
            <div className="text-xs text-[#4299E1] font-bold">
              {person.year_of_registration || "-"}
            </div>
          </div>
        </div>

        {/* Contact Info */}
        <div className="grid grid-cols-1 gap-2 p-2">
          {/* Phones & Email */}
          <div className="grid grid-cols-3 gap-2">
            <div className="bg-[#ED64A6]/20 border border-[#ED64A6] rounded p-2 text-center flex flex-col justify-center">
              <div className="text-[10px] text-[#ED64A6] uppercase mb-1">
                Mobile
              </div>
              <div className="text-xs text-white break-all">
                {person.mobile || "-"}
              </div>
            </div>
            <div className="bg-[#805AD5]/20 border border-[#805AD5] rounded p-2 text-center flex flex-col justify-center">
              <div className="text-[10px] text-[#805AD5] uppercase mb-1">
                Alt Phone
              </div>
              <div className="text-xs text-white break-all">
                {person.alt || "-"}
              </div>
            </div>
            <div className={`rounded p-2 text-center flex flex-col justify-center transition-all ${person.email ? 'bg-orange-500/10 border border-orange-500/30 text-orange-400' : 'bg-white/5 border border-white/10 text-white/40'}`}>
              <div className="text-[10px] uppercase mb-1 font-bold">
                Email
              </div>
              <div className={`text-xs break-all ${person.email ? 'text-white font-medium' : ''}`}>
                {person.email || "-"}
              </div>
            </div>
          </div>

          {/* Addresses */}
          <div className="space-y-2 mt-1">
            <div className="bg-[#68D391]/20 border border-[#68D391] rounded p-2">
              <div className="text-[10px] text-[#68D391] uppercase mb-1 font-bold">
                Address
              </div>
              <div className="text-xs text-white break-words">
                {formatAddress(person.address)}
              </div>
            </div>
            <div className="bg-[#F56565]/20 border border-[#F56565] rounded p-2">
              <div className="text-[10px] text-[#F56565] uppercase mb-1 font-bold">
                Alt Address
              </div>
              <div className="text-xs text-white break-words">
                {formatAddress(person.alt_address)}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Desktop Grid View */}
      <div className="hidden md:grid grid-cols-[repeat(16,minmax(0,1fr))] gap-2 text-sm bg-white/5 hover:bg-white/10 transition-all duration-300 rounded-lg overflow-hidden border border-white/5 hover:border-white/15 hover:shadow-[0_4px_20px_rgba(0,0,0,0.25)] hover:-translate-y-[1px] transform">
        {/* Master ID */}
        <div className="col-span-1 bg-[#2D3748]/85 text-white p-3 flex items-center justify-center">
        <div className="break-all text-xs font-mono">{person.id || "-"}</div>
      </div>

      {/* ID */}
      <div className="col-span-1 bg-[#2D3748]/85 text-white p-3 flex items-center justify-center">
        <div className="break-all text-xs font-mono">{person.oid || "-"}</div>
      </div>

      {/* Name */}
      <div className="col-span-2 bg-[#2D3748]/85 text-white p-3 flex items-center">
        <div className="break-words text-xs">{person.name || "-"}</div>
      </div>

      {/* Father Name */}
      <div className="col-span-1 bg-[#2D3748]/85 text-white p-3 flex items-center justify-center">
        <div className="break-words text-xs text-center">
          {person.fname || "-"}
        </div>
      </div>

      {/* Mobile */}
      <div className="col-span-1 bg-[#ED64A6] text-white p-3 flex items-center justify-center">
        <div className="text-xs">{person.mobile || "-"}</div>
      </div>

      {/* Alt Phone */}
      <div className="col-span-1 bg-[#805AD5] text-white p-3 flex items-center justify-center">
        <div className="text-xs">{person.alt || "-"}</div>
      </div>

      {/* Email */}
      <div className={`col-span-1 p-3 flex items-center justify-center text-xs break-all transition-colors ${person.email ? 'bg-[#f97316] text-black font-extrabold' : 'bg-white/5 text-white/40'}`}>
        <div className="text-center">{person.email || "-"}</div>
      </div>

      {/* Address */}
      <div className="col-span-3 bg-[#68D391] text-gray-900 p-3 flex items-center">
        <div className="text-xs break-words">
          {formatAddress(person.address)}
        </div>
      </div>

      {/* Alt Address */}
      <div className="col-span-3 bg-[#F56565] text-white p-3 flex items-center">
        <div className="text-xs break-words">
          {formatAddress(person.alt_address)}
        </div>
      </div>

      {/* Year */}
      <div className="col-span-1 bg-[#4299E1] text-white p-3 flex items-center justify-center">
        <div className="font-semibold text-xs">
          {person.year_of_registration || "-"}
        </div>
      </div>

      {/* Action */}
      <div className="col-span-1 bg-gray-700/85 text-white p-3 flex items-center justify-center">
        <button
          onClick={() => onCopy(person, index)}
          className="hover:bg-gray-600 p-1 rounded transition-colors"
          title="Copy all data"
        >
          {isCopied ? (
            <Check className="h-4 w-4 text-green-400" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </button>
      </div>
      </div>
    </>
  );
}
