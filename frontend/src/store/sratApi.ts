import { emptySplitApi as api } from "./emptyApi";
export const addTagTypes = [
  "system",
  "Issues",
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
      getApiFilesystems: build.query<
        GetApiFilesystemsApiResponse,
        GetApiFilesystemsApiArg
      >({
        query: () => ({ url: `/api/filesystems` }),
        providesTags: ["system"],
      }),
      getApiHealth: build.query<GetApiHealthApiResponse, GetApiHealthApiArg>({
        query: () => ({ url: `/api/health` }),
        providesTags: ["system"],
      }),
      getApiHostname: build.query<
        GetApiHostnameApiResponse,
        GetApiHostnameApiArg
      >({
        query: () => ({ url: `/api/hostname` }),
        providesTags: ["system"],
      }),
      getApiIssues: build.query<GetApiIssuesApiResponse, GetApiIssuesApiArg>({
        query: () => ({ url: `/api/issues` }),
        providesTags: ["Issues"],
      }),
      postApiIssues: build.mutation<
        PostApiIssuesApiResponse,
        PostApiIssuesApiArg
      >({
        query: (queryArg) => ({
          url: `/api/issues`,
          method: "POST",
          body: queryArg.issue,
        }),
        invalidatesTags: ["Issues"],
      }),
      deleteApiIssuesById: build.mutation<
        DeleteApiIssuesByIdApiResponse,
        DeleteApiIssuesByIdApiArg
      >({
        query: (queryArg) => ({
          url: `/api/issues/${queryArg.id}`,
          method: "DELETE",
        }),
        invalidatesTags: ["Issues"],
      }),
      putApiIssuesById: build.mutation<
        PutApiIssuesByIdApiResponse,
        PutApiIssuesByIdApiArg
      >({
        query: (queryArg) => ({
          url: `/api/issues/${queryArg.id}`,
          method: "PUT",
          body: queryArg.issue,
        }),
        invalidatesTags: ["Issues"],
      }),
      getApiNics: build.query<GetApiNicsApiResponse, GetApiNicsApiArg>({
        query: () => ({ url: `/api/nics` }),
        providesTags: ["system"],
      }),
      putApiRestart: build.mutation<
        PutApiRestartApiResponse,
        PutApiRestartApiArg
      >({
        query: () => ({ url: `/api/restart`, method: "PUT" }),
        invalidatesTags: ["system"],
      }),
      putApiSambaApply: build.mutation<
        PutApiSambaApplyApiResponse,
        PutApiSambaApplyApiArg
      >({
        query: () => ({ url: `/api/samba/apply`, method: "PUT" }),
        invalidatesTags: ["samba"],
      }),
      getApiSambaConfig: build.query<
        GetApiSambaConfigApiResponse,
        GetApiSambaConfigApiArg
      >({
        query: () => ({ url: `/api/samba/config` }),
        providesTags: ["samba"],
      }),
      getApiSambaStatus: build.query<
        GetApiSambaStatusApiResponse,
        GetApiSambaStatusApiArg
      >({
        query: () => ({ url: `/api/samba/status` }),
        providesTags: ["samba"],
      }),
      getApiSettings: build.query<
        GetApiSettingsApiResponse,
        GetApiSettingsApiArg
      >({
        query: () => ({ url: `/api/settings` }),
        providesTags: ["system"],
      }),
      patchApiSettings: build.mutation<
        PatchApiSettingsApiResponse,
        PatchApiSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/api/settings`,
          method: "PATCH",
          body: queryArg.body,
        }),
        invalidatesTags: ["system"],
      }),
      putApiSettings: build.mutation<
        PutApiSettingsApiResponse,
        PutApiSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/api/settings`,
          method: "PUT",
          body: queryArg.settings,
        }),
        invalidatesTags: ["system"],
      }),
      postApiShare: build.mutation<PostApiShareApiResponse, PostApiShareApiArg>(
        {
          query: (queryArg) => ({
            url: `/api/share`,
            method: "POST",
            body: queryArg.sharedResource,
          }),
          invalidatesTags: ["share"],
        },
      ),
      deleteApiShareByShareName: build.mutation<
        DeleteApiShareByShareNameApiResponse,
        DeleteApiShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/api/share/${queryArg.shareName}`,
          method: "DELETE",
        }),
        invalidatesTags: ["share"],
      }),
      getApiShareByShareName: build.query<
        GetApiShareByShareNameApiResponse,
        GetApiShareByShareNameApiArg
      >({
        query: (queryArg) => ({ url: `/api/share/${queryArg.shareName}` }),
        providesTags: ["share"],
      }),
      patchApiShareByShareName: build.mutation<
        PatchApiShareByShareNameApiResponse,
        PatchApiShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/api/share/${queryArg.shareName}`,
          method: "PATCH",
          body: queryArg.body,
        }),
        invalidatesTags: ["share"],
      }),
      putApiShareByShareName: build.mutation<
        PutApiShareByShareNameApiResponse,
        PutApiShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/api/share/${queryArg.shareName}`,
          method: "PUT",
          body: queryArg.sharedResource,
        }),
        invalidatesTags: ["share"],
      }),
      putApiShareByShareNameDisable: build.mutation<
        PutApiShareByShareNameDisableApiResponse,
        PutApiShareByShareNameDisableApiArg
      >({
        query: (queryArg) => ({
          url: `/api/share/${queryArg.shareName}/disable`,
          method: "PUT",
        }),
        invalidatesTags: ["share"],
      }),
      putApiShareByShareNameEnable: build.mutation<
        PutApiShareByShareNameEnableApiResponse,
        PutApiShareByShareNameEnableApiArg
      >({
        query: (queryArg) => ({
          url: `/api/share/${queryArg.shareName}/enable`,
          method: "PUT",
        }),
        invalidatesTags: ["share"],
      }),
      getApiShares: build.query<GetApiSharesApiResponse, GetApiSharesApiArg>({
        query: () => ({ url: `/api/shares` }),
        providesTags: ["share"],
      }),
      sse: build.query<SseApiResponse, SseApiArg>({
        query: () => ({ url: `/api/sse` }),
        providesTags: ["system"],
      }),
      getApiStatus: build.query<GetApiStatusApiResponse, GetApiStatusApiArg>({
        query: () => ({ url: `/api/status` }),
        providesTags: ["system"],
      }),
      getApiTelemetryInternetConnection: build.query<
        GetApiTelemetryInternetConnectionApiResponse,
        GetApiTelemetryInternetConnectionApiArg
      >({
        query: () => ({ url: `/api/telemetry/internet-connection` }),
        providesTags: ["system"],
      }),
      getApiTelemetryModes: build.query<
        GetApiTelemetryModesApiResponse,
        GetApiTelemetryModesApiArg
      >({
        query: () => ({ url: `/api/telemetry/modes` }),
        providesTags: ["system"],
      }),
      getApiUpdate: build.query<GetApiUpdateApiResponse, GetApiUpdateApiArg>({
        query: () => ({ url: `/api/update` }),
        providesTags: ["system"],
      }),
      putApiUpdate: build.mutation<PutApiUpdateApiResponse, PutApiUpdateApiArg>(
        {
          query: () => ({ url: `/api/update`, method: "PUT" }),
          invalidatesTags: ["system"],
        },
      ),
      getApiUpdateChannels: build.query<
        GetApiUpdateChannelsApiResponse,
        GetApiUpdateChannelsApiArg
      >({
        query: () => ({ url: `/api/update_channels` }),
        providesTags: ["system"],
      }),
      postApiUser: build.mutation<PostApiUserApiResponse, PostApiUserApiArg>({
        query: (queryArg) => ({
          url: `/api/user`,
          method: "POST",
          body: queryArg.user,
        }),
        invalidatesTags: ["user"],
      }),
      deleteApiUserByUsername: build.mutation<
        DeleteApiUserByUsernameApiResponse,
        DeleteApiUserByUsernameApiArg
      >({
        query: (queryArg) => ({
          url: `/api/user/${queryArg.username}`,
          method: "DELETE",
        }),
        invalidatesTags: ["user"],
      }),
      putApiUserByUsername: build.mutation<
        PutApiUserByUsernameApiResponse,
        PutApiUserByUsernameApiArg
      >({
        query: (queryArg) => ({
          url: `/api/user/${queryArg.username}`,
          method: "PUT",
          body: queryArg.user,
        }),
        invalidatesTags: ["user"],
      }),
      putApiUseradmin: build.mutation<
        PutApiUseradminApiResponse,
        PutApiUseradminApiArg
      >({
        query: (queryArg) => ({
          url: `/api/useradmin`,
          method: "PUT",
          body: queryArg.user,
        }),
        invalidatesTags: ["user"],
      }),
      getApiUsers: build.query<GetApiUsersApiResponse, GetApiUsersApiArg>({
        query: () => ({ url: `/api/users` }),
        providesTags: ["user"],
      }),
      postApiVolumeDiskByDiskIdEject: build.mutation<
        PostApiVolumeDiskByDiskIdEjectApiResponse,
        PostApiVolumeDiskByDiskIdEjectApiArg
      >({
        query: (queryArg) => ({
          url: `/api/volume/disk/${queryArg.diskId}/eject`,
          method: "POST",
        }),
        invalidatesTags: ["volume"],
      }),
      deleteApiVolumeByMountPathHashMount: build.mutation<
        DeleteApiVolumeByMountPathHashMountApiResponse,
        DeleteApiVolumeByMountPathHashMountApiArg
      >({
        query: (queryArg) => ({
          url: `/api/volume/${queryArg.mountPathHash}/mount`,
          method: "DELETE",
          params: {
            force: queryArg.force,
            lazy: queryArg.lazy,
          },
        }),
        invalidatesTags: ["volume"],
      }),
      postApiVolumeByMountPathHashMount: build.mutation<
        PostApiVolumeByMountPathHashMountApiResponse,
        PostApiVolumeByMountPathHashMountApiArg
      >({
        query: (queryArg) => ({
          url: `/api/volume/${queryArg.mountPathHash}/mount`,
          method: "POST",
          body: queryArg.mountPointData,
        }),
        invalidatesTags: ["volume"],
      }),
      patchApiVolumeByMountPathHashSettings: build.mutation<
        PatchApiVolumeByMountPathHashSettingsApiResponse,
        PatchApiVolumeByMountPathHashSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/api/volume/${queryArg.mountPathHash}/settings`,
          method: "PATCH",
          body: queryArg.mountPointData,
        }),
        invalidatesTags: ["volume"],
      }),
      putApiVolumeByMountPathHashSettings: build.mutation<
        PutApiVolumeByMountPathHashSettingsApiResponse,
        PutApiVolumeByMountPathHashSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/api/volume/${queryArg.mountPathHash}/settings`,
          method: "PUT",
          body: queryArg.mountPointData,
        }),
        invalidatesTags: ["volume"],
      }),
      getApiVolumes: build.query<GetApiVolumesApiResponse, GetApiVolumesApiArg>(
        {
          query: () => ({ url: `/api/volumes` }),
          providesTags: ["volume"],
        },
      ),
    }),
    overrideExisting: false,
  });
export { injectedRtkApi as sratApi };
export type GetApiFilesystemsApiResponse =
  | /** status 200 OK */ (FilesystemType[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiFilesystemsApiArg = void;
export type GetApiHealthApiResponse = /** status 200 OK */
  | HealthPing
  | /** status default Error */ ErrorModel;
export type GetApiHealthApiArg = void;
export type GetApiHostnameApiResponse = /** status 200 OK */
  | string
  | /** status default Error */ ErrorModel;
export type GetApiHostnameApiArg = void;
export type GetApiIssuesApiResponse =
  | /** status 200 OK */ (Issue[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiIssuesApiArg = void;
export type PostApiIssuesApiResponse = /** status 200 OK */
  | Issue
  | /** status default Error */ ErrorModel;
export type PostApiIssuesApiArg = {
  issue: Issue;
};
export type DeleteApiIssuesByIdApiResponse = /** status 200 OK */
  | ResolveIssueOutputBody
  | /** status default Error */ ErrorModel;
export type DeleteApiIssuesByIdApiArg = {
  id: number;
};
export type PutApiIssuesByIdApiResponse = /** status 200 OK */
  | Issue
  | /** status default Error */ ErrorModel;
export type PutApiIssuesByIdApiArg = {
  id: number;
  issue: Issue;
};
export type GetApiNicsApiResponse =
  | /** status 200 OK */ (InterfaceStat[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiNicsApiArg = void;
export type PutApiRestartApiResponse = /** status default Error */ ErrorModel;
export type PutApiRestartApiArg = void;
export type PutApiSambaApplyApiResponse =
  /** status default Error */ ErrorModel;
export type PutApiSambaApplyApiArg = void;
export type GetApiSambaConfigApiResponse = /** status 200 OK */
  | SmbConf
  | /** status default Error */ ErrorModel;
export type GetApiSambaConfigApiArg = void;
export type GetApiSambaStatusApiResponse = /** status 200 OK */
  | SambaStatus
  | /** status default Error */ ErrorModel;
export type GetApiSambaStatusApiArg = void;
export type GetApiSettingsApiResponse = /** status 200 OK */
  | Settings
  | /** status default Error */ ErrorModel;
export type GetApiSettingsApiArg = void;
export type PatchApiSettingsApiResponse = /** status 200 OK */
  | Settings
  | /** status default Error */ ErrorModel;
export type PatchApiSettingsApiArg = {
  body: JsonPatchOp[] | null;
};
export type PutApiSettingsApiResponse = /** status 200 OK */
  | Settings
  | /** status default Error */ ErrorModel;
export type PutApiSettingsApiArg = {
  settings: Settings;
};
export type PostApiShareApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PostApiShareApiArg = {
  sharedResource: SharedResource;
};
export type DeleteApiShareByShareNameApiResponse =
  /** status default Error */ ErrorModel;
export type DeleteApiShareByShareNameApiArg = {
  /** Name of the share */
  shareName: string;
};
export type GetApiShareByShareNameApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type GetApiShareByShareNameApiArg = {
  /** Name of the share */
  shareName: string;
};
export type PatchApiShareByShareNameApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PatchApiShareByShareNameApiArg = {
  /** Name of the share */
  shareName: string;
  body: JsonPatchOp[] | null;
};
export type PutApiShareByShareNameApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PutApiShareByShareNameApiArg = {
  /** Name of the share */
  shareName: string;
  sharedResource: SharedResource;
};
export type PutApiShareByShareNameDisableApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PutApiShareByShareNameDisableApiArg = {
  /** Name of the share to disable */
  shareName: string;
};
export type PutApiShareByShareNameEnableApiResponse = /** status 200 OK */
  | SharedResource
  | /** status default Error */ ErrorModel;
export type PutApiShareByShareNameEnableApiArg = {
  /** Name of the share to enable */
  shareName: string;
};
export type GetApiSharesApiResponse =
  | /** status 200 OK */ (SharedResource[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiSharesApiArg = void;
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
export type GetApiStatusApiResponse = /** status 200 OK */
  | boolean
  | /** status default Error */ ErrorModel;
export type GetApiStatusApiArg = void;
export type GetApiTelemetryInternetConnectionApiResponse = /** status 200 OK */
  | boolean
  | /** status default Error */ ErrorModel;
export type GetApiTelemetryInternetConnectionApiArg = void;
export type GetApiTelemetryModesApiResponse =
  | /** status 200 OK */ (string[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiTelemetryModesApiArg = void;
export type GetApiUpdateApiResponse = /** status 200 OK */
  | ReleaseAsset
  | /** status default Error */ ErrorModel;
export type GetApiUpdateApiArg = void;
export type PutApiUpdateApiResponse = /** status 200 OK */
  | UpdateProgress
  | /** status default Error */ ErrorModel;
export type PutApiUpdateApiArg = void;
export type GetApiUpdateChannelsApiResponse =
  | /** status 200 OK */ (string[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiUpdateChannelsApiArg = void;
export type PostApiUserApiResponse = /** status 200 OK */
  | User
  | /** status default Error */ ErrorModel;
export type PostApiUserApiArg = {
  user: User;
};
export type DeleteApiUserByUsernameApiResponse =
  /** status default Error */ ErrorModel;
export type DeleteApiUserByUsernameApiArg = {
  /** Username */
  username: string;
};
export type PutApiUserByUsernameApiResponse = /** status 200 OK */
  | User
  | /** status default Error */ ErrorModel;
export type PutApiUserByUsernameApiArg = {
  /** Username */
  username: string;
  user: User;
};
export type PutApiUseradminApiResponse = /** status 200 OK */
  | User
  | /** status default Error */ ErrorModel;
export type PutApiUseradminApiArg = {
  user: User;
};
export type GetApiUsersApiResponse =
  | /** status 200 OK */ (User[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiUsersApiArg = void;
export type PostApiVolumeDiskByDiskIdEjectApiResponse =
  /** status default Error */ ErrorModel;
export type PostApiVolumeDiskByDiskIdEjectApiArg = {
  /** The ID of the disk to eject (e.g., sda, sdb) */
  diskId: string;
};
export type DeleteApiVolumeByMountPathHashMountApiResponse =
  /** status default Error */ ErrorModel;
export type DeleteApiVolumeByMountPathHashMountApiArg = {
  mountPathHash: string;
  /** Force umount operation */
  force?: boolean;
  /** Lazy umount operation */
  lazy?: boolean;
};
export type PostApiVolumeByMountPathHashMountApiResponse = /** status 200 OK */
  | MountPointData
  | /** status default Error */ ErrorModel;
export type PostApiVolumeByMountPathHashMountApiArg = {
  mountPathHash: string;
  mountPointData: MountPointData;
};
export type PatchApiVolumeByMountPathHashSettingsApiResponse =
  /** status 200 OK */ MountPointData | /** status default Error */ ErrorModel;
export type PatchApiVolumeByMountPathHashSettingsApiArg = {
  mountPathHash: string;
  mountPointData: MountPointData;
};
export type PutApiVolumeByMountPathHashSettingsApiResponse =
  /** status 200 OK */ MountPointData | /** status default Error */ ErrorModel;
export type PutApiVolumeByMountPathHashSettingsApiArg = {
  mountPathHash: string;
  mountPointData: MountPointData;
};
export type GetApiVolumesApiResponse =
  | /** status 200 OK */ (Disk[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiVolumesApiArg = void;
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
  total_iops: number;
  total_read_latency_ms: number;
  total_write_latency_ms: number;
};
export type SmartRangeValue = {
  code?: number;
  min?: number;
  thresholds?: number;
  value: number;
  worst?: number;
};
export type SmartTempValue = {
  max?: number;
  min?: number;
  overtemp_counter?: number;
  value: number;
};
export type SmartInfo = {
  disk_type?: Disk_type;
  others?: {
    [key: string]: SmartRangeValue;
  };
  power_cycle_count: SmartRangeValue;
  power_on_hours: SmartRangeValue;
  temperature: SmartTempValue;
};
export type DiskIoStats = {
  device_description: string;
  device_name: string;
  read_iops: number;
  read_latency_ms: number;
  smart_data?: SmartInfo;
  write_iops: number;
  write_latency_ms: number;
};
export type PerPartitionInfo = {
  device: string;
  free_space_bytes: number;
  fsck_needed: boolean;
  fsck_supported: boolean;
  fstype: string;
  mount_point: string;
  name?: string;
  total_space_bytes: number;
};
export type DiskHealth = {
  global: GlobalDiskStats;
  per_disk_io: DiskIoStats[] | null;
  per_partition_info: {
    [key: string]: PerPartitionInfo[] | null;
  };
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
  deviceMaxSpeed: number;
  deviceName: string;
  inboundTraffic: number;
  outboundTraffic: number;
};
export type NetworkStats = {
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
  nmbd: ProcessStatus;
  smbd: ProcessStatus;
  srat: ProcessStatus;
  wsdd2: ProcessStatus;
};
export type Value = {
  channel_id: string;
  creation_time: string;
  local_address: string;
  remote_address: string;
};
export type SambaSessionEncryptionStruct = {
  cipher: string;
  degree: string;
};
export type SambaServerId = {
  pid: string;
  task_id: string;
  unique_id: string;
  vnn: string;
};
export type SambaSessionSigningStruct = {
  cipher: string;
  degree: string;
};
export type SambaSession = {
  auth_time: string;
  channels: {
    [key: string]: Value;
  };
  creation_time: string;
  encryption: SambaSessionEncryptionStruct;
  gid: number;
  groupname: string;
  hostname: string;
  remote_machine: string;
  server_id: SambaServerId;
  session_dialect: string;
  session_id: string;
  signing: SambaSessionSigningStruct;
  uid: number;
  username: string;
};
export type SambaTconEncryptionStruct = {
  cipher: string;
  degree: string;
};
export type SambaTconSigningStruct = {
  cipher: string;
  degree: string;
};
export type SambaTcon = {
  connected_at: string;
  device: string;
  encryption: SambaTconEncryptionStruct;
  machine: string;
  server_id: SambaServerId;
  service: string;
  session_id: string;
  share: string;
  signing: SambaTconSigningStruct;
  tcon_id: string;
};
export type SambaStatus = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  sessions: {
    [key: string]: SambaSession;
  };
  smb_conf: string;
  tcons: {
    [key: string]: SambaTcon;
  };
  timestamp: string;
  version: string;
};
export type HealthPing = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  addon_stats: AddonStatsData;
  alive: boolean;
  aliveTime: number;
  dirty_tracking: DataDirtyTracker;
  disk_health: DiskHealth;
  last_error: string;
  last_release: ReleaseAsset;
  network_health: NetworkStats;
  samba_process_status: SambaProcessStatus;
  samba_status: SambaStatus;
  uptime: number;
};
export type Issue = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  date: string;
  description: string;
  detailLink?: string;
  id: number;
  ignored: boolean;
  repeating: number;
  resolutionLink?: string;
  severity?: Severity;
  title: string;
};
export type ResolveIssueOutputBody = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
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
  export_stats_to_ha?: boolean;
  hostname?: string;
  interfaces?: string[];
  local_master?: boolean;
  log_level?: string;
  mountoptions?: string[] | null;
  multi_channel?: boolean;
  telemetry_mode?: Telemetry_mode;
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
export type Partition = {
  device_path?: string;
  fs_type?: string;
  host_mount_point_data?: MountPointData[];
  id?: string;
  legacy_device_name?: string;
  legacy_device_path?: string;
  mount_point_data?: MountPointData[];
  name?: string;
  size?: number;
  system?: boolean;
};
export type MountPointData = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  custom_flags?: MountFlag[];
  device_id?: string;
  disk_label?: string;
  disk_serial?: string;
  disk_size?: number;
  flags?: MountFlag[];
  fstype?: string;
  invalid?: boolean;
  invalid_error?: string;
  is_mounted?: boolean;
  is_to_mount_at_startup?: boolean;
  is_write_supported?: boolean;
  partition?: Partition;
  path: string;
  path_hash?: string;
  shares?: SharedResource[] | null;
  time_machine_support?: Time_machine_support;
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
  guest_ok?: boolean;
  ha_status?: string;
  invalid?: boolean;
  is_ha_mounted?: boolean;
  mount_point_data?: MountPointData;
  name?: string;
  recycle_bin_enabled?: boolean;
  ro_users?: User[] | null;
  timemachine?: boolean;
  timemachine_max_size?: string;
  usage?: Usage;
  users?: User[] | null;
  veto_files?: string[];
  [key: string]: any;
};
export type Welcome = {
  active_clients: number;
  build_version: string;
  machine_id?: string;
  message: string;
  protected_mode: boolean;
  read_only: boolean;
  secure_mode: boolean;
  startTime: number;
  supported_events: Supported_events[] | null;
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
export type Disk = {
  connection_bus?: string;
  device_path?: string;
  ejectable?: boolean;
  id?: string;
  legacy_device_name?: string;
  legacy_device_path?: string;
  model?: string;
  partitions?: Partition[];
  removable?: boolean;
  revision?: string;
  seat?: string;
  serial?: string;
  size?: number;
  smart_info?: SmartInfo;
  vendor?: string;
};
export enum Disk_type {
  Sata = "SATA",
  NvMe = "NVMe",
  Scsi = "SCSI",
  Unknown = "Unknown",
}
export enum Severity {
  Error = "error",
  Warning = "warning",
  Info = "info",
  Success = "success",
}
export enum Telemetry_mode {
  Ask = "Ask",
  All = "All",
  Errors = "Errors",
  Disabled = "Disabled",
}
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
export enum Time_machine_support {
  Unsupported = "unsupported",
  Supported = "supported",
  Experimental = "experimental",
  Unknown = "unknown",
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
  useGetApiFilesystemsQuery,
  useGetApiHealthQuery,
  useGetApiHostnameQuery,
  useGetApiIssuesQuery,
  usePostApiIssuesMutation,
  useDeleteApiIssuesByIdMutation,
  usePutApiIssuesByIdMutation,
  useGetApiNicsQuery,
  usePutApiRestartMutation,
  usePutApiSambaApplyMutation,
  useGetApiSambaConfigQuery,
  useGetApiSambaStatusQuery,
  useGetApiSettingsQuery,
  usePatchApiSettingsMutation,
  usePutApiSettingsMutation,
  usePostApiShareMutation,
  useDeleteApiShareByShareNameMutation,
  useGetApiShareByShareNameQuery,
  usePatchApiShareByShareNameMutation,
  usePutApiShareByShareNameMutation,
  usePutApiShareByShareNameDisableMutation,
  usePutApiShareByShareNameEnableMutation,
  useGetApiSharesQuery,
  useSseQuery,
  useGetApiStatusQuery,
  useGetApiTelemetryInternetConnectionQuery,
  useGetApiTelemetryModesQuery,
  useGetApiUpdateQuery,
  usePutApiUpdateMutation,
  useGetApiUpdateChannelsQuery,
  usePostApiUserMutation,
  useDeleteApiUserByUsernameMutation,
  usePutApiUserByUsernameMutation,
  usePutApiUseradminMutation,
  useGetApiUsersQuery,
  usePostApiVolumeDiskByDiskIdEjectMutation,
  useDeleteApiVolumeByMountPathHashMountMutation,
  usePostApiVolumeByMountPathHashMountMutation,
  usePatchApiVolumeByMountPathHashSettingsMutation,
  usePutApiVolumeByMountPathHashSettingsMutation,
  useGetApiVolumesQuery,
} = injectedRtkApi;
