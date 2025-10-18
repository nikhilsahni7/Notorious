export interface SearchFields {
  id: string;
  oid: string;
  name: string;
  fname: string;
  mobile: string;
  alt: string;
  email: string;
  address: string;
}

export interface Person {
  mobile: string;
  name: string;
  fname: string;
  address: string;
  alt_address: string;
  alt: string;
  id: string;
  oid: string;
  email: string;
  year_of_registration: number;
}

export interface SearchResponse {
  total: number;
  results: Person[];
  took_ms: number;
}

export type SearchOperator = "AND" | "OR";

// Refinement types
export interface Refinement {
  field: string;
  value: string;
}

export interface RefineRequest {
  base_query: string;
  base_operator: SearchOperator;
  refinements: Refinement[];
  refinement_operator: SearchOperator;
  size: number;
  from: number;
}

export interface RefineResponse {
  total: number;
  results: Person[];
  took_ms: number;
  searches_used_today: number;
  daily_search_limit: number;
  searches_remaining: number;
  is_refinement: boolean;
}

// Available fields for refinement
export const REFINEMENT_FIELDS = [
  { value: "name", label: "Name" },
  { value: "fname", label: "Father Name" },
  { value: "mobile", label: "Mobile" },
  { value: "alt", label: "Alt Phone" },
  { value: "email", label: "Email" },
  { value: "address", label: "Address" },
  { value: "id", label: "Master ID" },
  { value: "oid", label: "OID" },
] as const;
