import { emptySplitApi as api } from "./emptyApi";
export const addTagTypes = [
  "system",
  "samba",
  "share",
  "user",
  "volume",
] as const;
const injectedRtkApi = api
  .enhanceEndpoints({
    addTagTypes,
  })
  .injectEndpoints({
    endpoints: (build) => ({
      getFilesystems: build.query<
        GetFilesystemsApiResponse,
        GetFilesystemsApiArg
      >({
        query: () => ({ url: `/filesystems` }),
        providesTags: ["system"],
      }),
      getHealth: build.query<GetHealthApiResponse, GetHealthApiArg>({
        query: () => ({ url: `/health` }),
        providesTags: ["system"],
      }),
      getHostname: build.query<GetHostnameApiResponse, GetHostnameApiArg>({
        query: () => ({ url: `/hostname` }),
        providesTags: ["system"],
      }),
      getNics: build.query<GetNicsApiResponse, GetNicsApiArg>({
        query: () => ({ url: `/nics` }),
        providesTags: ["system"],
      }),
      putRestart: build.mutation<PutRestartApiResponse, PutRestartApiArg>({
        query: () => ({ url: `/restart`, method: "PUT" }),
        invalidatesTags: ["system"],
      }),
      putSambaApply: build.mutation<
        PutSambaApplyApiResponse,
        PutSambaApplyApiArg
      >({
        query: () => ({ url: `/samba/apply`, method: "PUT" }),
        invalidatesTags: ["samba"],
      }),
      getSambaConfig: build.query<
        GetSambaConfigApiResponse,
        GetSambaConfigApiArg
      >({
        query: () => ({ url: `/samba/config` }),
        providesTags: ["samba"],
      }),
      getSettings: build.query<GetSettingsApiResponse, GetSettingsApiArg>({
        query: () => ({ url: `/settings` }),
        providesTags: ["system"],
      }),
      patchSettings: build.mutation<
        PatchSettingsApiResponse,
        PatchSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/settings`,
          method: "PATCH",
          body: queryArg.body,
        }),
        invalidatesTags: ["system"],
      }),
      putSettings: build.mutation<PutSettingsApiResponse, PutSettingsApiArg>({
        query: (queryArg) => ({
          url: `/settings`,
          method: "PUT",
          body: queryArg.settings,
        }),
        invalidatesTags: ["system"],
      }),
      postShare: build.mutation<PostShareApiResponse, PostShareApiArg>({
        query: (queryArg) => ({
          url: `/share`,
          method: "POST",
          body: queryArg.sharedResource,
        }),
        invalidatesTags: ["share"],
      }),
      deleteShareByShareName: build.mutation<
        DeleteShareByShareNameApiResponse,
        DeleteShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/share/${queryArg.shareName}`,
          method: "DELETE",
        }),
        invalidatesTags: ["share"],
      }),
      getShareByShareName: build.query<
        GetShareByShareNameApiResponse,
        GetShareByShareNameApiArg
      >({
        query: (queryArg) => ({ url: `/share/${queryArg.shareName}` }),
        providesTags: ["share"],
      }),
      patchShareByShareName: build.mutation<
        PatchShareByShareNameApiResponse,
        PatchShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/share/${queryArg.shareName}`,
          method: "PATCH",
          body: queryArg.body,
        }),
        invalidatesTags: ["share"],
      }),
      putShareByShareName: build.mutation<
        PutShareByShareNameApiResponse,
        PutShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/share/${queryArg.shareName}`,
          method: "PUT",
          body: queryArg.sharedResource,
        }),
        invalidatesTags: ["share"],
      }),
      putShareByShareNameDisable: build.mutation<
        PutShareByShareNameDisableApiResponse,
        PutShareByShareNameDisableApiArg
      >({
        query: (queryArg) => ({
          url: `/share/${queryArg.shareName}/disable`,
          method: "PUT",
        }),
        invalidatesTags: ["share"],
      }),
      putShareByShareNameEnable: build.mutation<
        PutShareByShareNameEnableApiResponse,
        PutShareByShareNameEnableApiArg
      >({
        query: (queryArg) => ({
          url: `/share/${queryArg.shareName}/enable`,
          method: "PUT",
        }),
        invalidatesTags: ["share"],
      }),
      getShares: build.query<GetSharesApiResponse, GetSharesApiArg>({
        query: () => ({ url: `/shares` }),
        providesTags: ["share"],
      }),
      sse: build.query<SseApiResponse, SseApiArg>({
        query: () => ({ url: `/sse` }),
        providesTags: ["system"],
      }),
      getUpdate: build.query<GetUpdateApiResponse, GetUpdateApiArg>({
        query: () => ({ url: `/update` }),
        providesTags: ["system"],
      }),
      putUpdate: build.mutation<PutUpdateApiResponse, PutUpdateApiArg>({
        query: () => ({ url: `/update`, method: "PUT" }),
        invalidatesTags: ["system"],
      }),
      getUpdateChannels: build.query<
        GetUpdateChannelsApiResponse,
        GetUpdateChannelsApiArg
      >({
        query: () => ({ url: `/update_channels` }),
        providesTags: ["system"],
      }),
      postUser: build.mutation<PostUserApiResponse, PostUserApiArg>({
        query: (queryArg) => ({
          url: `/user`,
          method: "POST",
          body: queryArg.user,
        }),
        invalidatesTags: ["user"],
      }),
      deleteUserByUsername: build.mutation<
        DeleteUserByUsernameApiResponse,
        DeleteUserByUsernameApiArg
      >({
        query: (queryArg) => ({
          url: `/user/${queryArg.username}`,
          method: "DELETE",
        }),
        invalidatesTags: ["user"],
      }),
      putUserByUsername: build.mutation<
        PutUserByUsernameApiResponse,
        PutUserByUsernameApiArg
      >({
        query: (queryArg) => ({
          url: `/user/${queryArg.username}`,
          method: "PUT",
          body: queryArg.user,
        }),
        invalidatesTags: ["user"],
      }),
      putUseradmin: build.mutation<PutUseradminApiResponse, PutUseradminApiArg>(
        {
          query: (queryArg) => ({
            url: `/useradmin`,
            method: "PUT",
            body: queryArg.user,
          }),
          invalidatesTags: ["user"],
        },
      ),
      getUsers: build.query<GetUsersApiResponse, GetUsersApiArg>({
        query: () => ({ url: `/users` }),
        providesTags: ["user"],
      }),
      postVolumeDiskByDiskIdEject: build.mutation<
        PostVolumeDiskByDiskIdEjectApiResponse,
        PostVolumeDiskByDiskIdEjectApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/disk/${queryArg.diskId}/eject`,
          method: "POST",
        }),
        invalidatesTags: ["volume"],
      }),
      deleteVolumeByMountPathHashMount: build.mutation<
        DeleteVolumeByMountPathHashMountApiResponse,
        DeleteVolumeByMountPathHashMountApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/${queryArg.mountPathHash}/mount`,
          method: "DELETE",
          params: {
            force: queryArg.force,
            lazy: queryArg.lazy,
          },
        }),
        invalidatesTags: ["volume"],
      }),
      postVolumeByMountPathHashMount: build.mutation<
        PostVolumeByMountPathHashMountApiResponse,
        PostVolumeByMountPathHashMountApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/${queryArg.mountPathHash}/mount`,
          method: "POST",
          body: queryArg.mountPointData,
        }),
        invalidatesTags: ["volume"],
      }),
      patchVolumeByMountPathHashSettings: build.mutation<
        PatchVolumeByMountPathHashSettingsApiResponse,
        PatchVolumeByMountPathHashSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/${queryArg.mountPathHash}/settings`,
          method: "PATCH",
          body: queryArg.mountPointData,
        }),
        invalidatesTags: ["volume"],
      }),
      putVolumeByMountPathHashSettings: build.mutation<
        PutVolumeByMountPathHashSettingsApiResponse,
        PutVolumeByMountPathHashSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/${queryArg.mountPathHash}/settings`,
          method: "PUT",
          body: queryArg.mountPointData,
        }),
        invalidatesTags: ["volume"],
      }),
      getVolumes: build.query<GetVolumesApiResponse, GetVolumesApiArg>({
        query: () => ({ url: `/volumes` }),
        providesTags: ["volume"],
      }),
    }),
    overrideExisting: false,
  });
export { injectedRtkApi as sratApi };
export type GetFilesystemsApiResponse =
  | /** status 200 OK */ (FilesystemType[] | null)
  | /** status default Error */ ErrorModel;
export type GetFilesystemsApiArg = void;
export type GetHealthApiResponse = /** status 200 OK */
  | HealthPing
  | /** status default Error */ ErrorModel;
export type GetHealthApiArg = void;
export type GetHostnameApiResponse = /** status 200 OK */
  | string
  | /** status default Error */ ErrorModel;
export type GetHostnameApiArg = void;
export type GetNicsApiResponse =
  | /** status 200 OK */ (InterfaceStat[] | null)
  | /** status default Error */ ErrorModel;
export type GetNicsApiArg = void;
export type PutRestartApiResponse = /** status default Error */ ErrorModel;
export type PutRestartApiArg = void;
export type PutSambaApplyApiResponse = /** status default Error */ ErrorModel;
export type PutSambaApplyApiArg = void;
export type GetSambaConfigApiResponse = /** status 200 OK */
  | SmbConf
  | /** status default Error */ ErrorModel;
export type GetSambaConfigApiArg = void;
export type GetSettingsApiResponse = /** status 200 OK */
  | Settings
  | /** status default Error */ ErrorModel;
export type GetSettingsApiArg = void;
export type PatchSettingsApiResponse = /** status 200 OK */
  | Settings
  | /** status default Error */ ErrorModel;
export type PatchSettingsApiArg = {
  body: JsonPatchOp[] | null;
};
export type PutSettingsApiResponse = /** status 200 OK */
  | Settings
  | /** status default Error */ ErrorModel;
export type PutSettingsApiArg = {
  settings: Settings;
};
export type PostShareApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PostShareApiArg = {
  sharedResource: SharedResource;
};
export type DeleteShareByShareNameApiResponse =
  /** status default Error */ ErrorModel;
export type DeleteShareByShareNameApiArg = {
  /** Name of the share */
  shareName: string;
};
export type GetShareByShareNameApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type GetShareByShareNameApiArg = {
  /** Name of the share */
  shareName: string;
};
export type PatchShareByShareNameApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PatchShareByShareNameApiArg = {
  /** Name of the share */
  shareName: string;
  body: JsonPatchOp[] | null;
};
export type PutShareByShareNameApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PutShareByShareNameApiArg = {
  /** Name of the share */
  shareName: string;
  sharedResource: SharedResource;
};
export type PutShareByShareNameDisableApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PutShareByShareNameDisableApiArg = {
  /** Name of the share to disable */
  shareName: string;
};
export type PutShareByShareNameEnableApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PutShareByShareNameEnableApiArg = {
  /** Name of the share to enable */
  shareName: string;
};
export type GetSharesApiResponse =
  | /** status 200 OK */ (SharedResource[] | null)
  | /** status default Error */ ErrorModel;
export type GetSharesApiArg = void;
export type SseApiResponse = /** status 200 OK */
  | (
      | {
          data: HealthPing;
          /** The event name. */
          event: "heartbeat";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
      | {
          data: Welcome;
          /** The event name. */
          event: "hello";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
      | {
          data: SharedResource[] | null;
          /** The event name. */
          event: "share";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
      | {
          data: UpdateProgress;
          /** The event name. */
          event: "updating";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
      | {
          data: Disk[] | null;
          /** The event name. */
          event: "volumes";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
    )[]
  | /** status default Error */ ErrorModel;
export type SseApiArg = void;
export type GetUpdateApiResponse = /** status 200 OK */
  | ReleaseAsset
  | /** status default Error */ ErrorModel;
export type GetUpdateApiArg = void;
export type PutUpdateApiResponse = /** status 200 OK */
  | UpdateProgress
  | /** status default Error */ ErrorModel;
export type PutUpdateApiArg = void;
export type GetUpdateChannelsApiResponse =
  | /** status 200 OK */ (string[] | null)
  | /** status default Error */ ErrorModel;
export type GetUpdateChannelsApiArg = void;
export type PostUserApiResponse = /** status 200 OK */
  | User
  | /** status default Error */ ErrorModel;
export type PostUserApiArg = {
  user: User;
};
export type DeleteUserByUsernameApiResponse =
  /** status default Error */ ErrorModel;
export type DeleteUserByUsernameApiArg = {
  /** Username */
  username: string;
};
export type PutUserByUsernameApiResponse = /** status 200 OK */
  | User
  | /** status default Error */ ErrorModel;
export type PutUserByUsernameApiArg = {
  /** Username */
  username: string;
  user: User;
};
export type PutUseradminApiResponse = /** status 200 OK */
  | User
  | /** status default Error */ ErrorModel;
export type PutUseradminApiArg = {
  user: User;
};
export type GetUsersApiResponse =
  | /** status 200 OK */ (User[] | null)
  | /** status default Error */ ErrorModel;
export type GetUsersApiArg = void;
export type PostVolumeDiskByDiskIdEjectApiResponse =
  /** status default Error */ ErrorModel;
export type PostVolumeDiskByDiskIdEjectApiArg = {
  /** The ID of the disk to eject (e.g., sda, sdb) */
  diskId: string;
};
export type DeleteVolumeByMountPathHashMountApiResponse =
  /** status default Error */ ErrorModel;
export type DeleteVolumeByMountPathHashMountApiArg = {
  mountPathHash: string;
  /** Force umount operation */
  force?: boolean;
  /** Lazy umount operation */
  lazy?: boolean;
};
export type PostVolumeByMountPathHashMountApiResponse = /** status 200 OK */
  | MountPointData
  | /** status default Error */ ErrorModel;
export type PostVolumeByMountPathHashMountApiArg = {
  mountPathHash: string;
  mountPointData: MountPointData;
};
export type PatchVolumeByMountPathHashSettingsApiResponse =
  /** status 200 OK */ MountPointData | /** status default Error */ ErrorModel;
export type PatchVolumeByMountPathHashSettingsApiArg = {
  mountPathHash: string;
  mountPointData: MountPointData;
};
export type PutVolumeByMountPathHashSettingsApiResponse = /** status 200 OK */
  | MountPointData
  | /** status default Error */ ErrorModel;
export type PutVolumeByMountPathHashSettingsApiArg = {
  mountPathHash: string;
  mountPointData: MountPointData;
};
export type GetVolumesApiResponse =
  | /** status 200 OK */ (Disk[] | null)
  | /** status default Error */ ErrorModel;
export type GetVolumesApiArg = void;
export type MountFlag = {
  description?: string;
  name: string;
  needsValue?: boolean;
  value?: string;
  value_description?: string;
  value_validation_regex?: string;
};
export type FilesystemType = {
  customMountFlags: MountFlag[] | null;
  mountFlags: MountFlag[] | null;
  name: string;
  type: string;
};
export type ErrorDetail = {
  /** Where the error occurred, e.g. 'body.items[3].tags' or 'path.thing-id' */
  location?: string;
  /** Error message text */
  message?: string;
  /** The value at the given location */
  value?: any;
};
export type ErrorModel = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  /** A human-readable explanation specific to this occurrence of the problem. */
  detail?: string;
  /** Optional list of individual error details */
  errors?: ErrorDetail[] | null;
  /** A URI reference that identifies the specific occurrence of the problem. */
  instance?: string;
  /** HTTP status code */
  status?: number;
  /** A short, human-readable summary of the problem type. This value should not change between occurrences of the error. */
  title?: string;
  /** A URI reference to human-readable documentation for the error. */
  type?: string;
};
export type AddonStatsData = {
  blk_read?: number;
  blk_write?: number;
  cpu_percent?: number;
  memory_limit?: number;
  memory_percent?: number;
  memory_usage?: number;
  network_rx?: number;
  network_tx?: number;
};
export type DataDirtyTracker = {
  settings: boolean;
  shares: boolean;
  users: boolean;
  volumes: boolean;
};
export type GlobalDiskStats = {
  total_iops?: number;
  total_read_latency_ms?: number;
  total_write_latency_ms?: number;
};
export type DiskIoStats = {
  device_name: string;
  read_iops?: number;
  read_latency_ms?: number;
  write_iops?: number;
  write_latency_ms?: number;
};
export type DiskHealth = {
  global: GlobalDiskStats;
  per_disk_io: DiskIoStats[] | null;
};
export type BinaryAsset = {
  browser_download_url?: string;
  id: number;
  name: string;
  size: number;
};
export type ReleaseAsset = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  arch_asset?: BinaryAsset;
  last_release?: string;
};
export type GlobalNicStats = {
  totalInboundTraffic: number;
  totalOutboundTraffic: number;
};
export type NicIoStats = {
  deviceName: string;
  inboundTraffic: number;
  outboundTraffic: number;
};
export type NetworkHealth = {
  global: GlobalNicStats;
  perNicIO: NicIoStats[] | null;
};
export type ProcessStatus = {
  connections: number;
  cpu_percent: number;
  create_time: string;
  is_running: boolean;
  memory_percent: number;
  name: string;
  open_files: number;
  pid: number;
  status: string[] | null;
};
export type SambaProcessStatus = {
  avahi: ProcessStatus;
  nmbd: ProcessStatus;
  smbd: ProcessStatus;
  wsdd2: ProcessStatus;
};
export type HealthPing = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  addon_stats: AddonStatsData;
  alive: boolean;
  aliveTime: number;
  build_version: string;
  dirty_tracking: DataDirtyTracker;
  disk_health: DiskHealth;
  last_error: string;
  last_release: ReleaseAsset;
  network_health: NetworkHealth;
  protected_mode: boolean;
  read_only: boolean;
  samba_process_status: SambaProcessStatus;
  secure_mode: boolean;
  startTime: number;
};
export type InterfaceAddr = {
  addr: string;
};
export type InterfaceStat = {
  addrs: InterfaceAddr[] | null;
  flags: string[] | null;
  hardwareAddr: string;
  index: number;
  mtu: number;
  name: string;
};
export type SmbConf = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  data: string;
};
export type Settings = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  allow_hosts?: string[];
  bind_all_interfaces?: boolean;
  compatibility_mode?: boolean;
  hostname?: string;
  interfaces?: string[];
  log_level?: string;
  mountoptions?: string[] | null;
  multi_channel?: boolean;
  update_channel?: Update_channel;
  workgroup?: string;
};
export type JsonPatchOp = {
  /** JSON Pointer for the source of a move or copy */
  from?: string;
  /** Operation name */
  op: Op;
  /** JSON Pointer to the field being operated on, or the destination of a move/copy operation */
  path: string;
  /** The value to set */
  value?: any;
};
export type MountPointData = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  custom_flags?: MountFlag[];
  device?: string;
  flags?: MountFlag[];
  fstype?: string;
  invalid?: boolean;
  invalid_error?: string;
  is_mounted?: boolean;
  is_to_mount_at_startup?: boolean;
  path: string;
  path_hash?: string;
  shares?: SharedResource[] | null;
  type: Type;
  warnings?: string;
};
export type User = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  is_admin?: boolean;
  password?: string;
  ro_shares?: string[] | null;
  rw_shares?: string[] | null;
  username: string;
  [key: string]: any;
};
export type SharedResource = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  disabled?: boolean;
  ha_status?: string;
  invalid?: boolean;
  is_ha_mounted?: boolean;
  mount_point_data?: MountPointData;
  name?: string;
  recycle_bin_enabled?: boolean;
  ro_users?: User[] | null;
  timemachine?: boolean;
  usage?: Usage;
  users?: User[] | null;
  veto_files?: string[];
  [key: string]: any;
};
export type Welcome = {
  message: string;
  supported_events: Supported_events;
  update_channel: Update_channel;
};
export type UpdateProgress = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  error_message?: string;
  last_release?: string;
  progress?: number;
  update_process_state?: Update_process_state;
};
export type Partition = {
  device?: string;
  host_mount_point_data?: MountPointData[];
  id?: string;
  mount_point_data?: MountPointData[];
  name?: string;
  size?: number;
  system?: boolean;
};
export type Disk = {
  connection_bus?: string;
  device?: string;
  ejectable?: boolean;
  id?: string;
  model?: string;
  partitions?: Partition[];
  removable?: boolean;
  revision?: string;
  seat?: string;
  serial?: string;
  size?: number;
  vendor?: string;
};
export enum Update_channel {
  None = "None",
  Develop = "Develop",
  Release = "Release",
  Prerelease = "Prerelease",
}
export enum Op {
  Add = "add",
  Remove = "remove",
  Replace = "replace",
  Move = "move",
  Copy = "copy",
  Test = "test",
}
export enum Type {
  Host = "HOST",
  Addon = "ADDON",
}
export enum Usage {
  None = "none",
  Backup = "backup",
  Media = "media",
  Share = "share",
  Internal = "internal",
}
export enum Supported_events {
  Hello = "hello",
  Updating = "updating",
  Volumes = "volumes",
  Heartbeat = "heartbeat",
  Share = "share",
  Dirty = "dirty",
}
export enum Update_process_state {
  Idle = "Idle",
  Checking = "Checking",
  NoUpgrde = "NoUpgrde",
  UpgradeAvailable = "UpgradeAvailable",
  Downloading = "Downloading",
  DownloadComplete = "DownloadComplete",
  Extracting = "Extracting",
  ExtractComplete = "ExtractComplete",
  Installing = "Installing",
  NeedRestart = "NeedRestart",
  Error = "Error",
}
export const {
  useGetFilesystemsQuery,
  useGetHealthQuery,
  useGetHostnameQuery,
  useGetNicsQuery,
  usePutRestartMutation,
  usePutSambaApplyMutation,
  useGetSambaConfigQuery,
  useGetSettingsQuery,
  usePatchSettingsMutation,
  usePutSettingsMutation,
  usePostShareMutation,
  useDeleteShareByShareNameMutation,
  useGetShareByShareNameQuery,
  usePatchShareByShareNameMutation,
  usePutShareByShareNameMutation,
  usePutShareByShareNameDisableMutation,
  usePutShareByShareNameEnableMutation,
  useGetSharesQuery,
  useSseQuery,
  useGetUpdateQuery,
  usePutUpdateMutation,
  useGetUpdateChannelsQuery,
  usePostUserMutation,
  useDeleteUserByUsernameMutation,
  usePutUserByUsernameMutation,
  usePutUseradminMutation,
  useGetUsersQuery,
  usePostVolumeDiskByDiskIdEjectMutation,
  useDeleteVolumeByMountPathHashMountMutation,
  usePostVolumeByMountPathHashMountMutation,
  usePatchVolumeByMountPathHashSettingsMutation,
  usePutVolumeByMountPathHashSettingsMutation,
  useGetVolumesQuery,
} = injectedRtkApi;
