import type {
  Settings,
  Telemetry_mode,
  Usage,
  User,
} from "../../store/sratApi";

export interface SecurityFormData {
  hostname: string;
  workgroup: string;
  newPassword: string;
  confirmPassword: string;
}

export interface NetworkFormData {
  bind_all_interfaces: boolean;
  interfaces: string[];
}

export interface FirstShareFormData {
  partitionId: string;
  shareName: string;
  usage: Usage;
}

export interface TelemetryFormData {
  telemetry_mode: Telemetry_mode;
}

export interface WizardCollectedData {
  security?: SecurityFormData;
  network?: NetworkFormData;
  firstShare?: FirstShareFormData;
  telemetry?: TelemetryFormData;
}

export type SettingsGuard = Settings;
export type UserGuard = User;

export interface SetupWizardContentProps {
  allowSkip: boolean;
}
