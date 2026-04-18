import { getCurrentEnv } from "../../macro/Environment" with { type: "macro" };

// --- IP Address and CIDR Validation Helpers ---
// Matches IPv4 address or IPv4 CIDR (e.g., 192.168.1.1 or 192.168.1.0/24)
// Mask range /0 to /32
export const IPV4_OR_CIDR_REGEX =
  /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\/(?:[0-9]|[12][0-9]|3[0-2]))?$/;

// Comprehensive IPv6 regex (source: https://stackoverflow.com/a/17871737/796832), modified to also accept CIDR notation.
// Covers various forms like ::1, fe80::%scope, IPv4-mapped, and their CIDR versions (e.g., 2001:db8::/32).
// Mask range /0 to /128
export const IPV6_OR_CIDR_REGEX = new RegExp(
  "^(" +
    "([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|" + // 1:2:3:4:5:6:7:8
    "([0-9a-fA-F]{1,4}:){1,7}:|" + // 1::                                 1:2:3:4:5:6:7::
    "([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|" + // 1::8               1:2:3:4:5:6::8   1:2:3:4:5:6::8
    "([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|" + // 1::7:8             1:2:3:4:5::7:8   1:2:3:4:5::8
    "([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|" + // 1::6:7:8           1:2:3:4::6:7:8   1:2:3:4::8
    "([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|" + // 1::5:6:7:8         1:2:3::5:6:7:8   1:2:3::8
    "([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|" + // 1::4:5:6:7:8       1:2::4:5:6:7:8   1:2::8
    "[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|" + // 1::3:4:5:6:7:8     1::3:4:5:6:7:8   1::8
    ":((:[0-9a-fA-F]{1,4}){1,7}|:)|" + // ::2:3:4:5:6:7:8    ::2:3:4:5:6:7:8  ::8       ::
    "fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|" + // fe80::7:8%eth0     fe80::7:8%1  (link-local IPv6 addresses with zone index)
    "::(ffff(:0{1,4}){0,1}:){0,1}" +
    "((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}" +
    "(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|" + // ::255.255.255.255  ::ffff:255.255.255.255  ::ffff:0:255.255.255.255 (IPv4-mapped IPv6 addresses and IPv4-translated addresses)
    "([0-9a-fA-F]{1,4}:){1,4}:" +
    "((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}" + // 2001:db8:3:4::192.0.2.33  64:ff9b::192.0.2.33 (IPv4-Embedded IPv6 Address)
    "(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])" +
    ")(/(?:[0-9]|[1-9][0-9]|1[01][0-9]|12[0-8]))?$", // Optional CIDR mask /0 to /128
);

export function isValidIpAddressOrCidr(ip: string): boolean {
  if (typeof ip !== "string") return false;
  return IPV4_OR_CIDR_REGEX.test(ip) || IPV6_OR_CIDR_REGEX.test(ip);
}

// --- Hostname Validation Helper ---
// Allows alphanumeric characters and hyphens. Cannot start or end with a hyphen. Length 1-63.
export const HOSTNAME_REGEX = /^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/;

// --- Workgroup Validation Helper ---
// Allows alphanumeric characters and hyphens. Cannot start or end with a hyphen. Length 1-15.
export const WORKGROUP_REGEX =
  /^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,13}[a-zA-Z0-9])?$/;

// --- Settings Tree Structure ---

export interface SettingTreeNode {
  id: string;
  label: string;
  children?: SettingTreeNode[];
  settingName?: string;
}

export const categories: {
  [key: string]: { [key: string]: string[] } | string[];
} = {
  General: [
    "hostname",
    "workgroup",
    "local_master",
    "compatibility_mode",
    "allow_guest",
    "disable_smart",
  ],
  Network: {
    Devices: [
      "bind_all_interfaces",
      "interfaces",
      "multi_channel",
      "smb_over_quic",
    ],
    "Access Control": ["allow_hosts"],
  },
  //'Update': ['update_channel'],
  Telemetry: ["telemetry_mode"],
  HomeAssistant: ["export_stats_to_ha", "ha_use_nfs"],
};

export const beta_categories: {
  [key: string]: { [key: string]: string[] } | string[];
} = {
  // TODO: Enable when HDIdle feature is ready
  // 'Power ( 🚧 WIP )': ['hdidle_enabled', 'hdidle_default_idle_time', 'hdidle_default_command_type', 'hdidle_ignore_spin_down_detection'],
};

export function buildSettingsTree(): SettingTreeNode[] {
  const tree: SettingTreeNode[] = [];

  let all_categories = { ...categories };
  if (getCurrentEnv() !== "production") {
    all_categories = { ...all_categories, ...beta_categories };
  }

  Object.entries(all_categories).forEach(([category, subCategories]) => {
    if (Array.isArray(subCategories)) {
      const leafNode: SettingTreeNode = {
        id: category.toLowerCase(),
        label: category,
        settingName: category.toLowerCase(),
      };
      tree.push(leafNode);
    } else {
      const categoryNode: SettingTreeNode = {
        id: category.toLowerCase(),
        label: category,
        children: [],
      };

      Object.entries(subCategories).forEach(([subCategory, _settings]) => {
        const leafNode: SettingTreeNode = {
          id: `${category.toLowerCase()}_${subCategory.toLowerCase().replace(/\s+/g, "_")}`,
          label: subCategory,
          settingName: subCategory.toLowerCase().replace(/\s+/g, "_"),
        };
        categoryNode.children?.push(leafNode);
      });

      tree.push(categoryNode);
    }
  });

  tree.push({
    id: "app_configuration",
    label: "App Configuration",
    settingName: "app_configuration",
  });

  return tree;
}
