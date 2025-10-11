import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { SearchFields, SearchOperator } from "@/types/person";

interface SearchFormProps {
  searchFields: SearchFields;
  operator: SearchOperator;
  onFieldChange: (field: keyof SearchFields, value: string) => void;
  onOperatorChange: (operator: SearchOperator) => void;
  onSearch?: () => void;
}

export function SearchForm({
  searchFields,
  operator,
  onFieldChange,
  onOperatorChange,
  onSearch,
}: SearchFormProps) {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && onSearch) {
      onSearch();
    }
  };

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
        <Input
          id="id"
          placeholder="Master ID"
          value={searchFields.id}
          onChange={(e) => onFieldChange("id", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-500 h-9 text-sm"
        />
        <Input
          id="name"
          placeholder="Name"
          value={searchFields.name}
          onChange={(e) => onFieldChange("name", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-500 h-9 text-sm"
        />
        <Input
          id="fname"
          placeholder="Father's Name"
          value={searchFields.fname}
          onChange={(e) => onFieldChange("fname", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-500 h-9 text-sm"
        />
        <Input
          id="mobile"
          placeholder="Mobile"
          value={searchFields.mobile}
          onChange={(e) => onFieldChange("mobile", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-500 h-9 text-sm"
        />
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
        <Input
          id="alt"
          placeholder="Alternate Number"
          value={searchFields.alt}
          onChange={(e) => onFieldChange("alt", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-500 h-9 text-sm"
        />
        <Input
          id="email"
          placeholder="Email"
          value={searchFields.email}
          onChange={(e) => onFieldChange("email", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-500 h-9 text-sm"
        />
        <Input
          id="address"
          placeholder="Address"
          value={searchFields.address}
          onChange={(e) => onFieldChange("address", e.target.value)}
          onKeyDown={handleKeyDown}
          className="bg-[#2D1B4E] border-gray-600 text-white placeholder:text-gray-500 h-9 text-sm md:col-span-2"
        />
      </div>

      <div className="flex items-center justify-end gap-2">
        <span className="text-xs text-gray-400">Search Mode:</span>
        <div className="flex items-center gap-2">
          <span className={`text-xs ${operator === "AND" ? "text-white font-medium" : "text-gray-500"}`}>
            AND
          </span>
          <Switch
            id="operator"
            checked={operator === "OR"}
            onCheckedChange={(checked) =>
              onOperatorChange(checked ? "OR" : "AND")
            }
          />
          <span className={`text-xs ${operator === "OR" ? "text-white font-medium" : "text-gray-500"}`}>
            OR
          </span>
        </div>
      </div>
    </div>
  );
}
