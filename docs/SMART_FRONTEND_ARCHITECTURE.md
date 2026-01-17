<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [SMART Frontend Component Architecture](#smart-frontend-component-architecture)
  - [Component Hierarchy](#component-hierarchy)
  - [Data Flow](#data-flow)
  - [State Management Flow](#state-management-flow)
  - [API Integration Points](#api-integration-points)
    - [Current State (Placeholder)](#current-state-placeholder)
    - [Future State (After Backend Implementation)](#future-state-after-backend-implementation)
  - [Type Definitions](#type-definitions)
    - [SmartInfo (from Backend)](#smartinfo-from-backend)
    - [SmartHealthStatus (Local)](#smarthealthstatus-local)
    - [SmartTestStatus (Local)](#smartteststatus-local)
    - [SmartTestType](#smarttesttype)
  - [User Interactions](#user-interactions)
    - [View SMART Information](#view-smart-information)
    - [Start Self-Test](#start-self-test)
    - [Abort Self-Test](#abort-self-test)
    - [Enable/Disable SMART](#enabledisable-smart)
  - [Disabled State Logic](#disabled-state-logic)
  - [Error Handling](#error-handling)
  - [Testing Architecture](#testing-architecture)
  - [Future Enhancements Diagram](#future-enhancements-diagram)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# SMART Frontend Component Architecture

## Component Hierarchy

```txt
Volumes Page
└── VolumeDetailsPanel
    ├── Disk Information Card
    ├── SmartStatusPanel ← NEW
    │   ├── Temperature Section
    │   ├── Power Statistics Section
    │   ├── Health Status Section
    │   │   └── Failing Attributes List
    │   ├── Self-Test Status Section
    │   │   └── Progress Bar (if running)
    │   ├── Control Actions Section
    │   │   ├── Start Test Button
    │   │   ├── Abort Test Button
    │   │   ├── Enable SMART Button
    │   │   └── Disable SMART Button
    │   └── Test Type Selection Dialog
    ├── Partition Information Card
    └── Share Information Cards
```

## Data Flow

```txt
┌─────────────────────────────────────────────────────────────────┐
│                     VolumeDetailsPanel                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Props:                                                           │
│  ├─ disk: Disk                                                   │
│  │  └─ smart_info: SmartInfo?                                   │
│  ├─ partition: Partition                                        │
│  └─ share: SharedResource?                                      │
│                                                                   │
│  Hooks:                                                           │
│  └─ useSmartOperations(diskId)                                  │
│     └─ Returns: {                                               │
│        startSelfTest,                                           │
│        abortSelfTest,                                           │
│        enableSmart,                                             │
│        disableSmart,                                            │
│        isLoading                                                │
│     }                                                            │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
           │
           │ Passes props & callbacks
           ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SmartStatusPanel                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Props:                                                           │
│  ├─ smartInfo: SmartInfo?                                       │
│  ├─ diskDevicePath?: string                                     │
│  ├─ healthStatus?: SmartHealthStatus                            │
│  ├─ testStatus?: SmartTestStatus                                │
│  ├─ isSmartSupported: boolean                                   │
│  ├─ isReadOnlyMode: boolean                                     │
│  ├─ onEnableSmart?: () => void                                  │
│  ├─ onDisableSmart?: () => void                                 │
│  ├─ onStartTest?: (testType: SmartTestType) => void             │
│  ├─ onAbortTest?: () => void                                    │
│  └─ isLoading?: boolean                                         │
│                                                                   │
│  State:                                                           │
│  ├─ smartExpanded: boolean                                      │
│  ├─ showStartTestDialog: boolean                                │
│  └─ selectedTestType: SmartTestType                             │
│                                                                   │
│  Renders:                                                         │
│  ├─ Collapsible Card with Health Status Icon                    │
│  ├─ Temperature Display                                          │
│  ├─ Power Statistics (Hours, Cycles)                            │
│  ├─ Health Check Status                                          │
│  ├─ Self-Test Status & Progress                                 │
│  ├─ Control Action Buttons                                      │
│  └─ Test Type Selection Dialog                                  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## State Management Flow

```txt
User Action (Click Button)
        │
        ▼
Component Handler Calls Hook Callback
(for example, onStartTest("short"))
        │
        ▼
useSmartOperations Hook
├─ Set isLoading = true
├─ Show Toast: "Starting test..."
├─ Simulate/Call API
│  └─ TODO: Replace with RTK Query when backend ready
├─ Set isLoading = false
└─ Show Result Toast (success/error)
        │
        ▼
Component Re-renders with:
├─ Updated isLoading state
└─ New data from API (once integrated)
```

## API Integration Points

### Current State (Placeholder)

```txt
useSmartOperations Hook
├─ startSelfTest()
│  └─ console.log + toast (simulated)
├─ abortSelfTest()
│  └─ console.log + toast (simulated)
├─ enableSmart()
│  └─ console.log + toast (simulated)
└─ disableSmart()
   └─ console.log + toast (simulated)
```

### Future State (After Backend Implementation)

```txt
useSmartOperations Hook
├─ startSelfTest()
│  └─ RTK Query: usePostDiskSmartTestStartMutation()
├─ abortSelfTest()
│  └─ RTK Query: usePostDiskSmartTestAbortMutation()
├─ enableSmart()
│  └─ RTK Query: usePostDiskSmartEnableMutation()
└─ disableSmart()
   └─ RTK Query: usePostDiskSmartDisableMutation()
```

## Type Definitions

### SmartInfo (from Backend)

```typescript
interface SmartInfo {
  disk_type: "SATA" | "NVMe" | "SCSI" | "Unknown";
  temperature: SmartTempValue;
  power_on_hours: SmartRangeValue;
  power_cycle_count: SmartRangeValue;
  others?: Record<string, SmartRangeValue>;
}

interface SmartTempValue {
  value: number;
  min?: number;
  max?: number;
  overtemp_counter?: number;
}

interface SmartRangeValue {
  value: number;
  code?: number;
  min?: number;
  worst?: number;
  thresholds?: number;
}
```

### SmartHealthStatus (Local)

```typescript
interface SmartHealthStatus {
  passed: boolean;
  failing_attributes?: string[];
  overall_status: "healthy" | "warning" | "failing";
}
```

### SmartTestStatus (Local)

```typescript
interface SmartTestStatus {
  status: "idle" | "running" | "completed" | "failed";
  test_type?: string;
  percent_complete?: number;
  lba_of_first_error?: string;
}
```

### SmartTestType

```typescript
type SmartTestType = "short" | "long" | "conveyance";
```

## User Interactions

### View SMART Information

```txt
User opens Volumes page
    ↓
User selects disk with SMART support
    ↓
VolumeDetailsPanel loads disk data
    ↓
SmartStatusPanel displays:
    - Temperature gauge
    - Power-on hours
    - Health status
    - Current test status (if any)
```

### Start Self-Test

```txt
User clicks "Start Test" button
    ↓
Test Type Dialog opens
    ├─ Short (2-5 min) - Default selected
    ├─ Long (hours)
    └─ Conveyance (minutes)
    ↓
User selects type and confirms
    ↓
Hook: onStartTest(selectedType)
    ↓
Button disabled while loading
    ↓
Toast: "Test started" or error message
    ↓
"Abort Test" button becomes enabled
    ↓
Progress bar shows test progress
```

### Abort Self-Test

```txt
User clicks "Abort Test" button
    ↓
Hook: onAbortTest()
    ↓
Button disabled while loading
    ↓
Toast: "Test aborted" or error message
    ↓
Test status returns to "idle"
    ↓
"Abort Test" button becomes disabled
```

### Enable/Disable SMART

```txt
User clicks "Enable SMART" or "Disable SMART"
    ↓
Hook: onEnableSmart() or onDisableSmart()
    ↓
Button disabled while loading
    ↓
Toast: Success or error message
    ↓
UI updates to reflect new SMART state
```

## Disabled State Logic

```txt
┌─ Is smartInfo undefined?
│  └─ YES → Component returns null (not rendered)
│
├─ Is SMART not supported? OR Is read-only mode?
│  └─ YES → All buttons disabled
│
├─ For "Start Test" button:
│  └─ Is test already running? → YES → Disabled
│
├─ For "Abort Test" button:
│  └─ Is test NOT running? → YES → Disabled
│
└─ General:
   └─ Is operation loading? → YES → All buttons disabled
```

## Error Handling

```txt
User Action
    ↓
Try/Catch in Hook
    ├─ Success
    │  └─ Toast: Operation succeeded
    │
    └─ Error
       ├─ console.error() for debugging
       ├─ Toast: "Failed to [operation]"
       └─ User can retry
```

## Testing Architecture

```txt
SmartStatusPanel.test.tsx
├─ Test Suite: SmartStatusPanel Component
├─ Setup
│  └─ localStorage shim
│  └─ Test setup import
│
├─ Test Cases (10 total)
│  ├─ Render tests
│  │  ├─ Returns null when smartInfo undefined
│  │  ├─ Renders with complete SMART data
│  │  └─ Displays temperature range
│  │
│  ├─ Display tests
│  │  ├─ Shows healthy status
│  │  ├─ Shows failing attributes
│  │  ├─ Shows test status & progress
│  │  └─ Shows test error info
│  │
│  ├─ Interaction tests
│  │  ├─ Start test dialog opens
│  │  ├─ Buttons call handlers
│  │  └─ Proper disabled states
│  │
│  └─ State tests
│     ├─ Disabled in read-only mode
│     ├─ Disabled when SMART unsupported
│     ├─ Disabled when test running
│     └─ Disabled during operations
│
└─ Coverage: 97.84% line coverage
```

## Future Enhancements Diagram

```txt
Current Implementation
    │
    ├─→ Real-time Updates (SSE/WebSocket)
    │   └─ Auto-refresh SMART data
    │
    ├─→ Historical Tracking
    │   └─ Graph of SMART metrics over time
    │
    ├─→ Predictive Failures
    │   └─ Machine learning based alerts
    │
    ├─→ Advanced Tests
    │   └─ Vendor-specific test options
    │
    └─→ Multi-Language Support
        └─ i18n translations
```
