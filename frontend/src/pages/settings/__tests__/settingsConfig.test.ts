import { describe, expect, it } from "bun:test";
import "../../../../test/setup";

describe("settingsConfig", () => {
    it("exports validation helpers and buildSettingsTree", async () => {
        const { isValidIpAddressOrCidr, HOSTNAME_REGEX, WORKGROUP_REGEX, buildSettingsTree } =
            await import("../settingsConfig");

        expect(typeof isValidIpAddressOrCidr).toBe("function");
        expect(HOSTNAME_REGEX).toBeInstanceOf(RegExp);
        expect(WORKGROUP_REGEX).toBeInstanceOf(RegExp);
        expect(typeof buildSettingsTree).toBe("function");
    });

    describe("isValidIpAddressOrCidr", () => {
        it("accepts valid IPv4 addresses", async () => {
            const { isValidIpAddressOrCidr } = await import("../settingsConfig");
            expect(isValidIpAddressOrCidr("192.168.1.1")).toBe(true);
            expect(isValidIpAddressOrCidr("10.0.0.0")).toBe(true);
            expect(isValidIpAddressOrCidr("0.0.0.0")).toBe(true);
            expect(isValidIpAddressOrCidr("255.255.255.255")).toBe(true);
        });

        it("accepts valid IPv4 CIDR notation", async () => {
            const { isValidIpAddressOrCidr } = await import("../settingsConfig");
            expect(isValidIpAddressOrCidr("192.168.0.0/24")).toBe(true);
            expect(isValidIpAddressOrCidr("10.0.0.0/8")).toBe(true);
            expect(isValidIpAddressOrCidr("0.0.0.0/0")).toBe(true);
            expect(isValidIpAddressOrCidr("192.168.1.1/32")).toBe(true);
        });

        it("accepts valid IPv6 addresses", async () => {
            const { isValidIpAddressOrCidr } = await import("../settingsConfig");
            expect(isValidIpAddressOrCidr("::1")).toBe(true);
            expect(isValidIpAddressOrCidr("fe80::1")).toBe(true);
            expect(isValidIpAddressOrCidr("2001:db8::1")).toBe(true);
        });

        it("accepts valid IPv6 CIDR notation", async () => {
            const { isValidIpAddressOrCidr } = await import("../settingsConfig");
            expect(isValidIpAddressOrCidr("2001:db8::/32")).toBe(true);
            expect(isValidIpAddressOrCidr("::/0")).toBe(true);
        });

        it("rejects invalid values", async () => {
            const { isValidIpAddressOrCidr } = await import("../settingsConfig");
            expect(isValidIpAddressOrCidr("")).toBe(false);
            expect(isValidIpAddressOrCidr("not-an-ip")).toBe(false);
            expect(isValidIpAddressOrCidr("999.999.999.999")).toBe(false);
            expect(isValidIpAddressOrCidr("192.168.0.0/33")).toBe(false);
            expect(isValidIpAddressOrCidr("192.168.1")).toBe(false);
        });

        it("rejects non-string input", async () => {
            const { isValidIpAddressOrCidr } = await import("../settingsConfig");
            expect(isValidIpAddressOrCidr(null as unknown as string)).toBe(false);
            expect(isValidIpAddressOrCidr(undefined as unknown as string)).toBe(false);
            expect(isValidIpAddressOrCidr(123 as unknown as string)).toBe(false);
        });
    });

    describe("HOSTNAME_REGEX", () => {
        it("accepts valid hostnames", async () => {
            const { HOSTNAME_REGEX } = await import("../settingsConfig");
            expect(HOSTNAME_REGEX.test("homeassistant")).toBe(true);
            expect(HOSTNAME_REGEX.test("my-server")).toBe(true);
            expect(HOSTNAME_REGEX.test("HOST01")).toBe(true);
            expect(HOSTNAME_REGEX.test("a")).toBe(true);
        });

        it("rejects invalid hostnames", async () => {
            const { HOSTNAME_REGEX } = await import("../settingsConfig");
            expect(HOSTNAME_REGEX.test("-bad")).toBe(false);
            expect(HOSTNAME_REGEX.test("bad-")).toBe(false);
            expect(HOSTNAME_REGEX.test("")).toBe(false);
            expect(HOSTNAME_REGEX.test("a".repeat(64))).toBe(false);
        });
    });

    describe("WORKGROUP_REGEX", () => {
        it("accepts valid workgroup names", async () => {
            const { WORKGROUP_REGEX } = await import("../settingsConfig");
            expect(WORKGROUP_REGEX.test("WORKGROUP")).toBe(true);
            expect(WORKGROUP_REGEX.test("MY-GROUP")).toBe(true);
            expect(WORKGROUP_REGEX.test("A")).toBe(true);
        });

        it("rejects invalid workgroup names", async () => {
            const { WORKGROUP_REGEX } = await import("../settingsConfig");
            expect(WORKGROUP_REGEX.test("-bad")).toBe(false);
            expect(WORKGROUP_REGEX.test("bad-")).toBe(false);
            expect(WORKGROUP_REGEX.test("")).toBe(false);
            expect(WORKGROUP_REGEX.test("a".repeat(16))).toBe(false);
        });
    });

    describe("buildSettingsTree", () => {
        it("returns an array with expected nodes", async () => {
            const { buildSettingsTree } = await import("../settingsConfig");
            const tree = buildSettingsTree();
            expect(Array.isArray(tree)).toBe(true);
            expect(tree.length).toBeGreaterThan(0);
        });

        it("includes general, network, telemetry, homeassistant, and app_configuration nodes", async () => {
            const { buildSettingsTree } = await import("../settingsConfig");
            const tree = buildSettingsTree();
            const ids = tree.flatMap((node) => [
                node.id,
                ...(node.children?.map((c) => c.id) ?? []),
            ]);
            expect(ids).toContain("general");
            expect(ids).toContain("telemetry");
            expect(ids).toContain("homeassistant");
            expect(ids).toContain("app_configuration");
        });

        it("includes network with devices and access_control children", async () => {
            const { buildSettingsTree } = await import("../settingsConfig");
            const tree = buildSettingsTree();
            const networkNode = tree.find((n) => n.id === "network");
            expect(networkNode).toBeTruthy();
            const childIds = networkNode?.children?.map((c) => c.id) ?? [];
            expect(childIds).toContain("network_devices");
            expect(childIds).toContain("network_access_control");
        });
    });
});
