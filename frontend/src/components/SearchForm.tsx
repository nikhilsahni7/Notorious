import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { SearchFields, SearchOperator } from "@/types/person";
import { Search } from "lucide-react";

interface SearchFormProps {
  searchFields: SearchFields;
  operator: SearchOperator;
  loading: boolean;
  onFieldChange: (field: keyof SearchFields, value: string) => void;
  onOperatorChange: (operator: SearchOperator) => void;
  onSearch: () => void;
}

export function SearchForm({
  searchFields,
  operator,
  loading,
  onFieldChange,
  onOperatorChange,
  onSearch,
}: SearchFormProps) {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      onSearch();
    }
  };

  return (
    <div className="mb-6 space-y-3">
      <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
        <Input
          id="id"
          placeholder="Enter master ID..."
          value={searchFields.id}
          onChange={(e) => onFieldChange("id", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
        />
        <Input
          id="name"
          placeholder="Enter name..."
          value={searchFields.name}
          onChange={(e) => onFieldChange("name", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
        />
        <Input
          id="fname"
          placeholder="Enter father's name..."
          value={searchFields.fname}
          onChange={(e) => onFieldChange("fname", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
        />
        <Input
          id="mobile"
          placeholder="Enter mobile number..."
          value={searchFields.mobile}
          onChange={(e) => onFieldChange("mobile", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
        <Input
          id="alt"
          placeholder="Enter alternate number..."
          value={searchFields.alt}
          onChange={(e) => onFieldChange("alt", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400 md:col-span-1"
        />
        <Input
          id="email"
          placeholder="Enter email..."
          value={searchFields.email}
          onChange={(e) => onFieldChange("email", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400 md:col-span-1"
        />
        <Input
          id="address"
          placeholder="Enter address..."
          value={searchFields.address}
          onChange={(e) => onFieldChange("address", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#1a0f2e] border-gray-600 text-white placeholder:text-gray-400 md:col-span-2"
        />
      </div>

      <div className="flex items-center justify-between gap-4">
        <Button
          onClick={onSearch}
          disabled={loading}
          className="bg-pink-500 hover:bg-pink-600 text-white px-6"
        >
          {loading ? (
            <>
              <Spinner size="sm" className="mr-2" />
              Searching...
            </>
          ) : (
            <>
              <Search className="h-4 w-4 mr-2" />
              Search
            </>
          )}
        </Button>
        <div className="flex items-center gap-2 text-white text-sm">
          <span className={operator === "AND" ? "text-white" : "text-gray-400"}>
            AND
          </span>
          <Switch
            id="operator"
            checked={operator === "OR"}
            onCheckedChange={(checked) =>
              onOperatorChange(checked ? "OR" : "AND")
            }
          />
          <span className={operator === "OR" ? "text-white" : "text-gray-400"}>
            OR
          </span>
        </div>
      </div>
    </div>
  );
}
