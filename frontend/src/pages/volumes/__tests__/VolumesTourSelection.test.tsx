import { describe, expect, it } from "vitest";

describe("Volumes tour selection", () => {
    it("prefers the first visible partition for guided tour selection", async () => {
        const { getTourVolumeSelection } = await import("../Volumes");

        const selection = getTourVolumeSelection([
            {
                id: "disk1",
                name: "sda",
                partitions: {
                    system: { id: "p1", name: "hassos-data", system: true },
                    data: { id: "p2", name: "sda2", system: false },
                },
            } as any,
        ], true);

        expect(selection?.partition?.name).toBe("sda2");
    });

    it("falls back to the first disk when no partitions are visible", async () => {
        const { getTourVolumeSelection } = await import("../Volumes");

        const selection = getTourVolumeSelection([
            {
                id: "disk1",
                name: "sda",
                partitions: {
                    system: { id: "p1", name: "hassos-data", system: true },
                },
            } as any,
        ], true);

        expect(selection?.disk.id).toBe("disk1");
        expect(selection?.partition).toBeUndefined();
    });
});
