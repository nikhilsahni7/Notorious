import { Person } from "@/types/person";
import { PersonTableRow } from "./PersonTableRow";
import { TableHeader } from "./TableHeader";

interface ResultsTableProps {
  results: Person[];
  copiedIndex: number | null;
  onCopy: (person: Person, index: number) => void;
}

export function ResultsTable({
  results,
  copiedIndex,
  onCopy,
}: ResultsTableProps) {
  return (
    <>
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
