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
    <div className="grid grid-cols-[repeat(14,minmax(0,1fr))] gap-2 text-sm bg-[#1a0f2e]/50 hover:bg-[#1a0f2e] transition-colors rounded overflow-hidden">
      <div className="col-span-2 bg-[#2D3748] text-white p-3 flex items-center">
        <div className="break-words">{person.name || "-"}</div>
      </div>

      <div className="col-span-1 bg-[#2D3748] text-white p-3 flex items-center justify-center">
        <div className="break-words text-xs text-center">
          {person.fname || "-"}
        </div>
      </div>

      <div className="col-span-1 bg-[#2D3748] text-white p-3 flex items-center justify-center">
        <div className="break-all text-xs font-mono">{person.id || "-"}</div>
      </div>

      <div className="col-span-1 bg-[#2D3748] text-white p-3 flex items-center justify-center">
        <div className="break-all text-xs font-mono">{person.oid || "-"}</div>
      </div>

      <div className="col-span-2 bg-[#68D391] text-gray-900 p-3 flex items-center">
        <div className="text-xs break-words">
          {formatAddress(person.address)}
        </div>
      </div>

      <div className="col-span-2 bg-[#F56565] text-white p-3 flex items-center">
        <div className="text-xs break-words">
          {formatAddress(person.alt_address)}
        </div>
      </div>

      <div className="col-span-1 bg-[#ECC94B] text-white p-3 flex items-center">
        <div className="text-xs break-all">{person.email || "-"}</div>
      </div>

      <div className="col-span-1 bg-[#4299E1] text-white p-3 flex items-center justify-center">
        <div className="font-semibold text-xs">
          {person.year_of_registration || "-"}
        </div>
      </div>

      <div className="col-span-1 bg-[#ED64A6] text-white p-3 flex items-center justify-center">
        <div className="text-xs">{person.mobile || "-"}</div>
      </div>

      <div className="col-span-1 bg-[#805AD5] text-white p-3 flex items-center justify-center">
        <div className="text-xs">{person.alt || "-"}</div>
      </div>

      <div className="col-span-1 bg-gray-700 text-white p-3 flex items-center justify-center">
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
  );
}
