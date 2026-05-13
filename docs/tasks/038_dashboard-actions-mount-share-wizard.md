<!-- DOCTOC SKIP -->

# [FEATURE]: Dashboard Actions — Mount & Share Wizard for Partitions

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _TBD_

## 🎯 Objective

When the user clicks the **"Mount"** or **"Create Share"** action button for a partition in `DashboardActions` → `ActionableItemsList`, open an inline wizard dialog that guides them through mounting and sharing that specific partition — without navigating away from the Dashboard.

The wizard reuses the existing `SetupWizard` / `FirstShareStepContent` infrastructure but is streamlined to a **single step** (FirstShare) plus a minimal confirmation, with the target partition pre-selected and locked.

> _Context for Copilot: The current flow navigates to the Volumes or Shares tab via `useNavigate`. The new flow should instead open a `MountShareWizard` dialog component. The wizard reuses `FirstShareStepContent` (already handles partition selection + share name + usage), `SetupWizardActions`, and the same `usePostApiVolumeMountMutation` + `usePostApiShareMutation` RTK Query hooks used in `SetupWizard`. The "enable-share" action can continue navigating to the Shares tab as today (it targets an existing share, not a new mount)._

## 🛠️ Technical Specifications

- **Inputs:**
  - `partition: Partition` — the target partition from `ActionableItemsList`
  - `action: "mount" | "share"` — which operation to perform (enables the wizard)
- **Outputs:**
  - New `frontend/src/pages/dashboard/components/MountShareWizard.tsx` — focused dialog wizard
  - Updated `ActionableItemsList.tsx` — "Mount" and "Create Share" buttons open `MountShareWizard` instead of navigating
- **Dependencies:**
  - `frontend/src/components/wizard/steps/FirstShareStepContent.tsx`
  - `frontend/src/components/wizard/SetupWizardActions.tsx`
  - `frontend/src/components/wizard/utils.ts` (`getWizardAvailablePartitions`, `WizardPartitionOption`, `sanitizeWizardShareName`)
  - `frontend/src/components/wizard/types.ts` (`FirstShareFormData`)
  - `frontend/src/store/sratApi.ts` — `usePostApiVolumeMountMutation`, `usePostApiShareMutation`, `useGetApiVolumesQuery`
  - `react-hook-form` + `react-hook-form-mui` (`useForm`, `FormContainer`)

## 📝 Task List

- [ ] Task 1: Create `frontend/src/pages/dashboard/components/MountShareWizard.tsx` — a single-step Dialog wizard that renders `FirstShareStepContent` with the target partition pre-selected
- [ ] Task 2: In `MountShareWizard`, implement submit logic: call `usePostApiVolumeMountMutation` when not yet mounted, then `usePostApiShareMutation`; on success call `onClose()`
- [ ] Task 3: Update `ActionableItemsList.tsx` — replace `handleMount` and `handleCreateShare` navigate calls with state to open `MountShareWizard`; pass the partition and action as props
- [ ] Task 4: Ensure the partition autocomplete in `FirstShareStepContent` is pre-filled and locked (disabled) when a specific partition is passed in
- [ ] Task 5: Handle the `"share"` action (partition already mounted): skip the mount call, go straight to `usePostApiShareMutation`
- [ ] Task 6: Unit tests for `MountShareWizard` (open/close, submit success, submit error) in `frontend/src/pages/dashboard/components/__tests__/MountShareWizard.test.tsx`
- [ ] Task 7: Unit tests for updated `ActionableItemsList` (buttons open wizard, not navigate) in existing `__tests__/` folder
- [ ] Task 8: Run `mise run //frontend:lint` and `bun tsgo --noEmit`; fix any type or lint issues
- [ ] Task 9: Run `mise run //frontend:test --rerun-each 10` for all modified/new test files to confirm stability
- [ ] Task 10: Capture lessons learned and update documentation
- [ ] Task 11: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

### MountShareWizard component structure

