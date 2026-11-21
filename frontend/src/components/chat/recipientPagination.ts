/** Load every page before returning, so a failed request cannot select a partial audience. */
export async function loadAllRecipients<T extends { id: string }>(
  fetchPage: (offset: number) => Promise<{ users: T[]; total: number }>,
): Promise<T[]> {
  const people: T[] = [];
  let total: number;
  do {
    const page = await fetchPage(people.length);
    total = page.total;
    if (page.users.length === 0 && people.length < total) {
      throw new Error("Incomplete user list");
    }
    people.push(...page.users);
  } while (people.length < total);
  return [...new Map(people.map((person) => [person.id, person])).values()];
}
