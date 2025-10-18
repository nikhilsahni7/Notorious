"use client";

import { Button } from "@/components/ui/button";
import { Refinement, REFINEMENT_FIELDS } from "@/types/person";
import { Filter, Plus, X } from "lucide-react";
import { useState } from "react";

interface RefineSearchProps {
  totalResults: number;
  activeRefinements: Refinement[];
  onAddRefinement: (refinement: Refinement) => void;
  onRemoveRefinement: (index: number) => void;
  onClearRefinements: () => void;
  onRefine: () => void;
  isRefining: boolean;
}

export function RefineSearch({
  totalResults,
  activeRefinements,
  onAddRefinement,
  onRemoveRefinement,
  onClearRefinements,
  onRefine,
  isRefining,
}: RefineSearchProps) {
  const [selectedField, setSelectedField] = useState("name");
  const [refinementValue, setRefinementValue] = useState("");

  const handleAddRefinement = () => {
    if (refinementValue.trim()) {
      onAddRefinement({
        field: selectedField,
        value: refinementValue.trim(),
      });
      setRefinementValue("");
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && refinementValue.trim()) {
      handleAddRefinement();
    }
  };

  const getFieldLabel = (fieldValue: string) => {
    const field = REFINEMENT_FIELDS.find((f) => f.value === fieldValue);
    return field?.label || fieldValue;
  };

  if (totalResults === 0) {
    return null;
  }

  return (
    <div className="bg-[#1a0f2e] p-4 rounded-lg border border-purple-500/30 mb-3">
      <div className="flex items-center gap-2 mb-3">
        <Filter className="h-5 w-5 text-purple-400" />
        <h3 className="text-lg font-semibold text-white">
          Refine Results
          <span className="text-sm text-gray-400 ml-2 font-normal">
            (doesn&apos;t use search credits)
          </span>
        </h3>
      </div>

      <div className="text-sm text-gray-300 mb-3">
        Found <span className="font-bold text-purple-400">{totalResults}</span>{" "}
        results. Add filters to narrow down your search.
      </div>

      {/* Add Refinement Form */}
      <div className="flex gap-2 mb-3">
        <select
          value={selectedField}
          onChange={(e) => setSelectedField(e.target.value)}
          className="bg-[#2D1B4E] border border-gray-600 rounded px-3 py-2 text-white focus:outline-none focus:border-purple-500"
        >
          {REFINEMENT_FIELDS.map((field) => (
            <option key={field.value} value={field.value}>
              {field.label}
            </option>
          ))}
        </select>

        <input
          type="text"
          value={refinementValue}
          onChange={(e) => setRefinementValue(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder={`Enter ${getFieldLabel(selectedField).toLowerCase()}...`}
          className="flex-1 bg-[#2D1B4E] border border-gray-600 rounded px-3 py-2 text-white placeholder-gray-500 focus:outline-none focus:border-purple-500"
        />

        <Button
          onClick={handleAddRefinement}
          disabled={!refinementValue.trim()}
          className="bg-purple-500 hover:bg-purple-600 text-white"
        >
          <Plus className="h-4 w-4 mr-1" />
          Add
        </Button>
      </div>

      {/* Active Refinements */}
      {activeRefinements.length > 0 && (
        <div className="mb-3">
          <div className="flex items-center justify-between mb-2">
            <div className="text-sm font-medium text-gray-300">
              Active Filters ({activeRefinements.length})
            </div>
            <Button
              onClick={onClearRefinements}
              variant="outline"
              size="sm"
              className="bg-transparent border-red-500 text-red-400 hover:bg-red-500/10 text-xs"
            >
              <X className="h-3 w-3 mr-1" />
              Clear All
            </Button>
          </div>

          <div className="flex flex-wrap gap-2">
            {activeRefinements.map((refinement, index) => (
              <div
                key={index}
                className="bg-purple-500/20 border border-purple-500 rounded px-3 py-1 flex items-center gap-2 group"
              >
                <span className="text-purple-300 text-sm font-medium">
                  {getFieldLabel(refinement.field)}:
                </span>
                <span className="text-white text-sm">{refinement.value}</span>
                <button
                  onClick={() => onRemoveRefinement(index)}
                  className="text-purple-300 hover:text-red-400 transition-colors"
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Refine Button */}
      {activeRefinements.length > 0 && (
        <Button
          onClick={onRefine}
          disabled={isRefining}
          className="w-full bg-gradient-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600 text-white font-semibold"
        >
          <Filter className="h-4 w-4 mr-2" />
          {isRefining ? "Refining..." : "Apply Filters"}
        </Button>
      )}
    </div>
  );
}
