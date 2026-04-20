import { beforeEach, describe, expect, it } from "bun:test";
import "../../../test/setup";

describe("useSmartTestStatus hook", () => {
    beforeEach(() => {
        // noop
    });

    it("initializes with disk id and default status", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useSmartTestStatus } = await import("../smartTestStatusHook");

        const store = await createTestStore();
        const wrapper = ({ children }: React.PropsWithChildren) =>
            React.createElement(Provider, { store, children });

        const diskId = "disk-1";
        const { result } = renderHook(() => useSmartTestStatus(diskId), { wrapper });

        expect(result.current.smartTestStatus).toBeTruthy();
        expect(result.current.smartTestStatus.disk_id).toBe(diskId);
        expect(result.current.smartTestStatus.running).toBe(false);
        expect(result.current.smartTestStatus.percent_complete).toBe(0);
        expect(typeof result.current.isLoading).toBe("boolean");
        expect("error" in result.current).toBe(true);
    });
});
