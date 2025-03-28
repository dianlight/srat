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
      getShares: build.query<GetSharesApiResponse, GetSharesApiArg>({
        query: () => ({ url: `/shares` }),
        providesTags: ["share"],
      }),
      sse: build.query<SseApiResponse, SseApiArg>({
        query: () => ({ url: `/sse` }),
      }),
      putUpdate: build.mutation<PutUpdateApiResponse, PutUpdateApiArg>({
        query: () => ({ url: `/update`, method: "PUT" }),
        invalidatesTags: ["system"],
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
      getUseradmin: build.query<GetUseradminApiResponse, GetUseradminApiArg>({
        query: () => ({ url: `/useradmin` }),
        providesTags: ["user"],
      }),
      patchUseradmin: build.mutation<
        PatchUseradminApiResponse,
        PatchUseradminApiArg
      >({
        query: (queryArg) => ({
          url: `/useradmin`,
          method: "PATCH",
          body: queryArg.body,
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
      deleteVolumeByIdMount: build.mutation<
        DeleteVolumeByIdMountApiResponse,
        DeleteVolumeByIdMountApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/${queryArg.id}/mount`,
          method: "DELETE",
          params: {
            force: queryArg.force,
            lazy: queryArg.lazy,
          },
        }),
        invalidatesTags: ["volume"],
      }),
      postVolumeByIdMount: build.mutation<
        PostVolumeByIdMountApiResponse,
        PostVolumeByIdMountApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/${queryArg.id}/mount`,
          method: "POST",
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
  | /** status 200 OK */ (string[] | null)
  | /** status default Error */ ErrorModel;
export type GetFilesystemsApiArg = void;
export type GetHealthApiResponse = /** status 200 OK */
  | HealthPing
  | /** status default Error */ ErrorModel;
export type GetHealthApiArg = void;
export type GetNicsApiResponse = /** status 200 OK */
  | NetworkInfo
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
export type GetSharesApiResponse =
  | /** status 200 OK */ (SharedResource[] | null)
  | /** status default Error */ ErrorModel;
export type GetSharesApiArg = void;
export type SseApiResponse = /** status 200 OK */
  | (
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
          data: BlockInfo;
          /** The event name. */
          event: "volumes";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
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
          data: SharedResource[] | null;
          /** The event name. */
          event: "share";
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
          data: ReleaseAsset;
          /** The event name. */
          event: "update";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
    )[]
  | /** status default Error */ ErrorModel;
export type SseApiArg = void;
export type PutUpdateApiResponse = /** status 200 OK */
  | UpdateProgress
  | /** status default Error */ ErrorModel;
export type PutUpdateApiArg = void;
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
export type GetUseradminApiResponse = /** status 200 OK */
  | User
  | /** status default Error */ ErrorModel;
export type GetUseradminApiArg = void;
export type PatchUseradminApiResponse = /** status 200 OK */
  | User
  | /** status default Error */ ErrorModel;
export type PatchUseradminApiArg = {
  body: JsonPatchOp[] | null;
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
export type DeleteVolumeByIdMountApiResponse =
  /** status default Error */ ErrorModel;
export type DeleteVolumeByIdMountApiArg = {
  /** ID of the driver to mount */
  id: number;
  /** Force umount operation */
  force?: boolean;
  /** Lazy umount operation */
  lazy?: boolean;
};
export type PostVolumeByIdMountApiResponse = /** status 200 OK */
  | MountPointData
  | /** status default Error */ ErrorModel;
export type PostVolumeByIdMountApiArg = {
  /** ID of the driver to mount */
  id: number;
  mountPointData: MountPointData;
};
export type GetVolumesApiResponse = /** status 200 OK */
  | BlockInfo
  | /** status default Error */ ErrorModel;
export type GetVolumesApiArg = void;
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
export type DataDirtyTracker = {
  settings: boolean;
  shares: boolean;
  users: boolean;
  volumes: boolean;
};
export type BinaryAsset = {
  id: number;
  size: number;
};
export type ReleaseAsset = {
  arch_asset?: BinaryAsset;
  last_release?: string;
};
export type SambaProcessStatus = {
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
export type HealthPing = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  alive: boolean;
  aliveTime: number;
  dirty_tracking: DataDirtyTracker;
  last_error: string;
  last_release: ReleaseAsset;
  read_only: boolean;
  samba_process_status: SambaProcessStatus;
};
export type Nic = {
  duplex: string;
  is_virtual: boolean;
  mac_address: string;
  name: string;
  speed: string;
};
export type NetworkInfo = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  nics: Nic[] | null;
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
  interfaces?: string[];
  log_level?: string;
  mountoptions?: string[] | null;
  multi_channel?: boolean;
  recyle_bin_enabled?: boolean;
  update_channel?: Update_channel;
  veto_files?: string[];
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
  flags?: Flags[] | null;
  fstype?: string;
  id: number;
  invalid?: boolean;
  invalid_error?: string;
  is_mounted?: boolean;
  path: string;
  primary_path?: string;
  source?: string;
  warnings?: string;
};
export type User = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  is_admin?: boolean;
  password?: string;
  username: string | null;
};
export type SharedResource = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  disabled?: boolean;
  invalid?: boolean;
  mount_point_data?: MountPointData;
  name?: string;
  ro_users?: User[] | null;
  timemachine?: boolean;
  usage?: Usage;
  users?: User[] | null;
};
export type UpdateProgress = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  last_release?: string;
  update_error?: string;
  update_status: number;
};
export type BlockPartition = {
  default_mount_point: string;
  device_id: number | null;
  filesystem_label: string;
  label: string;
  mount_data: string;
  mount_flags: Mount_flags[] | null;
  mount_point: string;
  mount_point_data: MountPointData;
  name: string;
  partition_flags: Partition_flags[] | null;
  read_only: boolean;
  size_bytes: number;
  type: string;
  uuid: string;
};
export type BlockInfo = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  partitions: BlockPartition[] | null;
  total_size_bytes: number;
};
export type Welcome = {
  message: string;
  supported_events: Supported_events;
};
export enum Update_channel {
  Stable = "stable",
  Prerelease = "prerelease",
  None = "none",
}
export enum Op {
  Add = "add",
  Remove = "remove",
  Replace = "replace",
  Move = "move",
  Copy = "copy",
  Test = "test",
}
export enum Flags {
  MsRdonly = "MS_RDONLY",
  MsNosuid = "MS_NOSUID",
  MsNodev = "MS_NODEV",
  MsNoexec = "MS_NOEXEC",
  MsSynchronous = "MS_SYNCHRONOUS",
  MsRemount = "MS_REMOUNT",
  MsMandlock = "MS_MANDLOCK",
  MsNoatime = "MS_NOATIME",
  MsNodiratime = "MS_NODIRATIME",
  MsBind = "MS_BIND",
  MsLazytime = "MS_LAZYTIME",
  MsNouser = "MS_NOUSER",
  MsRelatime = "MS_RELATIME",
}
export enum Usage {
  None = "none",
  Backup = "backup",
  Media = "media",
  Share = "share",
  Internal = "internal",
}
export enum Mount_flags {
  MsRdonly = "MS_RDONLY",
  MsNosuid = "MS_NOSUID",
  MsNodev = "MS_NODEV",
  MsNoexec = "MS_NOEXEC",
  MsSynchronous = "MS_SYNCHRONOUS",
  MsRemount = "MS_REMOUNT",
  MsMandlock = "MS_MANDLOCK",
  MsNoatime = "MS_NOATIME",
  MsNodiratime = "MS_NODIRATIME",
  MsBind = "MS_BIND",
  MsLazytime = "MS_LAZYTIME",
  MsNouser = "MS_NOUSER",
  MsRelatime = "MS_RELATIME",
}
export enum Partition_flags {
  MsRdonly = "MS_RDONLY",
  MsNosuid = "MS_NOSUID",
  MsNodev = "MS_NODEV",
  MsNoexec = "MS_NOEXEC",
  MsSynchronous = "MS_SYNCHRONOUS",
  MsRemount = "MS_REMOUNT",
  MsMandlock = "MS_MANDLOCK",
  MsNoatime = "MS_NOATIME",
  MsNodiratime = "MS_NODIRATIME",
  MsBind = "MS_BIND",
  MsLazytime = "MS_LAZYTIME",
  MsNouser = "MS_NOUSER",
  MsRelatime = "MS_RELATIME",
}
export enum Supported_events {
  Hello = "hello",
  Update = "update",
  Updating = "updating",
  Volumes = "volumes",
  Heartbeat = "heartbeat",
  Share = "share",
  Dirty = "dirty",
}
export const {
  useGetFilesystemsQuery,
  useGetHealthQuery,
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
  useGetSharesQuery,
  useSseQuery,
  usePutUpdateMutation,
  usePostUserMutation,
  useDeleteUserByUsernameMutation,
  usePutUserByUsernameMutation,
  useGetUseradminQuery,
  usePatchUseradminMutation,
  usePutUseradminMutation,
  useGetUsersQuery,
  useDeleteVolumeByIdMountMutation,
  usePostVolumeByIdMountMutation,
  useGetVolumesQuery,
} = injectedRtkApi;