```tsx
// frontend/src/pages/dashboard/components/MountShareWizard.tsx
interface MountShareWizardProps {
  open: boolean;
  onClose: () => void;
  partition: Partition;           // pre-selected partition
  action: "mount" | "share";     // "mount" = mount+share, "share" = share only
}

export function MountShareWizard({ open, onClose, partition, action }: MountShareWizardProps) {
  const formContext = useForm<FirstShareFormData>({ defaultValues: {...} });
  const [mountVolume] = usePostApiVolumeMountMutation();
  const [createShare] = usePostApiShareMutation();

  const handleSubmit = async (data: FirstShareFormData) => {
    // if action === "mount": call mountVolume first, then createShare
    // if action === "share": call createShare only (already mounted)
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose}>
      <FormContainer formContext={formContext} onSuccess={handleSubmit}>
        <DialogTitle>Mount & Share Partition</DialogTitle>
        <FirstShareStepContent
          availablePartitions={...}          // only the target partition, locked
          hasAvailablePartitions={true}
          isVolumesLoading={false}
          selectedPartitionId={partition.id}
        />
        <DialogActions>
          <Button onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="contained">
            {formContext.formState.isSubmitting ? "Saving..." : "Mount & Share"}
          </Button>
        </DialogActions>
      </FormContainer>
    </Dialog>
  );
}
```

### Pre-selecting and locking the partition

`FirstShareStepContent` renders an `AutocompleteElement` for `partitionId`. To pre-select and lock it:
- On wizard open, call `formContext.setValue("partitionId", partition.id)` in a `useEffect`.
- Pass only the target partition in `availablePartitions` so the autocomplete has a single option.
- Set `autocompleteProps={{ disabled: true }}` to prevent changing the selection.
- Also pre-populate `shareName` using `sanitizeWizardShareName(partition.name)` as a suggested default.

### "share" action (already mounted)

When `action === "share"`, the partition is already mounted (has a `mount_point_data` entry). Skip `mountVolume` and call `createShare` directly using the existing mount point path.

### ActionableItemsList changes

Replace navigate calls for "mount" and "share" with a `useState`:

```tsx
const [wizardState, setWizardState] = useState<{
  partition: Partition;
  action: "mount" | "share";
} | null>(null);

const handleMount = (partition: Partition) => {
  if (disabled) return;
  setWizardState({ partition, action: "mount" });
};

const handleCreateShare = (partition: Partition) => {
  if (disabled) return;
  setWizardState({ partition, action: "share" });
};

// in JSX:
{wizardState && (
  <MountShareWizard
    open={true}
    onClose={() => setWizardState(null)}
    partition={wizardState.partition}
    action={wizardState.action}
  />
)}
```

The `"enable-share"` action (navigate to Shares tab) is **not changed** — it targets an existing share that just needs to be re-enabled.

### Guard-Before-Hooks Rule

`MountShareWizard` uses RTK Query hooks. Apply the guard-before-hooks pattern: if `!open` guard is needed, split into a thin wrapper + inner component so hooks only fire when the dialog is actually open.

### Form lifecycle

Follow `react-hook-form-mui` canonical patterns (see `.github/instructions/react-hook-form-mui.instructions.md`):
- `<FormContainer formContext={formContext} onSuccess={handleSubmit}>` wraps both content and actions.
- `type="submit"` on the action button — never a manual `onClick` handler.
- API errors → `setError("root", { message: "..." })`; display via `formState.errors.root?.message`.
- `formState.isSubmitting` for the loading state — no manual `useState`.

## 🔗 Code References & TODOs

- [ ] `frontend/src/pages/dashboard/components/ActionableItemsList.tsx` — `handleMount` and `handleCreateShare` to be replaced with wizard open
- [ ] `frontend/src/components/wizard/steps/FirstShareStepContent.tsx` — consumed as-is; verify `autocompleteProps.disabled` propagation
- [ ] `frontend/src/components/wizard/utils.ts` — reuse `sanitizeWizardShareName` for default share name
- [ ] `frontend/src/components/wizard/types.ts` — reuse `FirstShareFormData`
