export interface SearchFields {
  id: string;
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
  email: string;
  year_of_registration: number;
}

export interface SearchResponse {
  total: number;
  results: Person[];
  took_ms: number;
}

export type SearchOperator = "AND" | "OR";
