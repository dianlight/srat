/* eslint-disable */
import type { ComponentProps } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import type { UserEditForm } from "../UserEditForm";

type UserEditFormProps = ComponentProps<typeof UserEditForm>;
type SubmittedUser = Parameters<UserEditFormProps["onSubmit"]>[0];

// localStorage shim for testing
if (!(globalThis as any).localStorage) {
  const _store: Record<string, string> = {};
  (globalThis as any).localStorage = {
    getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
    setItem: (k: string, v: string) => {
      _store[k] = String(v);
    },
    removeItem: (k: string) => {
      delete _store[k];
    },
    clear: () => {
      for (const k of Object.keys(_store)) delete _store[k];
    },
  };
}

describe("UserEditForm component", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    localStorage.clear();
  });

  it("renders empty form for new user", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: { username: "", password: "", doCreate: true },
        onSubmit: mockSubmit,
      }),
    );

    const usernameField = await screen.findByLabelText(/username/i);
    expect(usernameField).toBeTruthy();
  });

  it("renders form with user data for existing user", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
        },
        onSubmit: mockSubmit,
      }),
    );

    const usernameField = await screen.findByLabelText(/username/i);
    expect(usernameField).toBeTruthy();
    expect((usernameField as HTMLInputElement).value).toBe("testuser");
  });

  it("shows username readonly for existing non-admin users", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
        },
        onSubmit: mockSubmit,
      }),
    );

    const usernameField = await screen.findByLabelText(/username/i);
    expect((usernameField as HTMLInputElement).readOnly).toBe(true);
  });

  it("allows username edit for admin users", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "admin",
          password: "",
          doCreate: false,
          is_admin: true,
        },
        onSubmit: mockSubmit,
      }),
    );

    const usernameField = await screen.findByLabelText(/username/i);
    expect((usernameField as HTMLInputElement).readOnly).toBe(false);
  });

  it("renders password fields", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: { username: "", password: "", doCreate: true },
        onSubmit: mockSubmit,
      }),
    );

    // Find all text inputs (username, password, repeat password)
    const inputs = await screen.findAllByRole("textbox");
    // The username field should be present
    expect(inputs.length).toBeGreaterThanOrEqual(1);

    // Check the form contains password inputs (type=password doesn't have textbox role)
    const passwordInputs = screen.getAllByLabelText(/password/i);
    let passwordCount = 0;
    for (let i = 0; i < passwordInputs.length; i++) {
      if ((passwordInputs[i] as HTMLInputElement)?.type === "password") {
        passwordCount++;
      }
    }
    expect(passwordCount).toBe(2);
  });

  it("renders read/write and read-only share selectors", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    render(
      React.createElement(UserEditForm as any, {
        userData: { username: "", password: "", doCreate: true },
        availableShares: ["media", "backup"],
        onSubmit: () => {},
      }),
    );

    expect(await screen.findByLabelText(/read\/write shares/i)).toBeTruthy();
    expect(await screen.findByLabelText(/read-only shares/i)).toBeTruthy();
  });

  it("renders cancel button when onCancel provided", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};
    const mockCancel = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: { username: "", password: "", doCreate: true },
        onSubmit: mockSubmit,
        onCancel: mockCancel,
      }),
    );

    const cancelButton = await screen.findByText(/cancel/i);
    expect(cancelButton).toBeTruthy();
  });

  it("shows Create User button for new users", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: { username: "", password: "", doCreate: true },
        onSubmit: mockSubmit,
      }),
    );

    const createButton = await screen.findByText(/create user/i);
    expect(createButton).toBeTruthy();
  });

  it("shows Save Changes button for existing users", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
        },
        onSubmit: mockSubmit,
      }),
    );

    const saveButton = await screen.findByText(/save changes/i);
    expect(saveButton).toBeTruthy();
  });

  it("disables Save Changes button while an existing user form is pristine", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    render(
      React.createElement(UserEditForm, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
          rw_shares: ["media"],
          ro_shares: [],
        },
        availableShares: ["media"],
        onSubmit: () => {},
      }),
    );

    const saveButton = await screen.findByRole("button", {
      name: /save changes/i,
    });
    expect(saveButton).toBeDisabled();
  });

  it("displays admin account info text for admin users", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "admin",
          password: "",
          doCreate: false,
          is_admin: true,
        },
        onSubmit: mockSubmit,
      }),
    );

    const adminInfo = await screen.findByText(/administrator account/i);
    expect(adminInfo).toBeTruthy();
  });

  it("displays new user info text for new users", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: { username: "", password: "", doCreate: true },
        onSubmit: mockSubmit,
      }),
    );

    const newUserInfo = await screen.findByText(/new user/i);
    expect(newUserInfo).toBeTruthy();
  });

  it("displays regular user info text for existing non-admin users", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
        },
        onSubmit: mockSubmit,
      }),
    );

    const userInfo = await screen.findByText(/regular user/i);
    expect(userInfo).toBeTruthy();
  });

  it("disables form when disabled prop is true", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    const mockSubmit = () => {};

    render(
      React.createElement(UserEditForm as any, {
        userData: { username: "", password: "", doCreate: true },
        onSubmit: mockSubmit,
        disabled: true,
      }),
    );

    const usernameField = await screen.findByLabelText(/username/i);
    expect((usernameField as HTMLInputElement).disabled).toBe(true);
  });

  it("does not require password fields in edit mode", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
        },
        onSubmit: () => {},
      }),
    );

    const passwordField = await screen.findByLabelText(/^password$/i);
    const repeatPasswordField =
      await screen.findByLabelText(/repeat password/i);

    expect((passwordField as HTMLInputElement).required).toBe(false);
    expect((repeatPasswordField as HTMLInputElement).required).toBe(false);
  });

  it("requires matching repeat password in edit mode when password is set", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { UserEditForm } = await import("../UserEditForm");

    let submitCount = 0;

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
        },
        onSubmit: () => {
          submitCount += 1;
        },
      }),
    );

    const user = userEvent.setup();
    const passwordField = await screen.findByLabelText(/^password$/i);
    const repeatPasswordField =
      await screen.findByLabelText(/repeat password/i);

    await user.type(passwordField, "secret1");
    await user.type(repeatPasswordField, "secret2");

    const saveButton = await screen.findByRole("button", {
      name: /save changes/i,
    });
    await user.click(saveButton);

    expect(submitCount).toBe(0);
  });

  it("submits in edit mode when password and repeat password match", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { UserEditForm } = await import("../UserEditForm");

    let submitCount = 0;

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
        },
        onSubmit: () => {
          submitCount += 1;
        },
      }),
    );

    const user = userEvent.setup();
    const passwordField = await screen.findByLabelText(/^password$/i);
    const repeatPasswordField =
      await screen.findByLabelText(/repeat password/i);

    await user.type(passwordField, "secret1");
    await user.type(repeatPasswordField, "secret1");

    const saveButton = await screen.findByRole("button", {
      name: /save changes/i,
    });
    await user.click(saveButton);

    expect(submitCount).toBe(1);
  });

  it("keeps share association exclusive between rw and ro on submit", async () => {
    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { UserEditForm } = await import("../UserEditForm");

    const submitted: any[] = [];

    render(
      React.createElement(UserEditForm as any, {
        userData: {
          username: "testuser",
          password: "",
          doCreate: false,
          is_admin: false,
          rw_shares: ["media"],
          ro_shares: ["media", "archive"],
        },
        availableShares: ["media", "archive"],
        onSubmit: (data: any) => {
          submitted.push(data);
        },
      }),
    );

    const user = userEvent.setup();
    const passwordField = await screen.findByLabelText(/^password$/i);
    const repeatPasswordField =
      await screen.findByLabelText(/repeat password/i);

    await user.type(passwordField, "secret1");
    await user.type(repeatPasswordField, "secret1");

    const saveButton = await screen.findByRole("button", {
      name: /save changes/i,
    });
    await user.click(saveButton);

    await waitFor(() => expect(submitted.length).toBe(1));
    expect(submitted[0]?.rw_shares).toEqual(["media"]);
    expect(submitted[0]?.ro_shares).toEqual(["archive"]);
  });

  it("disables Create User submit button while the form is pristine", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { UserEditForm } = await import("../UserEditForm");

    render(
      React.createElement(UserEditForm, {
        userData: { username: "", password: "", doCreate: true },
        onSubmit: () => {},
      }),
    );

    const createButton = await screen.findByRole("button", {
      name: /create user/i,
    });
    expect(createButton).toBeDisabled();
  });

  it("enables Create User submit button after the user types a username", async () => {
    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { UserEditForm } = await import("../UserEditForm");

    render(
      React.createElement(UserEditForm, {
        userData: { username: "", password: "", doCreate: true },
        onSubmit: () => {},
      }),
    );

    const user = userEvent.setup();
    const createButton = await screen.findByRole("button", {
      name: /create user/i,
    });
    expect(createButton).toBeDisabled();

    const usernameField = (await screen.findByLabelText(
      /username/i,
    )) as HTMLInputElement;
    await user.type(usernameField, "newuser");

    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: /create user/i }),
      ).toBeEnabled(),
    );
  });

  it("does not submit user form and selects highlighted share on Enter in read/write shares autocomplete", async () => {
    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { UserEditForm } = await import("../UserEditForm");

    const submitted: SubmittedUser[] = [];

    render(
      React.createElement(UserEditForm, {
        userData: { username: "", password: "", doCreate: true },
        availableShares: ["media", "backup"],
        onSubmit: (data: SubmittedUser) => {
          submitted.push(data);
        },
      }),
    );

    const user = userEvent.setup();
    const usernameField = (await screen.findByLabelText(
      /username/i,
    )) as HTMLInputElement;
    await user.type(usernameField, "newuser");
    const passwordField = (await screen.findByLabelText(
      /^password/i,
    )) as HTMLInputElement;
    await user.type(passwordField, "Secret123!");
    const repeatPasswordField = (await screen.findByLabelText(
      /repeat password/i,
    )) as HTMLInputElement;
    await user.type(repeatPasswordField, "Secret123!");

    const rwSharesInput = (await screen.findByLabelText(
      /read\/write shares/i,
    )) as HTMLInputElement;
    await user.type(rwSharesInput, "media");

    const createButton = screen.getByRole("button", { name: /create user/i });
    await waitFor(() => expect(createButton).toBeEnabled());

    // Enter in the autocomplete must select the typed option, not submit.
    await user.keyboard("{Enter}");

    const createUserButton = screen.getByRole("button", {
      name: /create user/i,
    });
    await user.click(createUserButton);

    await waitFor(() => expect(submitted.length).toBe(1));
    expect(submitted[0]?.rw_shares).toEqual(["media"]);
  });

  it("does not submit user form and selects highlighted share on Enter in read-only shares autocomplete", async () => {
    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { UserEditForm } = await import("../UserEditForm");

    const submitted: SubmittedUser[] = [];

    render(
      React.createElement(UserEditForm, {
        userData: { username: "", password: "", doCreate: true },
        availableShares: ["media", "backup"],
        onSubmit: (data: SubmittedUser) => {
          submitted.push(data);
        },
      }),
    );

    const user = userEvent.setup();
    const usernameField = (await screen.findByLabelText(
      /username/i,
    )) as HTMLInputElement;
    await user.type(usernameField, "newuser");
    const passwordField = (await screen.findByLabelText(
      /^password/i,
    )) as HTMLInputElement;
    await user.type(passwordField, "Secret123!");
    const repeatPasswordField = (await screen.findByLabelText(
      /repeat password/i,
    )) as HTMLInputElement;
    await user.type(repeatPasswordField, "Secret123!");

    const roSharesInput = (await screen.findByLabelText(
      /read-only shares/i,
    )) as HTMLInputElement;
    await user.type(roSharesInput, "media");

    const createButton = screen.getByRole("button", { name: /create user/i });
    await waitFor(() => expect(createButton).toBeEnabled());

    // Enter in the read-only autocomplete must select the option, not submit.
    await user.keyboard("{Enter}");
    await user.click(screen.getByRole("button", { name: /create user/i }));

    await waitFor(() => expect(submitted.length).toBe(1));
    expect(submitted[0]?.ro_shares).toEqual(["media"]);
  });

  it("renders backend disabled share with disabled label in read/write shares autocomplete", async () => {
    const React = await import("react");
    const { render, screen, within } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { UserEditForm } = await import("../UserEditForm");

    render(
      React.createElement(UserEditForm, {
        userData: { username: "", password: "", doCreate: true },
        availableShares: ["media", "backup"],
        disabledShares: ["backup"],
        onSubmit: () => {},
      }),
    );

    const user = userEvent.setup();
    const rwSharesInput = (await screen.findByLabelText(
      /read\/write shares/i,
    )) as HTMLInputElement;
    await user.click(rwSharesInput);

    const disabledOption = await screen.findByRole("option", {
      name: /backup \(disabled\)/i,
    });
    const enabledOption = await screen.findByRole("option", {
      name: /^media$/i,
    });
    const disabledText = within(disabledOption).getByText(
      /backup \(disabled\)/i,
    ) as HTMLElement;
    const enabledText = within(enabledOption).getByText(
      /^media$/i,
    ) as HTMLElement;
    const disabledStyle = getComputedStyle(disabledText);
    const enabledStyle = getComputedStyle(enabledText);

    expect(await screen.findByText(/backup \(disabled\)/i)).toBeTruthy();
    expect(disabledStyle.color).toBe("rgba(0, 0, 0, 0.38)");
    expect(Number.parseFloat(disabledStyle.opacity)).toBeCloseTo(0.7);
    expect(enabledStyle.color).not.toBe(disabledStyle.color);
    expect(Number.parseFloat(enabledStyle.opacity || "1")).toBe(1);
  });

  it("selects a backend disabled share on Enter without submitting the form", async () => {
    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { UserEditForm } = await import("../UserEditForm");

    const submitted: SubmittedUser[] = [];

    render(
      React.createElement(UserEditForm, {
        userData: { username: "", password: "", doCreate: true },
        availableShares: ["media", "backup"],
        disabledShares: ["backup"],
        onSubmit: (data: SubmittedUser) => {
          submitted.push(data);
        },
      }),
    );

    const user = userEvent.setup();
    const usernameField = (await screen.findByLabelText(
      /username/i,
    )) as HTMLInputElement;
    await user.type(usernameField, "newuser");
    const passwordField = (await screen.findByLabelText(
      /^password/i,
    )) as HTMLInputElement;
    await user.type(passwordField, "Secret123!");
    const repeatPasswordField = (await screen.findByLabelText(
      /repeat password/i,
    )) as HTMLInputElement;
    await user.type(repeatPasswordField, "Secret123!");

    const rwSharesInput = (await screen.findByLabelText(
      /read\/write shares/i,
    )) as HTMLInputElement;
    await user.type(rwSharesInput, "backup");

    const createButton = screen.getByRole("button", { name: /create user/i });
    await waitFor(() => expect(createButton).toBeEnabled());

    // Enter must select the highlighted (muted) option, not submit the form.
    await user.keyboard("{Enter}");
    await user.click(screen.getByRole("button", { name: /create user/i }));

    await waitFor(() => expect(submitted.length).toBe(1));
    expect(submitted[0]?.rw_shares).toEqual(["backup"]);
  });
});
