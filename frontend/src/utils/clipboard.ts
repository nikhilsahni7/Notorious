import { Person } from "@/types/person";

const formatAddress = (address: string | undefined) => {
  if (!address) return "";
  return address
    .replace(/!/g, ", ")
    .replace(/, ,/g, ",")
    .replace(/^,/g, "")
    .replace(/,$/g, "");
};

// Format single person for Excel/CSV with tab-separated values
export const formatPersonForClipboard = (person: Person): string => {
  return `${person.name || ""}\t${person.fname || ""}\t${person.id || ""}\t${
    person.oid || ""
  }\t${person.mobile || ""}\t${person.alt || ""}\t${
    person.email || ""
  }\t${formatAddress(person.address)}\t${formatAddress(person.alt_address)}\t${
    person.year_of_registration || ""
  }`;
};

// Format multiple persons for Excel/CSV with headers
export const formatPersonsForClipboard = (persons: Person[]): string => {
  const header =
    "Name\tFather Name\tMaster ID\tOID\tMobile\tAlternate Phone\tEmail\tAddress\tAlt Address\tYear of Registration";
  const rows = persons.map((person) => formatPersonForClipboard(person));
  return [header, ...rows].join("\n");
};

export const copyToClipboard = async (text: string): Promise<void> => {
  await navigator.clipboard.writeText(text);
};
