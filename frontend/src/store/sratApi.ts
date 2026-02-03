import { emptySplitApi as api } from "./emptyApi";
export const addTagTypes = [
  "system",
  "disk",
  "smart",
  "hdidle",
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
      getApiCapabilities: build.query<
        GetApiCapabilitiesApiResponse,
        GetApiCapabilitiesApiArg
      >({
        query: () => ({ url: `/api/capabilities` }),
        providesTags: ["system"],
      }),
      getApiDiskByDiskIdHdidleConfig: build.query<
        GetApiDiskByDiskIdHdidleConfigApiResponse,
        GetApiDiskByDiskIdHdidleConfigApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/hdidle/config`,
        }),
        providesTags: ["disk"],
      }),
      patchApiDiskByDiskIdHdidleConfig: build.mutation<
        PatchApiDiskByDiskIdHdidleConfigApiResponse,
        PatchApiDiskByDiskIdHdidleConfigApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/hdidle/config`,
          method: "PATCH",
          body: queryArg.body,
        }),
        invalidatesTags: ["disk"],
      }),
      putApiDiskByDiskIdHdidleConfig: build.mutation<
        PutApiDiskByDiskIdHdidleConfigApiResponse,
        PutApiDiskByDiskIdHdidleConfigApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/hdidle/config`,
          method: "PUT",
          body: queryArg.hdIdleDevice,
        }),
        invalidatesTags: ["disk"],
      }),
      getApiDiskByDiskIdHdidleInfo: build.query<
        GetApiDiskByDiskIdHdidleInfoApiResponse,
        GetApiDiskByDiskIdHdidleInfoApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/hdidle/info`,
        }),
        providesTags: ["disk"],
      }),
      getApiDiskByDiskIdHdidleSupport: build.query<
        GetApiDiskByDiskIdHdidleSupportApiResponse,
        GetApiDiskByDiskIdHdidleSupportApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/hdidle/support`,
        }),
        providesTags: ["disk"],
      }),
      postApiDiskByDiskIdSmartDisable: build.mutation<
        PostApiDiskByDiskIdSmartDisableApiResponse,
        PostApiDiskByDiskIdSmartDisableApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/smart/disable`,
          method: "POST",
        }),
        invalidatesTags: ["smart"],
      }),
      postApiDiskByDiskIdSmartEnable: build.mutation<
        PostApiDiskByDiskIdSmartEnableApiResponse,
        PostApiDiskByDiskIdSmartEnableApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/smart/enable`,
          method: "POST",
        }),
        invalidatesTags: ["disk"],
      }),
      getApiDiskByDiskIdSmartHealth: build.query<
        GetApiDiskByDiskIdSmartHealthApiResponse,
        GetApiDiskByDiskIdSmartHealthApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/smart/health`,
        }),
        providesTags: ["disk"],
      }),
      getApiDiskByDiskIdSmartInfo: build.query<
        GetApiDiskByDiskIdSmartInfoApiResponse,
        GetApiDiskByDiskIdSmartInfoApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/smart/info`,
        }),
        providesTags: ["disk"],
      }),
      getApiDiskByDiskIdSmartStatus: build.query<
        GetApiDiskByDiskIdSmartStatusApiResponse,
        GetApiDiskByDiskIdSmartStatusApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/smart/status`,
        }),
        providesTags: ["disk"],
      }),
      getApiDiskByDiskIdSmartTest: build.query<
        GetApiDiskByDiskIdSmartTestApiResponse,
        GetApiDiskByDiskIdSmartTestApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/smart/test`,
        }),
        providesTags: ["disk"],
      }),
      postApiDiskByDiskIdSmartTestAbort: build.mutation<
        PostApiDiskByDiskIdSmartTestAbortApiResponse,
        PostApiDiskByDiskIdSmartTestAbortApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/smart/test/abort`,
          method: "POST",
        }),
        invalidatesTags: ["disk"],
      }),
      postApiDiskByDiskIdSmartTestStart: build.mutation<
        PostApiDiskByDiskIdSmartTestStartApiResponse,
        PostApiDiskByDiskIdSmartTestStartApiArg
      >({
        query: (queryArg) => ({
          url: `/api/disk/${queryArg.diskId}/smart/test/start`,
          method: "POST",
          body: queryArg.postDiskByDiskIdSmartTestStartRequest,
        }),
        invalidatesTags: ["disk"],
      }),
      getApiFilesystems: build.query<
        GetApiFilesystemsApiResponse,
        GetApiFilesystemsApiArg
      >({
        query: () => ({ url: `/api/filesystems` }),
        providesTags: ["system"],
      }),
      postApiHdidleStart: build.mutation<
        PostApiHdidleStartApiResponse,
        PostApiHdidleStartApiArg
      >({
        query: () => ({ url: `/api/hdidle/start`, method: "POST" }),
        invalidatesTags: ["hdidle"],
      }),
      postApiHdidleStop: build.mutation<
        PostApiHdidleStopApiResponse,
        PostApiHdidleStopApiArg
      >({
        query: () => ({ url: `/api/hdidle/stop`, method: "POST" }),
        invalidatesTags: ["hdidle"],
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
      postApiIssuesReport: build.mutation<
        PostApiIssuesReportApiResponse,
        PostApiIssuesReportApiArg
      >({
        query: (queryArg) => ({
          url: `/api/issues/report`,
          method: "POST",
          body: queryArg.issueReportRequest,
        }),
        invalidatesTags: ["Issues"],
      }),
      getApiIssuesTemplate: build.query<
        GetApiIssuesTemplateApiResponse,
        GetApiIssuesTemplateApiArg
      >({
        query: () => ({ url: `/api/issues/template` }),
        providesTags: ["Issues"],
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
            body: queryArg.sharedResourcePostData,
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
          body: queryArg.sharedResourcePostData,
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
      deleteApiVolume: build.mutation<
        DeleteApiVolumeApiResponse,
        DeleteApiVolumeApiArg
      >({
        query: (queryArg) => ({
          url: `/api/volume`,
          method: "DELETE",
          params: {
            mount_path: queryArg.mountPath,
            force: queryArg.force,
          },
        }),
        invalidatesTags: ["volume"],
      }),
      postApiVolumeMount: build.mutation<
        PostApiVolumeMountApiResponse,
        PostApiVolumeMountApiArg
      >({
        query: (queryArg) => ({
          url: `/api/volume/mount`,
          method: "POST",
          body: queryArg.mountPointData,
        }),
        invalidatesTags: ["volume"],
      }),
      patchApiVolumeSettings: build.mutation<
        PatchApiVolumeSettingsApiResponse,
        PatchApiVolumeSettingsApiArg
      >({
        query: (queryArg) => ({
          url: `/api/volume/settings`,
          method: "PATCH",
          body: queryArg.patchMountPointData,
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
export type GetApiCapabilitiesApiResponse = /** status 200 OK */
  | SystemCapabilities
  | /** status default Error */ ErrorModel;
export type GetApiCapabilitiesApiArg = void;
export type GetApiDiskByDiskIdHdidleConfigApiResponse = /** status 200 OK */
  | HdIdleDevice
  | /** status default Error */ ErrorModel;
export type GetApiDiskByDiskIdHdidleConfigApiArg = {
  /** The disk ID (not the device path) */
  diskId: string;
};
export type PatchApiDiskByDiskIdHdidleConfigApiResponse = /** status 200 OK */
  | HdIdleDevice
  | /** status default Error */ ErrorModel;
export type PatchApiDiskByDiskIdHdidleConfigApiArg = {
  /** The disk ID or device path */
  diskId: string;
  body: JsonPatchOp[] | null;
};
export type PutApiDiskByDiskIdHdidleConfigApiResponse = /** status 200 OK */
  | HdIdleDevice
  | /** status default Error */ ErrorModel;
export type PutApiDiskByDiskIdHdidleConfigApiArg = {
  /** The disk ID or device path */
  diskId: string;
  hdIdleDevice: HdIdleDevice;
};
export type GetApiDiskByDiskIdHdidleInfoApiResponse = /** status 200 OK */
  | HdIdleDeviceStatus
  | /** status default Error */ ErrorModel;
export type GetApiDiskByDiskIdHdidleInfoApiArg = {
  /** The disk ID (not the device path) */
  diskId: string;
};
export type GetApiDiskByDiskIdHdidleSupportApiResponse = /** status 200 OK */
  | HdIdleDeviceSupport
  | /** status default Error */ ErrorModel;
export type GetApiDiskByDiskIdHdidleSupportApiArg = {
  /** The disk ID (not the device path) */
  diskId: string;
};
export type PostApiDiskByDiskIdSmartDisableApiResponse = /** status 200 OK */
  | string
  | /** status default Error */ ErrorModel;
export type PostApiDiskByDiskIdSmartDisableApiArg = {
  /** The disk ID or device path */
  diskId: string;
};
export type PostApiDiskByDiskIdSmartEnableApiResponse = /** status 200 OK */
  | string
  | /** status default Error */ ErrorModel;
export type PostApiDiskByDiskIdSmartEnableApiArg = {
  /** The disk ID or device path */
  diskId: string;
};
export type GetApiDiskByDiskIdSmartHealthApiResponse = /** status 200 OK */
  | SmartHealthStatus
  | /** status default Error */ ErrorModel;
export type GetApiDiskByDiskIdSmartHealthApiArg = {
  /** The disk ID or device path */
  diskId: string;
};
export type GetApiDiskByDiskIdSmartInfoApiResponse = /** status 200 OK */
  | SmartInfo
  | /** status default Error */ ErrorModel;
export type GetApiDiskByDiskIdSmartInfoApiArg = {
  /** The disk ID or device path */
  diskId: string;
};
export type GetApiDiskByDiskIdSmartStatusApiResponse = /** status 200 OK */
  | SmartStatus
  | /** status default Error */ ErrorModel;
export type GetApiDiskByDiskIdSmartStatusApiArg = {
  /** The disk ID or device path */
  diskId: string;
};
export type GetApiDiskByDiskIdSmartTestApiResponse = /** status 200 OK */
  | SmartTestStatus
  | /** status default Error */ ErrorModel;
export type GetApiDiskByDiskIdSmartTestApiArg = {
  /** The disk ID or device path */
  diskId: string;
};
export type PostApiDiskByDiskIdSmartTestAbortApiResponse = /** status 200 OK */
  | string
  | /** status default Error */ ErrorModel;
export type PostApiDiskByDiskIdSmartTestAbortApiArg = {
  /** The disk ID or device path */
  diskId: string;
};
export type PostApiDiskByDiskIdSmartTestStartApiResponse = /** status 200 OK */
  | string
  | /** status default Error */ ErrorModel;
export type PostApiDiskByDiskIdSmartTestStartApiArg = {
  /** The disk ID or device path */
  diskId: string;
  postDiskByDiskIdSmartTestStartRequest: PostDiskByDiskIdSmartTestStartRequest;
};
export type GetApiFilesystemsApiResponse =
  | /** status 200 OK */ (FilesystemType[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiFilesystemsApiArg = void;
export type PostApiHdidleStartApiResponse = /** status 200 OK */
  | StartHdIdleServiceOutputBody
  | /** status default Error */ ErrorModel;
export type PostApiHdidleStartApiArg = void;
export type PostApiHdidleStopApiResponse = /** status 200 OK */
  | StopHdIdleServiceOutputBody
  | /** status default Error */ ErrorModel;
export type PostApiHdidleStopApiArg = void;
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
export type PostApiIssuesReportApiResponse = /** status 200 OK */
  | IssueReportResponse
  | /** status default Error */ ErrorModel;
export type PostApiIssuesReportApiArg = {
  issueReportRequest: IssueReportRequest;
};
export type GetApiIssuesTemplateApiResponse = /** status 200 OK */
  | IssueTemplateResponse
  | /** status default Error */ ErrorModel;
export type GetApiIssuesTemplateApiArg = void;
export type DeleteApiIssuesByIdApiResponse =
  /** status default Error */ ErrorModel;
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
  sharedResourcePostData: SharedResourcePostData;
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
  sharedResourcePostData: SharedResourcePostData;
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
          data: DataDirtyTracker;
          /** The event name. */
          event: "dirty_data_tracker";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
      | {
          data: ErrorDataModel;
          /** The event name. */
          event: "error";
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
          event: "shares";
          /** The event ID. */
          id?: number;
          /** The retry time in milliseconds. */
          retry?: number;
        }
      | {
          data: SmartTestStatus;
          /** The event name. */
          event: "smart_test_status";
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
export type DeleteApiVolumeApiResponse = /** status default Error */ ErrorModel;
export type DeleteApiVolumeApiArg = {
  mountPath?: string;
  /** Force umount operation */
  force?: boolean;
};
export type PostApiVolumeMountApiResponse = /** status 200 OK */
  | MountPointData
  | /** status default Error */ ErrorModel;
export type PostApiVolumeMountApiArg = {
  mountPointData: MountPointData;
};
export type PatchApiVolumeSettingsApiResponse = /** status 200 OK */
  | MountPointData
  | /** status default Error */ ErrorModel;
export type PatchApiVolumeSettingsApiArg = {
  patchMountPointData: PatchMountPointData;
};
export type GetApiVolumesApiResponse =
  | /** status 200 OK */ (Disk[] | null)
  | /** status default Error */ ErrorModel;
export type GetApiVolumesApiArg = void;
export type SystemCapabilities = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  /** Whether QUIC kernel module is loaded */
  has_kernel_module: boolean;
  /** Installed Samba version */
  samba_version: string;
  /** Whether Samba version >= 4.23.0 */
  samba_version_sufficient: boolean;
  /** Whether NFS is supported */
  support_nfs: boolean;
  /** Whether SMB over QUIC is supported */
  supports_quic: boolean;
  /** Reason why QUIC is not supported */
  unsupported_reason?: string;
};
export type ErrorDetail = {
  /** Where the error occurred, e.g. 'body.items[3].tags' or 'path.thing-id' */
  location?: string;
  /** Error message text */
  message?: string;
  /** The value at the given location */
  value?: unknown;
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
export type HdIdleDevice = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  command_type?: Command_type;
  device_path?: string;
  disk_id?: string;
  enabled?: Enabled;
  error_message?: string;
  idle_time: number;
  power_condition: number;
  recommended_command?: string;
  supported?: boolean;
  supports_ata?: boolean;
  supports_scsi?: boolean;
};
export type JsonPatchOp = {
  /** JSON Pointer for the source of a move or copy */
  from?: string;
  /** Operation name */
  op: Op;
  /** JSON Pointer to the field being operated on, or the destination of a move/copy operation */
  path: string;
  /** The value to set */
  value?: unknown;
};
export type HdIdleDeviceStatus = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  last_io_at?: string;
  name?: string;
  spin_down_at?: string;
  spin_up_at?: string;
  spun_down: boolean;
};
export type HdIdleDeviceSupport = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  device_path?: string;
  error_message?: string;
  recommended_command?: string;
  supported?: boolean;
  supports_ata?: boolean;
  supports_scsi?: boolean;
};
export type SmartHealthStatus = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  failing_attributes?: string[] | null;
  overall_status: string;
  passed: boolean;
};
export type SmartInfo = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  disk_id?: string;
  disk_type?: Disk_type;
  firmware_version?: string;
  model_family?: string;
  model_name?: string;
  rotation_rate?: number;
  serial_number?: string;
  supported: boolean;
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
export type SmartStatus = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  enabled: boolean;
  in_standby: boolean;
  is_in_danger: boolean;
  is_in_warning: boolean;
  is_test_passed: boolean;
  is_test_running: boolean;
  others?: {
    [key: string]: SmartRangeValue;
  };
  power_cycle_count: SmartRangeValue;
  power_on_hours: SmartRangeValue;
  temperature: SmartTempValue;
};
export type SmartTestStatus = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  disk_id: string;
  lba_of_first_error?: string;
  percent_complete?: number;
  running: boolean;
  status: string;
  test_type: string;
};
export type PostDiskByDiskIdSmartTestStartRequest = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  /** Type of test: short, long, or conveyance */
  test_type: string;
};
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
export type StartHdIdleServiceOutputBody = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  message: string;
  running: boolean;
};
export type StopHdIdleServiceOutputBody = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  message: string;
  running: boolean;
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
};
export type GlobalDiskStats = {
  total_iops: number;
  total_read_latency_ms: number;
  total_write_latency_ms: number;
};
export type PerDiskInfo = {
  device_id: string;
  device_path?: string;
  hdidle_status?: HdIdleDeviceStatus;
  smart_health?: SmartHealthStatus;
  smart_info?: SmartInfo;
};
export type DiskIoStats = {
  device_description: string;
  device_name: string;
  read_iops: number;
  read_latency_ms: number;
  smart_data?: SmartStatus;
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
  hdidle_running: boolean;
  per_disk_info?: {
    [key: string]: PerDiskInfo;
  };
  per_disk_io: DiskIoStats[] | null;
  per_partition_info: {
    [key: string]: PerPartitionInfo[] | null;
  };
};
export type GlobalNicStats = {
  totalInboundTraffic: number;
  totalOutboundTraffic: number;
};
export type NicIoStats = {
  deviceMaxSpeed: number;
  deviceName: string;
  inboundTraffic: number;
  ip?: string;
  netmask?: string;
  outboundTraffic: number;
};
export type NetworkStats = {
  global: GlobalNicStats;
  perNicIO: NicIoStats[] | null;
};
export type ProcessStatus = {
  children: ProcessStatus[] | null;
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
  network_health: NetworkStats;
  samba_process_status: {
    [key: string]: ProcessStatus;
  };
  samba_status: SambaStatus;
  update_available: boolean;
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
export type IssueReportResponse = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  addon_logs?: string;
  database_dump?: string;
  github_url: string;
  issue_title: string;
  sanitized_addon_config?: string;
  sanitized_srat_config?: string;
};
export type IssueReportRequest = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  browser_info?: string;
  console_errors?: string[] | null;
  current_url?: string;
  description: string;
  include_addon_config: boolean;
  include_addon_logs: boolean;
  include_context_data: boolean;
  include_database_dump: boolean;
  include_srat_config: boolean;
  navigation_history?: string[] | null;
  problem_type: Problem_type;
  reproducing_steps: string;
  title: string;
};
export type IssueTemplateFieldAttr = {
  description?: string;
  label: string;
  multiple?: boolean;
  options?: string[] | null;
  placeholder?: string;
  render?: string;
};
export type IssueTemplateValidity = {
  required: boolean;
};
export type IssueTemplateField = {
  attributes: IssueTemplateFieldAttr;
  id: string;
  type: string;
  validations?: IssueTemplateValidity;
};
export type IssueTemplate = {
  body: IssueTemplateField[] | null;
  description: string;
  labels: string[] | null;
  name: string;
  title: string;
};
export type IssueTemplateResponse = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  error?: string;
  template: IssueTemplate;
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
  allow_guest?: boolean;
  allow_hosts?: string[];
  bind_all_interfaces?: boolean;
  compatibility_mode?: boolean;
  export_stats_to_ha?: boolean;
  ha_use_nfs?: boolean;
  hdidle_default_command_type?: Hdidle_default_command_type;
  hdidle_default_idle_time?: number;
  hdidle_default_power_condition?: number;
  hdidle_enabled?: boolean;
  hdidle_ignore_spin_down_detection?: boolean;
  hostname?: string;
  interfaces?: string[];
  local_master?: boolean;
  multi_channel?: boolean;
  smb_over_quic?: boolean;
  telemetry_mode?: Telemetry_mode;
  workgroup?: string;
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
  path: string;
  refresh_version?: number;
  root?: string;
  share?: SharedResource;
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
  [key: string]: unknown;
};
export type SharedResourceStatus = {
  is_ha_mounted?: boolean;
  is_valid?: boolean;
};
export type SharedResource = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  disabled?: boolean;
  guest_ok?: boolean;
  mount_point_data?: MountPointData;
  name?: string;
  recycle_bin_enabled?: boolean;
  ro_users?: User[] | null;
  status?: SharedResourceStatus;
  timemachine?: boolean;
  timemachine_max_size?: string;
  usage?: Usage;
  users?: User[] | null;
  veto_files?: string[];
  [key: string]: unknown;
};
export type SharedResourcePostData = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  disabled?: boolean;
  guest_ok?: boolean;
  name?: string;
  recycle_bin_enabled?: boolean;
  ro_users?: User[] | null;
  status?: SharedResourceStatus;
  timemachine?: boolean;
  timemachine_max_size?: string;
  usage?: Usage;
  users?: User[] | null;
  veto_files?: string[];
  [key: string]: unknown;
};
export type ErrorDataModel = {
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
export type BinaryAsset = {
  browser_download_url?: string;
  digest?: string;
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
export type UpdateProgress = {
  /** A URL to the JSON Schema for this object. */
  $schema?: string;
  error_message?: string;
  progress?: number;
  release_asset?: ReleaseAsset;
  update_process_state?: Update_process_state;
};
export type Partition = {
  device_path?: string;
  disk_id?: string;
  fs_type?: string;
  host_mount_point_data?: {
    [key: string]: MountPointData;
  };
  id?: string;
  legacy_device_name?: string;
  legacy_device_path?: string;
  mount_point_data?: {
    [key: string]: MountPointData;
  };
  name?: string;
  refresh_version?: number;
  size?: number;
  system?: boolean;
  uuid?: string;
};
export type Disk = {
  connection_bus?: string;
  device_path?: string;
  ejectable?: boolean;
  hdidle_device?: HdIdleDevice;
  id?: string;
  legacy_device_name?: string;
  legacy_device_path?: string;
  model?: string;
  partitions?: {
    [key: string]: Partition;
  };
  refresh_version?: number;
  removable?: boolean;
  revision?: string;
  seat?: string;
  serial?: string;
  size?: number;
  smart_info?: SmartInfo;
  vendor?: string;
};
export type PatchMountPointData = {
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
  path: string;
  refresh_version?: number;
  root?: string;
  time_machine_support?: Time_machine_support;
  type: Type;
  warnings?: string;
  [key: string]: unknown;
};
export enum Command_type {
  Scsi = "scsi",
  Ata = "ata",
}
export enum Enabled {
  Yes = "yes",
  Custom = "custom",
  No = "no",
}
export enum Op {
  Add = "add",
  Remove = "remove",
  Replace = "replace",
  Move = "move",
  Copy = "copy",
  Test = "test",
}
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
export enum Problem_type {
  FrontendUi = "frontend_ui",
  HaIntegration = "ha_integration",
  Addon = "addon",
  Samba = "samba",
}
export enum Hdidle_default_command_type {
  Scsi = "scsi",
  Ata = "ata",
}
export enum Telemetry_mode {
  Ask = "Ask",
  All = "All",
  Errors = "Errors",
  Disabled = "Disabled",
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
  Shares = "shares",
  DirtyDataTracker = "dirty_data_tracker",
  SmartTestStatus = "smart_test_status",
}
export enum Update_channel {
  None = "None",
  Develop = "Develop",
  Release = "Release",
  Prerelease = "Prerelease",
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
  useGetApiCapabilitiesQuery,
  useGetApiDiskByDiskIdHdidleConfigQuery,
  usePatchApiDiskByDiskIdHdidleConfigMutation,
  usePutApiDiskByDiskIdHdidleConfigMutation,
  useGetApiDiskByDiskIdHdidleInfoQuery,
  useGetApiDiskByDiskIdHdidleSupportQuery,
  usePostApiDiskByDiskIdSmartDisableMutation,
  usePostApiDiskByDiskIdSmartEnableMutation,
  useGetApiDiskByDiskIdSmartHealthQuery,
  useGetApiDiskByDiskIdSmartInfoQuery,
  useGetApiDiskByDiskIdSmartStatusQuery,
  useGetApiDiskByDiskIdSmartTestQuery,
  usePostApiDiskByDiskIdSmartTestAbortMutation,
  usePostApiDiskByDiskIdSmartTestStartMutation,
  useGetApiFilesystemsQuery,
  usePostApiHdidleStartMutation,
  usePostApiHdidleStopMutation,
  useGetApiHealthQuery,
  useGetApiHostnameQuery,
  useGetApiIssuesQuery,
  usePostApiIssuesMutation,
  usePostApiIssuesReportMutation,
  useGetApiIssuesTemplateQuery,
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
  useDeleteApiVolumeMutation,
  usePostApiVolumeMountMutation,
  usePatchApiVolumeSettingsMutation,
  useGetApiVolumesQuery,
} = injectedRtkApi;
