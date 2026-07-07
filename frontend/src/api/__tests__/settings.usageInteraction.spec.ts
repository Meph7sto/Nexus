import { describe, expect, it } from "vitest";

import { normalizeUsageInteractionRetentionDays } from "@/api/admin/settings";

describe("admin settings usage interaction helpers", () => {
  it("keeps arbitrary non-negative integer retention days", () => {
    expect(normalizeUsageInteractionRetentionDays(0)).toBe(0);
    expect(normalizeUsageInteractionRetentionDays(7)).toBe(7);
    expect(normalizeUsageInteractionRetentionDays(36500)).toBe(36500);
  });

  it("coerces invalid retention values to the safe non-negative contract", () => {
    expect(normalizeUsageInteractionRetentionDays(-1)).toBe(0);
    expect(normalizeUsageInteractionRetentionDays("")).toBe(0);
    expect(normalizeUsageInteractionRetentionDays("13")).toBe(13);
    expect(normalizeUsageInteractionRetentionDays(Number.NaN)).toBe(0);
    expect(normalizeUsageInteractionRetentionDays(Number.POSITIVE_INFINITY)).toBe(0);
    expect(normalizeUsageInteractionRetentionDays(3.8)).toBe(3);
  });
});
