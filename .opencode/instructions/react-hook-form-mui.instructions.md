<!-- DOCTOC SKIP -->

---

description: 'Mandatory form handling standard using react-hook-form and react-hook-form-mui'
applyTo: '**/frontend/**/*.{tsx,jsx}'

---

# react-hook-form + react-hook-form-mui — Canonical Patterns

`react-hook-form` + `react-hook-form-mui` is the **mandatory standard** for all user input in this project. This applies to every dialog, modal, and page form. Never manage form field values, validation errors, or submit-loading state with raw `useState`.

## Core Rules

1. **Always use `FormContainer`** from `react-hook-form-mui` as the form root. Pass `formContext` (from `useForm`) and `onSuccess` (the async submit handler that receives validated data).
2. **Never use `<form onSubmit>` directly** — use `<FormContainer onSuccess>` instead.
3. **All submit buttons must be `type="submit"`** — never wire `onClick` to a manual submit handler.
4. **`formState.isSubmitting`** is automatically `true` while `onSuccess` runs. Do not add a manual `isSubmitting` `useState`.
5. **API errors → `setError("root", { message })`** — render via `formState.errors.root?.message`. Do not add a manual error `useState`.

## Dialog Pattern (MUI `Dialog`)

In a MUI `Dialog`, place `<FormContainer>` so it wraps **both** `DialogContent` and `DialogActions`. This ensures the `type="submit"` button inside `DialogActions` correctly triggers form validation and submission.

```tsx
const formContext = useForm<MyFormData>({ defaultValues: { ... } });
const { setError, formState: { isSubmitting } } = formContext;

const handleSubmit = async (data: MyFormData) => {
  try {
    await myMutation({ ... }).unwrap();
    onClose();
  } catch {
    setError("root", { message: "Failed to save. Please try again." });
  }
};

return (
  <Dialog open={open} onClose={() => {}} disableEscapeKeyDown>
    <FormContainer formContext={formContext} onSuccess={handleSubmit}>
      <DialogTitle>...</DialogTitle>
      <DialogContent>
        {/* form fields here */}
        {formContext.formState.errors.root && (
          <Alert severity="error">{formContext.formState.errors.root.message}</Alert>
        )}
      </DialogContent>
      <DialogActions>
        <Button type="submit" variant="contained" disabled={isSubmitting}>
          {isSubmitting ? "Saving..." : "Save"}
        </Button>
      </DialogActions>
    </FormContainer>
  </Dialog>
);
```

## Password Fields

Use `PasswordElement` from `react-hook-form-mui` — it includes a built-in show/hide toggle. **Never** re-implement with `useState` + `VisibilityIcon`/`VisibilityOffIcon`.

```tsx
<PasswordElement
  name="password"
  label="Password"
  autoComplete="new-password"
  fullWidth
  rules={{
    required: "Password is required",
    minLength: { value: 6, message: "At least 6 characters" },
    validate: (value) => value !== "changeme!" || "Cannot use the default password",
  }}
/>
```

## Cross-Field Validation (e.g., Confirm Password)

Use the second `formValues` argument in `rules.validate`:

```tsx
<PasswordElement
  name="confirmPassword"
  label="Confirm Password"
  autoComplete="new-password"
  fullWidth
  rules={{
    required: "Please confirm your password",
    validate: (value, formValues) =>
      value === formValues.password || "Passwords do not match",
  }}
/>
```

## Populating Form from Async Data

Use `setValue` inside a `useEffect` when RTK Query data arrives. Do not set `defaultValues` from an async source directly.

```tsx
useEffect(() => {
  if (settings) {
    setValue("hostname", settings.hostname ?? "");
    setValue("workgroup", settings.workgroup ?? "");
  }
}, [settings, setValue]);
```

## Available Input Elements

Use the appropriate `react-hook-form-mui` element for each input type:

| Input type | Element |
|---|---|
| Text / number | `TextFieldElement` |
| Password | `PasswordElement` |
| Select / dropdown | `SelectElement` |
| Autocomplete / multi-select | `AutocompleteElement` |
| Toggle / boolean | `SwitchElement` |
| Checkbox | `CheckboxElement` |
| Textarea | `TextareaAutosizeElement` |
| Custom/complex | `Controller` (last resort) |

## What NOT to Do

- ❌ `useState` for field values — use `useForm` fields
- ❌ `useState` for validation errors — use `rules` and `setError("root", ...)`
- ❌ `useState` for submit loading — use `formState.isSubmitting`
- ❌ `useState` for password show/hide — use `PasswordElement`
- ❌ `<form onSubmit={handleSubmit(...)}>` — use `<FormContainer onSuccess={...}>`
- ❌ `onClick` on submit button calling async logic — use `type="submit"` + `onSuccess`
- ❌ Placing `<FormContainer>` only around fields while leaving the submit button outside it in a Dialog

## Reference Implementations

- Dialog with password + text fields: `frontend/src/components/BaseConfigModal.tsx`
- Full page form with autocomplete + password: `frontend/src/pages/users/components/UserEditForm.tsx`
- Dialog with complex form: `frontend/src/pages/volumes/components/VolumeMountDialog.tsx`
- Dialog with custom content and select: `frontend/src/components/ReportIssueDialog.tsx`
