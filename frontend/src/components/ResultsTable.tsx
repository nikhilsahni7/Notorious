import { Person } from "@/types/person";
import { Check, Copy } from "lucide-react";
import { PersonTableRow } from "./PersonTableRow";
import { TableHeader } from "./TableHeader";
import { Button } from "./ui/button";

interface ResultsTableProps {
  results: Person[];
  copiedIndex: number | null;
  onCopy: (person: Person, index: number) => void;
  onCopyAll?: () => void;
  isCopyingAll?: boolean;
}

export function ResultsTable({
  results,
  copiedIndex,
  onCopy,
  onCopyAll,
  isCopyingAll = false,
}: ResultsTableProps) {
  return (
    <>
      {onCopyAll && results.length > 0 && (
        <div className="mb-3 flex justify-end">
          <Button
            onClick={onCopyAll}
            disabled={isCopyingAll}
            className="bg-purple-600 hover:bg-purple-700 text-white"
          >
            {isCopyingAll ? (
              <>
                <Check className="h-4 w-4 mr-2" />
                Copied All!
              </>
            ) : (
              <>
                <Copy className="h-4 w-4 mr-2" />
                Copy All Results
              </>
            )}
          </Button>
        </div>
      )}
      <TableHeader />
      <div className="space-y-2">
        {results.map((person, index) => (
          <PersonTableRow
            key={`${person.id}-${person.mobile}-${index}`}
            person={person}
            index={index}
            isCopied={copiedIndex === index}
            onCopy={onCopy}
          />
        ))}
      </div>
    </>
  );
}
