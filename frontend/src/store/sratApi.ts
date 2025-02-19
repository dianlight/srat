import { emptySplitApi as api } from "./emptyApi";
const injectedRtkApi = api.injectEndpoints({
  endpoints: (build) => ({
    getFilesystems: build.query<
      GetFilesystemsApiResponse,
      GetFilesystemsApiArg
    >({
      query: () => ({ url: `/filesystems` }),
    }),
    getHealth: build.query<GetHealthApiResponse, GetHealthApiArg>({
      query: () => ({ url: `/health` }),
    }),
    getNics: build.query<GetNicsApiResponse, GetNicsApiArg>({
      query: () => ({ url: `/nics` }),
    }),
    putRestart: build.mutation<PutRestartApiResponse, PutRestartApiArg>({
      query: () => ({ url: `/restart`, method: "PUT" }),
    }),
    putSambaApply: build.mutation<
      PutSambaApplyApiResponse,
      PutSambaApplyApiArg
    >({
      query: () => ({ url: `/samba/apply`, method: "PUT" }),
    }),
    getSambaConfig: build.query<
      GetSambaConfigApiResponse,
      GetSambaConfigApiArg
    >({
      query: () => ({ url: `/samba/config` }),
    }),
    getSettings: build.query<GetSettingsApiResponse, GetSettingsApiArg>({
      query: () => ({ url: `/settings` }),
    }),
    putSettings: build.mutation<PutSettingsApiResponse, PutSettingsApiArg>({
      query: (queryArg) => ({
        url: `/settings`,
        method: "PUT",
        body: queryArg.dtoSettings,
      }),
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
    }),
    postShare: build.mutation<PostShareApiResponse, PostShareApiArg>({
      query: (queryArg) => ({
        url: `/share`,
        method: "POST",
        body: queryArg.dtoSharedResource,
      }),
    }),
    getShareByShareName: build.query<
      GetShareByShareNameApiResponse,
      GetShareByShareNameApiArg
    >({
      query: (queryArg) => ({ url: `/share/${queryArg.shareName}` }),
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
    }),
    deleteShareByShareName: build.mutation<
      DeleteShareByShareNameApiResponse,
      DeleteShareByShareNameApiArg
    >({
      query: (queryArg) => ({
        url: `/share/${queryArg.shareName}`,
        method: "DELETE",
      }),
    }),
    getShares: build.query<GetSharesApiResponse, GetSharesApiArg>({
      query: () => ({ url: `/shares` }),
    }),
    getSse: build.query<GetSseApiResponse, GetSseApiArg>({
      query: () => ({ url: `/sse` }),
    }),
    getSseEvents: build.query<GetSseEventsApiResponse, GetSseEventsApiArg>({
      query: () => ({ url: `/sse/events` }),
    }),
    putUpdate: build.mutation<PutUpdateApiResponse, PutUpdateApiArg>({
      query: () => ({ url: `/update`, method: "PUT" }),
    }),
    postUser: build.mutation<PostUserApiResponse, PostUserApiArg>({
      query: (queryArg) => ({
        url: `/user`,
        method: "POST",
        body: queryArg.dtoUser,
      }),
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
    }),
    deleteUserByUsername: build.mutation<
      DeleteUserByUsernameApiResponse,
      DeleteUserByUsernameApiArg
    >({
      query: (queryArg) => ({
        url: `/user/${queryArg.username}`,
        method: "DELETE",
      }),
    }),
    getUseradmin: build.query<GetUseradminApiResponse, GetUseradminApiArg>({
      query: () => ({ url: `/useradmin` }),
    }),
    putUseradmin: build.mutation<PutUseradminApiResponse, PutUseradminApiArg>({
      query: (queryArg) => ({
        url: `/useradmin`,
        method: "PUT",
        body: queryArg.dtoUser,
      }),
    }),
    getUsers: build.query<GetUsersApiResponse, GetUsersApiArg>({
      query: () => ({ url: `/users` }),
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
    }),
    getVolumes: build.query<GetVolumesApiResponse, GetVolumesApiArg>({
      query: () => ({ url: `/volumes` }),
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
export type GetSseApiResponse = unknown;
export type GetSseApiArg = void;
export type GetSseEventsApiResponse = /** status 200 OK */ DtoEventType[];
export type GetSseEventsApiArg = void;
export type PutUpdateApiResponse = /** status 200 OK */ DtoReleaseAsset;
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
  aliveTime?: string;
  dirty_tracking?: DtoDataDirtyTracker;
  last_error?: string;
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
export type GithubTimestamp = {
  "time.Time"?: string;
};
export type GithubPlan = {
  collaborators?: number;
  filled_seats?: number;
  name?: string;
  private_repos?: number;
  seats?: number;
  space?: number;
};
export type GithubOrganization = {
  /** AdvancedSecurityAuditLogEnabled toggles whether the advanced security audit log is enabled. */
  advanced_security_enabled_for_new_repositories?: boolean;
  avatar_url?: string;
  billing_email?: string;
  blog?: string;
  collaborators?: number;
  company?: string;
  created_at?: GithubTimestamp;
  /** DefaultRepoPermission can be one of: "read", "write", "admin", or "none". (Default: "read").
    It is only used in OrganizationsService.Edit. */
  default_repository_permission?: string;
  /** DefaultRepoSettings can be one of: "read", "write", "admin", or "none". (Default: "read").
    It is only used in OrganizationsService.Get. */
  default_repository_settings?: string;
  /** DependabotAlertsEnabled toggles whether dependabot alerts are enabled. */
  dependabot_alerts_enabled_for_new_repositories?: boolean;
  /** DependabotSecurityUpdatesEnabled toggles whether dependabot security updates are enabled. */
  dependabot_security_updates_enabled_for_new_repositories?: boolean;
  /** DependabotGraphEnabledForNewRepos toggles whether dependabot graph is enabled on new repositories. */
  dependency_graph_enabled_for_new_repositories?: boolean;
  description?: string;
  disk_usage?: number;
  email?: string;
  events_url?: string;
  followers?: number;
  following?: number;
  has_organization_projects?: boolean;
  has_repository_projects?: boolean;
  hooks_url?: string;
  html_url?: string;
  id?: number;
  is_verified?: boolean;
  issues_url?: string;
  location?: string;
  login?: string;
  /** MembersAllowedRepositoryCreationType denotes if organization members can create repositories
    and the type of repositories they can create. Possible values are: "all", "private", or "none".
    
    Deprecated: Use MembersCanCreatePublicRepos, MembersCanCreatePrivateRepos, MembersCanCreateInternalRepos
    instead. The new fields overrides the existing MembersAllowedRepositoryCreationType during 'edit'
    operation and does not consider 'internal' repositories during 'get' operation */
  members_allowed_repository_creation_type?: string;
  members_can_create_internal_repositories?: boolean;
  /** MembersCanCreatePages toggles whether organization members can create GitHub Pages sites. */
  members_can_create_pages?: boolean;
  /** MembersCanCreatePrivatePages toggles whether organization members can create private GitHub Pages sites. */
  members_can_create_private_pages?: boolean;
  members_can_create_private_repositories?: boolean;
  /** MembersCanCreatePublicPages toggles whether organization members can create public GitHub Pages sites. */
  members_can_create_public_pages?: boolean;
  /** https://developer.github.com/changes/2019-12-03-internal-visibility-changes/#rest-v3-api */
  members_can_create_public_repositories?: boolean;
  /** MembersCanCreateRepos default value is true and is only used in Organizations.Edit. */
  members_can_create_repositories?: boolean;
  /** MembersCanForkPrivateRepos toggles whether organization members can fork private organization repositories. */
  members_can_fork_private_repositories?: boolean;
  members_url?: string;
  name?: string;
  node_id?: string;
  owned_private_repos?: number;
  plan?: GithubPlan;
  private_gists?: number;
  public_gists?: number;
  public_members_url?: string;
  public_repos?: number;
  repos_url?: string;
  /** SecretScanningEnabled toggles whether secret scanning is enabled on new repositories. */
  secret_scanning_enabled_for_new_repositories?: boolean;
  /** SecretScanningPushProtectionEnabledForNewRepos toggles whether secret scanning push protection is enabled on new repositories. */
  secret_scanning_push_protection_enabled_for_new_repositories?: boolean;
  /** SecretScanningValidityChecksEnabled toggles whether secret scanning validity check is enabled. */
  secret_scanning_validity_checks_enabled?: boolean;
  total_private_repos?: number;
  twitter_username?: string;
  two_factor_requirement_enabled?: boolean;
  type?: string;
  updated_at?: GithubTimestamp;
  /** API URLs */
  url?: string;
  /** WebCommitSignoffRequire toggles */
  web_commit_signoff_required?: boolean;
};
export type GithubTeam = {
  /** Assignment identifies how a team was assigned to an organization role. Its
    possible values are: "direct", "indirect", "mixed". This is only populated when
    calling the ListTeamsAssignedToOrgRole method. */
  assignment?: string;
  description?: string;
  html_url?: string;
  id?: number;
  /** LDAPDN is only available in GitHub Enterprise and when the team
    membership is synchronized with LDAP. */
  ldap_dn?: string;
  members_count?: number;
  members_url?: string;
  name?: string;
  node_id?: string;
  organization?: GithubOrganization;
  parent?: GithubTeam;
  /** Permission specifies the default permission for repositories owned by the team. */
  permission?: string;
  /** Permissions identifies the permissions that a team has on a given
    repository. This is only populated when calling Repositories.ListTeams. */
  permissions?: {
    [key: string]: boolean;
  };
  /** Privacy identifies the level of privacy this team should have.
    Possible values are:
        secret - only visible to organization owners and members of this team
        closed - visible to all members of this organization
    Default is "secret". */
  privacy?: string;
  repos_count?: number;
  repositories_url?: string;
  slug?: string;
  url?: string;
};
export type GithubMatch = {
  indices?: number[];
  text?: string;
};
export type GithubTextMatch = {
  fragment?: string;
  matches?: GithubMatch[];
  object_type?: string;
  object_url?: string;
  property?: string;
};
export type GithubUser = {
  /** Assignment identifies how a user was assigned to an organization role. Its
    possible values are: "direct", "indirect", "mixed". This is only populated when
    calling the ListUsersAssignedToOrgRole method. */
  assignment?: string;
  avatar_url?: string;
  bio?: string;
  blog?: string;
  collaborators?: number;
  company?: string;
  created_at?: GithubTimestamp;
  disk_usage?: number;
  email?: string;
  events_url?: string;
  followers?: number;
  followers_url?: string;
  following?: number;
  following_url?: string;
  gists_url?: string;
  gravatar_id?: string;
  hireable?: boolean;
  html_url?: string;
  id?: number;
  /** InheritedFrom identifies the team that a user inherited their organization role
    from. This is only populated when calling the ListUsersAssignedToOrgRole method. */
  inherited_from?: GithubTeam;
  ldap_dn?: string;
  location?: string;
  login?: string;
  name?: string;
  node_id?: string;
  organizations_url?: string;
  owned_private_repos?: number;
  /** Permissions and RoleName identify the permissions and role that a user has on a given
    repository. These are only populated when calling Repositories.ListCollaborators. */
  permissions?: {
    [key: string]: boolean;
  };
  plan?: GithubPlan;
  private_gists?: number;
  public_gists?: number;
  public_repos?: number;
  received_events_url?: string;
  repos_url?: string;
  role_name?: string;
  site_admin?: boolean;
  starred_url?: string;
  subscriptions_url?: string;
  suspended_at?: GithubTimestamp;
  /** TextMatches is only populated from search results that request text matches
    See: search.go and https://docs.github.com/rest/search/#text-match-metadata */
  text_matches?: GithubTextMatch[];
  total_private_repos?: number;
  twitter_username?: string;
  two_factor_authentication?: boolean;
  type?: string;
  updated_at?: GithubTimestamp;
  /** API URLs */
  url?: string;
};
export type GithubReleaseAsset = {
  browser_download_url?: string;
  content_type?: string;
  created_at?: GithubTimestamp;
  download_count?: number;
  id?: number;
  label?: string;
  name?: string;
  node_id?: string;
  size?: number;
  state?: string;
  updated_at?: GithubTimestamp;
  uploader?: GithubUser;
  url?: string;
};
export type GithubRepositoryRelease = {
  assets?: GithubReleaseAsset[];
  assets_url?: string;
  author?: GithubUser;
  body?: string;
  created_at?: GithubTimestamp;
  discussion_category_name?: string;
  draft?: boolean;
  /** The following fields are not used in EditRelease: */
  generate_release_notes?: boolean;
  html_url?: string;
  /** The following fields are not used in CreateRelease or EditRelease: */
  id?: number;
  /** MakeLatest can be one of: "true", "false", or "legacy". */
  make_latest?: string;
  name?: string;
  node_id?: string;
  prerelease?: boolean;
  published_at?: GithubTimestamp;
  tag_name?: string;
  tarball_url?: string;
  target_commitish?: string;
  upload_url?: string;
  url?: string;
  zipball_url?: string;
};
export type DtoReleaseAsset = {
  arch?: GithubReleaseAsset;
  last_release?: GithubRepositoryRelease;
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
