import { emptySplitApi as api } from "./emptyApi";
export const addTagTypes = [
  "system",
  "dev",
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
        query: (queryArg) => ({
          url: `/filesystems`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["system"],
      }),
      getHealth: build.query<GetHealthApiResponse, GetHealthApiArg>({
        query: (queryArg) => ({
          url: `/health`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["system"],
      }),
      getNics: build.query<GetNicsApiResponse, GetNicsApiArg>({
        query: (queryArg) => ({
          url: `/nics`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["system"],
      }),
      putRestart: build.mutation<PutRestartApiResponse, PutRestartApiArg>({
        query: (queryArg) => ({
          url: `/restart`,
          method: "PUT",
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["dev"],
      }),
      putSambaApply: build.mutation<
        PutSambaApplyApiResponse,
        PutSambaApplyApiArg
      >({
        query: (queryArg) => ({
          url: `/samba/apply`,
          method: "PUT",
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["samba"],
      }),
      getSambaConfig: build.query<
        GetSambaConfigApiResponse,
        GetSambaConfigApiArg
      >({
        query: (queryArg) => ({
          url: `/samba/config`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["samba"],
      }),
      getSettings: build.query<GetSettingsApiResponse, GetSettingsApiArg>({
        query: (queryArg) => ({
          url: `/settings`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["samba"],
      }),
      patchSettings: build.mutation<
        PatchSettingsApiResponse,
        PatchSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/settings`,
          method: "PATCH",
          body: queryArg.settings,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["samba"],
      }),
      putSettings: build.mutation<PutSettingsApiResponse, PutSettingsApiArg>({
        query: (queryArg) => ({
          url: `/settings`,
          method: "PUT",
          body: queryArg.settings,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["samba"],
      }),
      postShare: build.mutation<PostShareApiResponse, PostShareApiArg>({
        query: (queryArg) => ({
          url: `/share`,
          method: "POST",
          body: queryArg.sharedResource,
          headers: {
            Accept: queryArg.accept,
          },
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
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["share"],
      }),
      getShareByShareName: build.query<
        GetShareByShareNameApiResponse,
        GetShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/share/${queryArg.shareName}`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["share"],
      }),
      putShareByShareName: build.mutation<
        PutShareByShareNameApiResponse,
        PutShareByShareNameApiArg
      >({
        query: (queryArg) => ({
          url: `/share/${queryArg.shareName}`,
          method: "PUT",
          body: queryArg.sharedResource,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["share"],
      }),
      getShares: build.query<GetSharesApiResponse, GetSharesApiArg>({
        query: (queryArg) => ({
          url: `/shares`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["share"],
      }),
      getSse: build.query<GetSseApiResponse, GetSseApiArg>({
        query: (queryArg) => ({
          url: `/sse`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["system"],
      }),
      putUpdate: build.mutation<PutUpdateApiResponse, PutUpdateApiArg>({
        query: (queryArg) => ({
          url: `/update`,
          method: "PUT",
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["system"],
      }),
      postUser: build.mutation<PostUserApiResponse, PostUserApiArg>({
        query: (queryArg) => ({
          url: `/user`,
          method: "POST",
          body: queryArg.user,
          headers: {
            Accept: queryArg.accept,
          },
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
          headers: {
            Accept: queryArg.accept,
          },
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
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["user"],
      }),
      getUseradmin: build.query<GetUseradminApiResponse, GetUseradminApiArg>({
        query: (queryArg) => ({
          url: `/useradmin`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["user"],
      }),
      putUseradmin: build.mutation<PutUseradminApiResponse, PutUseradminApiArg>(
        {
          query: (queryArg) => ({
            url: `/useradmin`,
            method: "PUT",
            body: queryArg.user,
            headers: {
              Accept: queryArg.accept,
            },
          }),
          invalidatesTags: ["user"],
        },
      ),
      getUsers: build.query<GetUsersApiResponse, GetUsersApiArg>({
        query: (queryArg) => ({
          url: `/users`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["user"],
      }),
      deleteVolumeByIdMount: build.mutation<
        DeleteVolumeByIdMountApiResponse,
        DeleteVolumeByIdMountApiArg
      >({
        query: (queryArg) => ({
          url: `/volume/${queryArg.id}/mount`,
          method: "DELETE",
          headers: {
            Accept: queryArg.accept,
          },
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
          headers: {
            Accept: queryArg.accept,
          },
        }),
        invalidatesTags: ["volume"],
      }),
      getVolumes: build.query<GetVolumesApiResponse, GetVolumesApiArg>({
        query: (queryArg) => ({
          url: `/volumes`,
          headers: {
            Accept: queryArg.accept,
          },
        }),
        providesTags: ["volume"],
      }),
    }),
    overrideExisting: false,
  });
export { injectedRtkApi as sratApi };
export type GetFilesystemsApiResponse = /** status 200 OK */ FilesystemType[];
export type GetFilesystemsApiArg = {
  accept?: string;
};
export type GetHealthApiResponse = /** status 200 OK */ HealthPing;
export type GetHealthApiArg = {
  accept?: string;
};
export type GetNicsApiResponse = /** status 200 OK */ NetworkInfo;
export type GetNicsApiArg = {
  accept?: string;
};
export type PutRestartApiResponse = /** status 200 OK */ Bool;
export type PutRestartApiArg = {
  accept?: string;
};
export type PutSambaApplyApiResponse = /** status 200 OK */ Bool;
export type PutSambaApplyApiArg = {
  accept?: string;
};
export type GetSambaConfigApiResponse = /** status 200 OK */ SmbConf;
export type GetSambaConfigApiArg = {
  accept?: string;
};
export type GetSettingsApiResponse = /** status 200 OK */ Settings;
export type GetSettingsApiArg = {
  accept?: string;
};
export type PatchSettingsApiResponse = /** status 200 OK */ Settings;
export type PatchSettingsApiArg = {
  accept?: string;
  /** Request body for dto.Settings */
  settings: Settings;
};
export type PutSettingsApiResponse = /** status 200 OK */ Settings;
export type PutSettingsApiArg = {
  accept?: string;
  /** Request body for dto.Settings */
  settings: Settings;
};
export type PostShareApiResponse = /** status 200 OK */ SharedResource;
export type PostShareApiArg = {
  accept?: string;
  /** Request body for dto.SharedResource */
  sharedResource: SharedResource;
};
export type DeleteShareByShareNameApiResponse = /** status 200 OK */ Bool;
export type DeleteShareByShareNameApiArg = {
  accept?: string;
  shareName: string;
};
export type GetShareByShareNameApiResponse =
  /** status 200 OK */ SharedResource;
export type GetShareByShareNameApiArg = {
  accept?: string;
  shareName: string;
};
export type PutShareByShareNameApiResponse =
  /** status 200 OK */ SharedResource;
export type PutShareByShareNameApiArg = {
  accept?: string;
  shareName: string;
  /** Request body for dto.SharedResource */
  sharedResource: SharedResource;
};
export type GetSharesApiResponse = /** status 200 OK */ SharedResource[];
export type GetSharesApiArg = {
  accept?: string;
};
export type GetSseApiResponse = /** status 200 OK */ EventMessageEnvelope;
export type GetSseApiArg = {
  accept?: string;
};
export type PutUpdateApiResponse = /** status 200 OK */ UpdateProgress;
export type PutUpdateApiArg = {
  accept?: string;
};
export type PostUserApiResponse = /** status 200 OK */ User;
export type PostUserApiArg = {
  accept?: string;
  /** Request body for dto.User */
  user: User;
};
export type DeleteUserByUsernameApiResponse = /** status 200 OK */ Bool;
export type DeleteUserByUsernameApiArg = {
  accept?: string;
  username: string;
};
export type PutUserByUsernameApiResponse = /** status 200 OK */ User;
export type PutUserByUsernameApiArg = {
  accept?: string;
  username: string;
  /** Request body for dto.User */
  user: User;
};
export type GetUseradminApiResponse = /** status 200 OK */ User;
export type GetUseradminApiArg = {
  accept?: string;
};
export type PutUseradminApiResponse = /** status 200 OK */ User;
export type PutUseradminApiArg = {
  accept?: string;
  /** Request body for dto.User */
  user: User;
};
export type GetUsersApiResponse = /** status 200 OK */ User[];
export type GetUsersApiArg = {
  accept?: string;
};
export type DeleteVolumeByIdMountApiResponse = /** status 200 OK */ Bool;
export type DeleteVolumeByIdMountApiArg = {
  /** Force Umount */
  force?: boolean;
  /** Lazy Umount  */
  lazy?: boolean;
  accept?: string;
  id: string;
};
export type PostVolumeByIdMountApiResponse =
  /** status 200 OK */ MountPointData;
export type PostVolumeByIdMountApiArg = {
  accept?: string;
  id: string;
  /** Request body for dto.MountPointData */
  mountPointData: MountPointData;
};
export type GetVolumesApiResponse = /** status 200 OK */ BlockInfo;
export type GetVolumesApiArg = {
  accept?: string;
};
export type FilesystemType = string;
export type HttpError = {
  /** Human readable error message */
  detail?: string | null;
  errors?:
    | {
        more?: {
          [key: string]: any;
        };
        name?: string;
        reason?: string;
      }[]
    | null;
  instance?: string | null;
  /** HTTP status code */
  status?: number | null;
  /** Short title of the error */
  title?: string | null;
  /** URL of the error type. Can be used to lookup the error in a documentation */
  type?: string | null;
};
export type HealthPing = {
  alive?: boolean;
  aliveTime?: number;
  dirty_tracking?: {
    settings?: boolean;
    shares?: boolean;
    users?: boolean;
    volumes?: boolean;
  };
  last_error?: string;
  last_release?: {
    arch_asset?: {
      id?: number;
      size?: number;
    } | null;
    last_release?: string | null;
  };
  read_only?: boolean;
  samba_process_status?: {
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
};
export type NetworkInfo = {
  nics?: {
    duplex?: string;
    is_virtual?: boolean;
    mac_address?: string;
    name?: string;
    speed?: string;
  }[];
};
export type Bool = boolean;
export type SmbConf = {
  data?: string;
};
export type Settings = {
  allow_hosts?: string[] | null;
  bind_all_interfaces?: boolean | null;
  compatibility_mode?: boolean | null;
  interfaces?: string[] | null;
  log_level?: string | null;
  mountoptions?: string[] | null;
  multi_channel?: boolean | null;
  recyle_bin_enabled?: boolean | null;
  update_channel?: string | null;
  veto_files?: string[] | null;
  workgroup?: string | null;
};
export type SharedResource = {
  /** bool schema */
  disabled?: boolean | null;
  /** bool schema */
  invalid?: boolean | null;
  /** MountPointData schema */
  mount_point_data?: {
    flags?: number[] | null;
    fstype?: string;
    id?: number;
    invalid?: boolean | null;
    invalid_error?: string | null;
    is_mounted?: boolean | null;
    path?: string;
    primary_path?: string;
    source?: string | null;
    warnings?: string | null;
  } | null;
  name?: string | null;
  ro_users?: {
    /** bool schema */
    is_admin?: boolean;
    password?: string | null;
    username?: string | null;
  }[];
  /** bool schema */
  timemachine?: boolean | null;
  usage?: string | null;
  users?: {
    /** bool schema */
    is_admin?: boolean;
    password?: string | null;
    username?: string | null;
  }[];
};
export type EventMessageEnvelope = {
  data?: any;
  event?: string;
  id?: string;
};
export type UpdateProgress = {
  last_release?: string | null;
  update_error?: string | null;
  update_status?: number;
};
export type User = {
  /** bool schema */
  is_admin?: boolean | null;
  password?: string | null;
  username?: string | null;
};
export type MountPointData = {
  flags?: number[] | null;
  fstype?: string;
  id?: number;
  invalid?: boolean | null;
  invalid_error?: string | null;
  is_mounted?: boolean | null;
  path?: string;
  primary_path?: string;
  source?: string | null;
  warnings?: string | null;
};
export type BlockInfo = {
  partitions?: ({
    default_mount_point?: string;
    device_id?: number | null;
    filesystem_label?: string;
    label?: string;
    mount_data?: string;
    mount_flags?: number[];
    mount_point?: string;
    mount_point_data?: {
      flags?: number[];
      fstype?: string;
      id?: number;
      invalid?: boolean;
      invalid_error?: string | null;
      is_mounted?: boolean;
      path?: string;
      primary_path?: string;
      source?: string;
      warnings?: string | null;
    };
    name?: string;
    partition_flags?: number[];
    read_only?: boolean;
    size_bytes?: number;
    type?: string;
    uuid?: string;
  } | null)[];
  total_size_bytes?: number;
};
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
  usePutShareByShareNameMutation,
  useGetSharesQuery,
  useGetSseQuery,
  usePutUpdateMutation,
  usePostUserMutation,
  useDeleteUserByUsernameMutation,
  usePutUserByUsernameMutation,
  useGetUseradminQuery,
  usePutUseradminMutation,
  useGetUsersQuery,
  useDeleteVolumeByIdMountMutation,
  usePostVolumeByIdMountMutation,
  useGetVolumesQuery,
} = injectedRtkApi;
