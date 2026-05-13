import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";
import { getMswServer, renderWithTestStore } from "/test/testing";

/** Minimal Partition fixture without a mount point (for "mount" action). */
const unmountedPartition = {
  id: "part-uuid-1",
  name: "DataDisk",
  legacy_device_name: "sdb1",
  device_path: "/dev/sdb1",
  mount_status: false,
  system: false,
};

/** Minimal Partition fixture already mounted (for "share" action). */
const mountedPartition = {
  id: "part-uuid-2",
  name: "MediaDisk",
  legacy_device_name: "sdc1",
  device_path: "/dev/sdc1",
  mount_status: true,
  system: false,
  mount_point_data: {
    "/mnt/mediadisk": {
      device_id: "part-uuid-2",
      path: "/mnt/mediadisk",
      root: "/",
    },
  },
};

describe("MountShareWizard", () => {
  it("renders nothing when open=false", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { MountShareWizard } = await import("../MountShareWizard");

    await renderWithTestStore(
      React.createElement(MountShareWizard as any, {
        open: false,
        onClose: () => {},
        partition: unmountedPartition,
        action: "mount",
      }),
    );

    // Dialog title should not appear when closed
    expect(screen.queryByText("Mount & Share Partition")).toBeNull();
  });

  it("renders Mount & Share dialog for 'mount' action", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { MountShareWizard } = await import("../MountShareWizard");

    await renderWithTestStore(
      React.createElement(MountShareWizard as any, {
        open: true,
        onClose: () => {},
        partition: unmountedPartition,
        action: "mount",
      }),
    );

    expect(await screen.findByText("Mount & Share Partition")).toBeTruthy();
    // Submit button label
    expect(await screen.findByRole("button", { name: /mount & share/i })).toBeTruthy();
    // Cancel button
    expect(await screen.findByRole("button", { name: /cancel/i })).toBeTruthy();
    // Share name field pre-filled with sanitized partition name
    const shareNameInput = screen.getByLabelText(/share name/i);
    expect((shareNameInput as HTMLInputElement).value).toBeTruthy();
  });

  it("renders Create Share dialog for 'share' action", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const { MountShareWizard } = await import("../MountShareWizard");

    await renderWithTestStore(
      React.createElement(MountShareWizard as any, {
        open: true,
        onClose: () => {},
        partition: mountedPartition,
        action: "share",
      }),
    );

    expect(await screen.findByRole("dialog", { name: /create share/i })).toBeTruthy();
    expect(await screen.findByRole("button", { name: /create share/i })).toBeTruthy();
  });

  it("calls mount and share API on submit for 'mount' action then closes", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { MountShareWizard } = await import("../MountShareWizard");

    const server = await getMswServer();
    let mountCalled = false;
    let shareCalled = false;

    server.use(
      http.post("http://localhost:3000/api/volume/mount", async () => {
        mountCalled = true;
        return HttpResponse.json(
          { device_id: "part-uuid-1", path: "/mnt/datadisk" },
          { status: 200 },
        );
      }),
      http.post("http://localhost:3000/api/share", async () => {
        shareCalled = true;
        return HttpResponse.json({ name: "DataDisk" }, { status: 200 });
      }),
    );

    let closed = false;
    const user = userEvent.setup();

    await renderWithTestStore(
      React.createElement(MountShareWizard as any, {
        open: true,
        onClose: () => { closed = true; },
        partition: unmountedPartition,
        action: "mount",
      }),
    );

    const submitBtn = await screen.findByRole("button", { name: /mount & share/i });
    await user.click(submitBtn);

    // Wait for async submit to complete
    await screen.findByRole("button", { name: /mount & share/i });

    expect(mountCalled).toBe(true);
    expect(shareCalled).toBe(true);
    expect(closed).toBe(true);
  });

  it("calls only share API on submit for 'share' action", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { MountShareWizard } = await import("../MountShareWizard");

    const server = await getMswServer();
    let mountCalled = false;
    let shareCalled = false;

    server.use(
      http.post("http://localhost:3000/api/volume/mount", async () => {
        mountCalled = true;
        return HttpResponse.json({}, { status: 200 });
      }),
      http.post("http://localhost:3000/api/share", async () => {
        shareCalled = true;
        return HttpResponse.json({ name: "MediaDisk" }, { status: 200 });
      }),
    );

    let closed = false;
    const user = userEvent.setup();

    await renderWithTestStore(
      React.createElement(MountShareWizard as any, {
        open: true,
        onClose: () => { closed = true; },
        partition: mountedPartition,
        action: "share",
      }),
    );

    const submitBtn = await screen.findByRole("button", { name: /create share/i });
    await user.click(submitBtn);

    await screen.findByRole("button", { name: /create share/i });

    expect(mountCalled).toBe(false);
    expect(shareCalled).toBe(true);
    expect(closed).toBe(true);
  });

  it("shows error alert when API returns an error", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { MountShareWizard } = await import("../MountShareWizard");

    const server = await getMswServer();
    server.use(
      http.post("http://localhost:3000/api/volume/mount", async () => {
        return HttpResponse.json({ error: "disk busy" }, { status: 500 });
      }),
    );

    const user = userEvent.setup();

    await renderWithTestStore(
      React.createElement(MountShareWizard as any, {
        open: true,
        onClose: () => {},
        partition: unmountedPartition,
        action: "mount",
      }),
    );

    const submitBtn = await screen.findByRole("button", { name: /mount & share/i });
    await user.click(submitBtn);

    expect(
      await screen.findByText(/failed to complete the operation/i),
    ).toBeTruthy();
  });

  it("closes dialog when Cancel is clicked", async () => {
    const React = await import("react");
    const { screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { MountShareWizard } = await import("../MountShareWizard");

    let closed = false;
    const user = userEvent.setup();

    await renderWithTestStore(
      React.createElement(MountShareWizard as any, {
        open: true,
        onClose: () => { closed = true; },
        partition: unmountedPartition,
        action: "mount",
      }),
    );

    const cancelBtn = await screen.findByRole("button", { name: /cancel/i });
    await user.click(cancelBtn);
    expect(closed).toBe(true);
  });
});
