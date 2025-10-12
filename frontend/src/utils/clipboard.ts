import { Person } from "@/types/person";

const formatAddress = (address: string | undefined) => {
  if (!address) return "";
  return address
    .replace(/!/g, ", ")
    .replace(/, ,/g, ",")
    .replace(/^,/g, "")
    .replace(/,$/g, "");
};

export const formatPersonForClipboard = (person: Person): string => {
  return `Name: ${person.name}
Father Name: ${person.fname}
Master ID: ${person.id}
OID: ${person.oid}
Mobile: ${person.mobile}
Alternate Phone: ${person.alt}
Email: ${person.email}
Address: ${formatAddress(person.address)}
Alt Address: ${formatAddress(person.alt_address)}
Year of Registration: ${person.year_of_registration}`;
};

export const copyToClipboard = async (text: string): Promise<void> => {
  await navigator.clipboard.writeText(text);
};
