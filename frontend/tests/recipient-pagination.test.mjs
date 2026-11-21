import assert from "node:assert/strict";
import test from "node:test";
import { loadAllRecipients } from "../src/components/chat/recipientPagination.ts";

test("select all loads users beyond the first 100", async () => {
  const people = Array.from({ length: 215 }, (_, index) => ({ id: String(index) }));
  const offsets = [];
  const result = await loadAllRecipients(async (offset) => {
    offsets.push(offset);
    return { users: people.slice(offset, offset + 100), total: people.length };
  });
  assert.deepEqual(offsets, [0, 100, 200]);
  assert.deepEqual(result, people);
});

test("select all fails instead of returning a partial audience", async () => {
  await assert.rejects(() => loadAllRecipients(async (offset) => ({
    users: offset === 0 ? [{ id: "first" }] : [], total: 2,
  })), /Incomplete user list/);
});
