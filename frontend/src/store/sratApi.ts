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
        providesTags: ["samba"],
      }),
      putSettings: build.mutation<PutSettingsApiResponse, PutSettingsApiArg>({
        query: (queryArg) => ({
          url: `/settings`,
          method: "PUT",
          body: queryArg.dtoSettings,
        }),
        invalidatesTags: ["samba"],
      }),
      patchSettings: build.mutation<
        PatchSettingsApiResponse,
        PatchSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/settings`,
          method: "PATCH",
          body: queryArg.dtoSettings,
        }),
        invalidatesTags: ["samba"],
      }),
      postShare: build.mutation<PostShareApiResponse, PostShareApiArg>({
        query: (queryArg) => ({
          url: `/share`,
          method: "POST",
          body: queryArg.dtoSharedResource,
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
      putShareByShareName: build.mutation<
        PutShareByShareNameApiResponse,
        PutShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/share/${queryArg.shareName}`,
          method: "PUT",
          body: queryArg.dtoSharedResource,
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
      getShares: build.query<GetSharesApiResponse, GetSharesApiArg>({
        query: () => ({ url: `/shares` }),
        providesTags: ["share"],
      }),
      getSharesUsages: build.query<
        GetSharesUsagesApiResponse,
        GetSharesUsagesApiArg
      >({
        query: () => ({ url: `/shares/usages` }),
        providesTags: ["share"],
      }),
      getSse: build.query<GetSseApiResponse, GetSseApiArg>({
        query: () => ({ url: `/sse` }),
        providesTags: ["system"],
      }),
      getSseEvents: build.query<GetSseEventsApiResponse, GetSseEventsApiArg>({
        query: () => ({ url: `/sse/events` }),
        providesTags: ["system"],
      }),
      putUpdate: build.mutation<PutUpdateApiResponse, PutUpdateApiArg>({
        query: () => ({ url: `/update`, method: "PUT" }),
        invalidatesTags: ["system"],
      }),
      postUser: build.mutation<PostUserApiResponse, PostUserApiArg>({
        query: (queryArg) => ({
          url: `/user`,
          method: "POST",
          body: queryArg.dtoUser,
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
          body: queryArg.dtoUser,
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
      getUseradmin: build.query<GetUseradminApiResponse, GetUseradminApiArg>({
        query: () => ({ url: `/useradmin` }),
        providesTags: ["user"],
      }),
      putUseradmin: build.mutation<PutUseradminApiResponse, PutUseradminApiArg>(
        {
          query: (queryArg) => ({
            url: `/useradmin`,
            method: "PUT",
            body: queryArg.dtoUser,
          }),
          invalidatesTags: ["user"],
        },
      ),
      getUsers: build.query<GetUsersApiResponse, GetUsersApiArg>({
        query: () => ({ url: `/users` }),
        providesTags: ["user"],
      }),
      postVolumeByIdMount: build.mutation<
        PostVolumeByIdMountApiResponse,
        PostVolumeByIdMountApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/${queryArg.id}/mount`,
          method: "POST",
          body: queryArg.dtoMountPointData,
        }),
        invalidatesTags: ["volume"],
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
      getVolumes: build.query<GetVolumesApiResponse, GetVolumesApiArg>({
        query: () => ({ url: `/volumes` }),
        providesTags: ["volume"],
      }),
    }),
    overrideExisting: false,
  });
export { injectedRtkApi as sratApi };
export type GetFilesystemsApiResponse = /** status 200 OK */ string[];
export type GetFilesystemsApiArg = void;
export type GetHealthApiResponse = /** status 200 OK */ DtoHealthPing;
export type GetHealthApiArg = void;
export type GetNicsApiResponse = /** status 200 OK */ DtoNetworkInfo;
export type GetNicsApiArg = void;
export type PutRestartApiResponse = unknown;
export type PutRestartApiArg = void;
export type PutSambaApplyApiResponse = unknown;
export type PutSambaApplyApiArg = void;
export type GetSambaConfigApiResponse = /** status 200 OK */ DtoSmbConf;
export type GetSambaConfigApiArg = void;
export type GetSettingsApiResponse = /** status 200 OK */ DtoSettings;
export type GetSettingsApiArg = void;
export type PutSettingsApiResponse = /** status 200 OK */ DtoSettings;
export type PutSettingsApiArg = {
  /** Update model */
  dtoSettings: DtoSettings;
};
export type PatchSettingsApiResponse = /** status 200 OK */ DtoSettings;
export type PatchSettingsApiArg = {
  /** Update model */
  dtoSettings: DtoSettings;
};
export type PostShareApiResponse = /** status 201 Created */ DtoSharedResource;
export type PostShareApiArg = {
  /** Create model */
  dtoSharedResource: DtoSharedResource;
};
export type GetShareByShareNameApiResponse =
  /** status 200 OK */ DtoSharedResource;
export type GetShareByShareNameApiArg = {
  /** Name */
  shareName: string;
};
export type PutShareByShareNameApiResponse =
  /** status 200 OK */ DtoSharedResource;
export type PutShareByShareNameApiArg = {
  /** Name */
  shareName: string;
  /** Update model */
  dtoSharedResource: DtoSharedResource;
};
export type DeleteShareByShareNameApiResponse = unknown;
export type DeleteShareByShareNameApiArg = {
  /** Name */
  shareName: string;
};
export type GetSharesApiResponse = /** status 200 OK */ DtoSharedResource[];
export type GetSharesApiArg = void;
export type GetSharesUsagesApiResponse = /** status 200 OK */ DtoHAMountUsage[];
export type GetSharesUsagesApiArg = void;
export type GetSseApiResponse = unknown;
export type GetSseApiArg = void;
export type GetSseEventsApiResponse = /** status 200 OK */ DtoEventType[];
export type GetSseEventsApiArg = void;
export type PutUpdateApiResponse = /** status 200 OK */ DtoUpdateProgress;
export type PutUpdateApiArg = void;
export type PostUserApiResponse = /** status 201 Created */ DtoUser;
export type PostUserApiArg = {
  /** Create model */
  dtoUser: DtoUser;
};
export type PutUserByUsernameApiResponse = /** status 200 OK */ DtoUser;
export type PutUserByUsernameApiArg = {
  /** Name */
  username: string;
  /** Update model */
  dtoUser: DtoUser;
};
export type DeleteUserByUsernameApiResponse = unknown;
export type DeleteUserByUsernameApiArg = {
  /** Name */
  username: string;
};
export type GetUseradminApiResponse = /** status 200 OK */ DtoUser;
export type GetUseradminApiArg = void;
export type PutUseradminApiResponse = /** status 200 OK */ DtoUser;
export type PutUseradminApiArg = {
  /** Update model */
  dtoUser: DtoUser;
};
export type GetUsersApiResponse = /** status 200 OK */ DtoUser[];
export type GetUsersApiArg = void;
export type PostVolumeByIdMountApiResponse =
  /** status 201 Created */ DtoMountPointData;
export type PostVolumeByIdMountApiArg = {
  /** id of the mountpoint to be mounted */
  id: number;
  /** Mount data */
  dtoMountPointData: DtoMountPointData;
};
export type DeleteVolumeByIdMountApiResponse = unknown;
export type DeleteVolumeByIdMountApiArg = {
  /** id of the mountpoint to be mounted */
  id: number;
  /** Umount forcefully - forces an unmount regardless of currently open or otherwise used files within the file system to be unmounted. */
  force: boolean;
  /** Umount lazily - disallows future uses of any files below path -- i.e. it hides the file system mounted at path, but the file system itself is still active and any currently open files can continue to be used. When all references to files from this file system are gone, the file system will actually be unmounted. */
  lazy: boolean;
};
export type GetVolumesApiResponse = /** status 200 OK */ DtoBlockInfo;
export type GetVolumesApiArg = void;
export type DtoErrorCode = {
  errorCode?: ErrorCode;
  errorMessage?: string;
  httpCode?: number;
};
export type TracerrFrame = {
  /** Func contains a function name. */
  func?: string;
  /** Line contains a line number. */
  line?: number;
  /** Path contains a file path. */
  path?: string;
};
export type DtoErrorInfo = {
  code?: DtoErrorCode;
  data?: {
    [key: string]: any;
  };
  deep_message?: string;
  error?: any;
  message?: string;
  trace?: TracerrFrame[];
};
export type DtoDataDirtyTracker = {
  settings?: boolean;
  shares?: boolean;
  users?: boolean;
  volumes?: boolean;
};
export type DtoBinaryAsset = {
  id?: number;
  size?: number;
};
export type DtoReleaseAsset = {
  arch_asset?: DtoBinaryAsset;
  /** ProgressStatus int8        `json:"update_status"` */
  last_release?: string;
};
export type DtoSambaProcessStatus = {
  connections?: number;
  cpu_percent?: number;
  create_time?: string;
  is_running?: boolean;
  memory_percent?: number;
  name?: string;
  open_files?: number;
  pid?: number;
  status?: string[];
};
export type DtoHealthPing = {
  alive?: boolean;
  aliveTime?: number;
  dirty_tracking?: DtoDataDirtyTracker;
  last_error?: string;
  last_release?: DtoReleaseAsset;
  read_only?: boolean;
  samba_process_status?: DtoSambaProcessStatus;
};
export type DtoNic = {
  /** Duplex is a string indicating the current duplex setting of this NIC,
    e.g. "Full" */
  duplex?: string;
  /** IsVirtual is true if the NIC is entirely virtual/emulated, false
    otherwise. */
  is_virtual?: boolean;
  /** MACAddress is the Media Access Control (MAC) address of this NIC. */
  mac_address?: string;
  /** Name is the string identifier the system gave this NIC. */
  name?: string;
  /** Speed is a string describing the link speed of this NIC, e.g. "1000Mb/s" */
  speed?: string;
};
export type DtoNetworkInfo = {
  nics?: DtoNic[];
};
export type DtoSmbConf = {
  data?: string;
};
export type DtoSettings = {
  allow_hosts?: string[];
  bind_all_interfaces?: boolean;
  compatibility_mode?: boolean;
  interfaces?: string[];
  log_level?: string;
  mountoptions?: string[];
  multi_channel?: boolean;
  recyle_bin_enabled?: boolean;
  update_channel?: DtoUpdateChannel;
  veto_files?: string[];
  workgroup?: string;
};
export type DtoMountPointData = {
  flags?: DtoMounDataFlag[];
  fstype?: string;
  id?: number;
  invalid?: boolean;
  invalid_error?: string;
  is_mounted?: boolean;
  path?: string;
  primary_path?: string;
  source?: string;
  warnings?: string;
};
export type DtoUser = {
  is_admin?: boolean;
  password?: string;
  username?: string;
};
export type DtoSharedResource = {
  disabled?: boolean;
  id?: number;
  invalid?: boolean;
  /** DeviceId       *uint64        `json:"device_id,omitempty"` */
  mount_point_data?: DtoMountPointData;
  name?: string;
  ro_users?: DtoUser[];
  timemachine?: boolean;
  usage?: DtoHAMountUsage;
  users?: DtoUser[];
};
export type DtoUpdateProgress = {
  last_release?: string;
  update_error?: string;
  update_status?: number;
};
export type DtoBlockPartition = {
  /** MountPoint is the path where this partition is mounted last time */
  default_mount_point?: string;
  /** DeviceId is the ID of the block device this partition is on. */
  device_id?: number;
  /** FilesystemLabel is the label of the filesystem contained on the
    partition. On Linux, this is derived from the `ID_FS_NAME` udev entry. */
  filesystem_label?: string;
  /** Label is the human-readable label given to the partition. On Linux, this
    is derived from the `ID_PART_ENTRY_NAME` udev entry. */
  label?: string;
  /** MountData contains additional data associated with the partition. */
  mount_data?: string;
  /** MountFlags contains the mount flags for the partition. */
  mount_flags?: DtoMounDataFlag[];
  /** MountPoint is the path where this partition is mounted. */
  mount_point?: string;
  /** Relative MountPointData */
  mount_point_data?: DtoMountPointData;
  /** Name is the system name given to the partition, e.g. "sda1". */
  name?: string;
  /** PartiionFlags contains the mount flags for the partition. */
  partition_flags?: DtoMounDataFlag[];
  /** IsReadOnly indicates if the partition is marked read-only. */
  read_only?: boolean;
  /** SizeBytes contains the total amount of storage, in bytes, this partition
    can consume. */
  size_bytes?: number;
  /** Type contains the type of the partition. */
  type?: string;
  /** UUID is the universally-unique identifier (UUID) for the partition.
    This will be volume UUID on Darwin, PartUUID on linux, empty on Windows. */
  uuid?: string;
};
export type DtoBlockInfo = {
  /** Partitions contains an array of pointers to `Partition` structs, one for
    each partition on any disk drive on the host system. */
  partitions?: DtoBlockPartition[];
  total_size_bytes?: number;
};
export enum ErrorCode {
  Unknown = 0,
  GenericError = 1,
  JsonMarshalError = 2,
  JsonUnmarshalError = 3,
  InvalidParameter = 4,
  MountFail = 5,
  UnmountFail = 6,
  DeviceNotFound = 7,
  NetworkTimeout = 8,
  PermissionDenied = 9,
}
export enum DtoUpdateChannel {
  Stable = "stable",
  Prerelease = "prerelease",
  None = "none",
}
export enum DtoMounDataFlag {
  MsRdonly = 1,
  MsNosuid = 2,
  MsNodev = 4,
  MsNoexec = 8,
  MsSynchronous = 16,
  MsRemount = 32,
  MsMandlock = 64,
  MsNoatime = 1024,
  MsNodiratime = 2048,
  MsBind = 4096,
  MsLazytime = 33554432,
  MsNouser = -2147483648,
  MsRelatime = 2097152,
}
export enum DtoHAMountUsage {
  None = "none",
  Backup = "backup",
  Media = "media",
  Share = "share",
  Internal = "internal",
}
export enum DtoEventType {
  Hello = "hello",
  Update = "update",
  Heartbeat = "heartbeat",
  Share = "share",
  Volumes = "volumes",
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
  usePutSettingsMutation,
  usePatchSettingsMutation,
  usePostShareMutation,
  useGetShareByShareNameQuery,
  usePutShareByShareNameMutation,
  useDeleteShareByShareNameMutation,
  useGetSharesQuery,
  useGetSharesUsagesQuery,
  useGetSseQuery,
  useGetSseEventsQuery,
  usePutUpdateMutation,
  usePostUserMutation,
  usePutUserByUsernameMutation,
  useDeleteUserByUsernameMutation,
  useGetUseradminQuery,
  usePutUseradminMutation,
  useGetUsersQuery,
  usePostVolumeByIdMountMutation,
  useDeleteVolumeByIdMountMutation,
  useGetVolumesQuery,
} = injectedRtkApi;
