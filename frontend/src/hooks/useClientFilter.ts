import { Person } from "@/types/person";
import { useMemo } from "react";

export function useClientFilter(results: Person[], query: string) {
  return useMemo(() => {
    if (!query.trim()) {
      return results;
    }

    const lowercaseQuery = query.toLowerCase();
    return results.filter((person) => {
      return (
        person.name?.toLowerCase().includes(lowercaseQuery) ||
        person.fname?.toLowerCase().includes(lowercaseQuery) ||
        person.mobile?.toLowerCase().includes(lowercaseQuery) ||
        person.address?.toLowerCase().includes(lowercaseQuery) ||
        person.alt_address?.toLowerCase().includes(lowercaseQuery) ||
        person.alt?.toLowerCase().includes(lowercaseQuery) ||
        person.id?.toLowerCase().includes(lowercaseQuery) ||
        person.oid?.toLowerCase().includes(lowercaseQuery) ||
        person.email?.toLowerCase().includes(lowercaseQuery)
      );
    });
  }, [results, query]);
}
