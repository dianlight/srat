/* eslint-disable */
/* tslint:disable */
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface BlockDisk {
  /** BusPath is the filepath to the bus for this disk. */
  bus_path?: string;
  /** DriveType is the category of disk drive for this disk. */
  drive_type?: BlockDriveType;
  /** Model is the model number of the disk. */
  model?: string;
  /** Name contains a short name for the disk, e.g. `sda` */
  name?: string;
  /**
   * Partitions contains an array of pointers to `Partition` structs, one for
   * each partition on the disk.
   */
  partitions?: BlockPartition[];
  /**
   * PhysicalBlockSizeBytes is the size, in bytes, of the physical blocks in
   * this disk. This is typically the minimum amount of data that can be
   * written to a disk in a single write operation.
   */
  physical_block_size_bytes?: number;
  /** IsRemovable indicates if the disk drive is removable. */
  removable?: boolean;
  /** SerialNumber is the serial number of the disk. */
  serial_number?: string;
  /** SizeBytes contains the total amount of storage, in bytes, for this disk */
  size_bytes?: number;
  /**
   * StorageController is the category of storage controller used by the
   * disk.
   */
  storage_controller?: BlockStorageController;
  /** Vendor is the manufacturer of the disk. */
  vendor?: string;
  /**
   * WWN is the World-wide Name of the disk.
   * See: https://en.wikipedia.org/wiki/World_Wide_Name
   */
  wwn?: string;
  /**
   * WWNNoExtension is the World-wide Name of the disk with any vendor
   * extensions excluded.
   * See: https://en.wikipedia.org/wiki/World_Wide_Name
   */
  wwnNoExtension?: string;
}

export enum BlockDriveType {
  DriveTypeUnknown = 0,
  DriveTypeHDD = 1,
  DriveTypeFDD = 2,
  DriveTypeODD = 3,
  DriveTypeSSD = 4,
  DriveTypeVirtual = 5,
  DRIVE_TYPE_UNKNOWN = 0,
  DRIVE_TYPE_HDD = 1,
  DRIVE_TYPE_FDD = 2,
  DRIVE_TYPE_ODD = 3,
  DRIVE_TYPE_SSD = 4,
  DRIVE_TYPE_VIRTUAL = 5,
}

export interface BlockInfo {
  /**
   * Disks contains an array of pointers to `Disk` structs, one for each disk
   * drive on the host system.
   */
  disks?: BlockDisk[];
  /**
   * TotalSizeBytes contains the total amount of storage, in bytes, on the
   * host system.
   */
  total_size_bytes?: number;
}

export interface BlockPartition {
  /**
   * FilesystemLabel is the label of the filesystem contained on the
   * partition. On Linux, this is derived from the `ID_FS_NAME` udev entry.
   */
  filesystem_label?: string;
  /**
   * Label is the human-readable label given to the partition. On Linux, this
   * is derived from the `ID_PART_ENTRY_NAME` udev entry.
   */
  label?: string;
  /** MountPoint is the path where this partition is mounted. */
  mount_point?: string;
  /** Name is the system name given to the partition, e.g. "sda1". */
  name?: string;
  /** IsReadOnly indicates if the partition is marked read-only. */
  read_only?: boolean;
  /**
   * SizeBytes contains the total amount of storage, in bytes, this partition
   * can consume.
   */
  size_bytes?: number;
  /** Type contains the type of the partition. */
  type?: string;
  /**
   * UUID is the universally-unique identifier (UUID) for the partition.
   * This will be volume UUID on Darwin, PartUUID on linux, empty on Windows.
   */
  uuid?: string;
}

export enum BlockStorageController {
  StorageControllerUnknown = 0,
  StorageControllerIDE = 1,
  StorageControllerSCSI = 2,
  StorageControllerNVMe = 3,
  StorageControllerVirtIO = 4,
  StorageControllerMMC = 5,
  StorageControllerLoop = 6,
  STORAGE_CONTROLLER_UNKNOWN = 0,
  STORAGE_CONTROLLER_IDE = 1,
  STORAGE_CONTROLLER_SCSI = 2,
  STORAGE_CONTROLLER_NVME = 3,
  STORAGE_CONTROLLER_VIRTIO = 4,
  STORAGE_CONTROLLER_MMC = 5,
  STORAGE_CONTROLLER_LOOP = 6,
}

export interface ConfigConfig {
  acl?: ConfigOptionsAcl[];
  allow_hosts?: string[];
  autodiscovery?: {
    disable_autoremove?: boolean;
    disable_discovery?: boolean;
    disable_persistent?: boolean;
  };
  automount?: boolean;
  available_disks_log?: boolean;
  bind_all_interfaces?: boolean;
  compatibility_mode?: boolean;
  currentFile?: string;
  docker_interface?: string;
  docker_net?: string;
  enable_smart?: boolean;
  hdd_idle_seconds?: number;
  interfaces?: string[];
  log_level?: string;
  meaning_of_life?: string;
  medialibrary?: {
    enable?: boolean;
    ssh_private_key?: string;
  };
  moredisks?: string[];
  mountoptions?: string[];
  mqtt_enable?: boolean;
  mqtt_host?: string;
  mqtt_nexgen_entities?: boolean;
  mqtt_password?: string;
  mqtt_port?: string;
  mqtt_topic?: string;
  mqtt_username?: string;
  multi_channel?: boolean;
  other_users?: ConfigUser[];
  password?: string;
  recyle_bin_enabled?: boolean;
  shares?: ConfigShares;
  update_channel?: ConfigUpdateChannel;
  username?: string;
  users?: ConfigUser[];
  version?: number;
  veto_files?: string[];
  workgroup?: string;
  wsdd?: boolean;
  wsdd2?: boolean;
}

export interface ConfigConfigSectionDirtySate {
  settings?: boolean;
  shares?: boolean;
  users?: boolean;
  volumes?: boolean;
}

export interface ConfigOptionsAcl {
  disabled?: boolean;
  ro_users?: string[];
  share?: string;
  timemachine?: boolean;
  usage?: string;
  users?: string[];
}

export interface ConfigShare {
  disabled?: boolean;
  fs?: string;
  name?: string;
  path?: string;
  ro_users?: string[];
  timemachine?: boolean;
  usage?: string;
  users?: string[];
}

export type ConfigShares = Record<string, ConfigShare>;

export enum ConfigUpdateChannel {
  Stable = "stable",
  Prerelease = "prerelease",
  None = "none",
}

export interface ConfigUser {
  password?: string;
  username?: string;
}

export interface GithubMatch {
  indices?: number[];
  text?: string;
}

export interface GithubOrganization {
  /** AdvancedSecurityAuditLogEnabled toggles whether the advanced security audit log is enabled. */
  advanced_security_enabled_for_new_repositories?: boolean;
  avatar_url?: string;
  billing_email?: string;
  blog?: string;
  collaborators?: number;
  company?: string;
  created_at?: GithubTimestamp;
  /**
   * DefaultRepoPermission can be one of: "read", "write", "admin", or "none". (Default: "read").
   * It is only used in OrganizationsService.Edit.
   */
  default_repository_permission?: string;
  /**
   * DefaultRepoSettings can be one of: "read", "write", "admin", or "none". (Default: "read").
   * It is only used in OrganizationsService.Get.
   */
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
  /**
   * MembersAllowedRepositoryCreationType denotes if organization members can create repositories
   * and the type of repositories they can create. Possible values are: "all", "private", or "none".
   *
   * Deprecated: Use MembersCanCreatePublicRepos, MembersCanCreatePrivateRepos, MembersCanCreateInternalRepos
   * instead. The new fields overrides the existing MembersAllowedRepositoryCreationType during 'edit'
   * operation and does not consider 'internal' repositories during 'get' operation
   */
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
}

export interface GithubPlan {
  collaborators?: number;
  filled_seats?: number;
  name?: string;
  private_repos?: number;
  seats?: number;
  space?: number;
}

export interface GithubReleaseAsset {
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
}

export interface GithubRepositoryRelease {
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
}

export interface GithubTeam {
  /**
   * Assignment identifies how a team was assigned to an organization role. Its
   * possible values are: "direct", "indirect", "mixed". This is only populated when
   * calling the ListTeamsAssignedToOrgRole method.
   */
  assignment?: string;
  description?: string;
  html_url?: string;
  id?: number;
  /**
   * LDAPDN is only available in GitHub Enterprise and when the team
   * membership is synchronized with LDAP.
   */
  ldap_dn?: string;
  members_count?: number;
  members_url?: string;
  name?: string;
  node_id?: string;
  organization?: GithubOrganization;
  parent?: GithubTeam;
  /** Permission specifies the default permission for repositories owned by the team. */
  permission?: string;
  /**
   * Permissions identifies the permissions that a team has on a given
   * repository. This is only populated when calling Repositories.ListTeams.
   */
  permissions?: Record<string, boolean>;
  /**
   * Privacy identifies the level of privacy this team should have.
   * Possible values are:
   *     secret - only visible to organization owners and members of this team
   *     closed - visible to all members of this organization
   * Default is "secret".
   */
  privacy?: string;
  repos_count?: number;
  repositories_url?: string;
  slug?: string;
  url?: string;
}

export interface GithubTextMatch {
  fragment?: string;
  matches?: GithubMatch[];
  object_type?: string;
  object_url?: string;
  property?: string;
}

export interface GithubTimestamp {
  "time.Time"?: string;
}

export interface GithubUser {
  /**
   * Assignment identifies how a user was assigned to an organization role. Its
   * possible values are: "direct", "indirect", "mixed". This is only populated when
   * calling the ListUsersAssignedToOrgRole method.
   */
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
  /**
   * InheritedFrom identifies the team that a user inherited their organization role
   * from. This is only populated when calling the ListUsersAssignedToOrgRole method.
   */
  inherited_from?: GithubTeam;
  ldap_dn?: string;
  location?: string;
  login?: string;
  name?: string;
  node_id?: string;
  organizations_url?: string;
  owned_private_repos?: number;
  /**
   * Permissions and RoleName identify the permissions and role that a user has on a given
   * repository. These are only populated when calling Repositories.ListCollaborators.
   */
  permissions?: Record<string, boolean>;
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
  /**
   * TextMatches is only populated from search results that request text matches
   * See: search.go and https://docs.github.com/rest/search/#text-match-metadata
   */
  text_matches?: GithubTextMatch[];
  total_private_repos?: number;
  twitter_username?: string;
  two_factor_authentication?: boolean;
  type?: string;
  updated_at?: GithubTimestamp;
  /** API URLs */
  url?: string;
}

export enum MainEventType {
  EventUpdate = "update",
  EventHeartbeat = "heartbeat",
  EventShare = "share",
  EventVolumes = "volumes",
  EventDirty = "dirty",
}

export interface MainGlobalConfig {
  allow_hosts?: string[];
  bind_all_interfaces?: boolean;
  compatibility_mode?: boolean;
  interfaces?: string[];
  log_level?: string;
  mountoptions?: string[];
  multi_channel?: boolean;
  recyle_bin_enabled?: boolean;
  update_channel?: ConfigUpdateChannel;
  veto_files?: string[];
  workgroup?: string;
}

export interface MainHealth {
  alive?: boolean;
  last_error?: string;
  read_only?: boolean;
  samba_pid?: number;
}

export enum MainMounDataFlag {
  MS_RDONLY = 1,
  MS_BIND = 4096,
  MS_LAZYTIME = 33554432,
  MS_NOEXEC = 8,
  MS_NOSUID = 2,
  MS_NOUSER = -2147483648,
  MS_RELATIME = 2097152,
  MS_SYNC = 4,
  MS_NOATIME = 1024,
  ReadOnlyMountPoindDataFlags = 1025,
}

export interface MainMountPointData {
  data?: string;
  flags?: MainMounDataFlag[];
  fstype?: string;
  label?: string;
  name?: string;
  path?: string;
}

export interface MainResponseError {
  body?: any;
  code?: number;
  error?: string;
}

export interface MainSRATReleaseAsset {
  arch?: GithubReleaseAsset;
  last_release?: GithubRepositoryRelease;
  update_status?: number;
}

export interface MainSambaProcessStatus {
  connections?: number;
  cpu_percent?: number;
  create_time?: string;
  is_running?: boolean;
  memory_percent?: number;
  name?: string;
  open_files?: number;
  pid?: number;
  status?: string[];
}

export interface NetInfo {
  /**
   * NICs is a slice of pointers to `NIC` structs describing the network
   * interface controllers (NICs) on the host system.
   */
  nics?: NetNIC[];
}

export interface NetNIC {
  /**
   * AvertisedFECModes is a slice of strings containing the advertised
   * (during auto-negotiation) Forward Error Correction (FEC) modes for this
   * NIC.
   */
  advertised_fec_modes?: string[];
  /**
   * AdvertiseLinkModes is a slice of strings containing the advertised
   * (during auto-negotiation) link modes of this NIC, e.g. "10baseT/Half",
   * "1000baseT/Full", etc.
   */
  advertised_link_modes?: string[];
  /**
   * Capabilities is a slice of pointers to `NICCapability` structs
   * describing a feature/capability of this NIC.
   */
  capabilities?: NetNICCapability[];
  /**
   * Duplex is a string indicating the current duplex setting of this NIC,
   * e.g. "Full"
   */
  duplex?: string;
  /**
   * IsVirtual is true if the NIC is entirely virtual/emulated, false
   * otherwise.
   */
  is_virtual?: boolean;
  /** MACAddress is the Media Access Control (MAC) address of this NIC. */
  mac_address?: string;
  /** Name is the string identifier the system gave this NIC. */
  name?: string;
  /**
   * PCIAddress is a pointer to the PCI address for this NIC, or nil if there
   * is no PCI address for this NIC.
   */
  pci_address?: string;
  /** Speed is a string describing the link speed of this NIC, e.g. "1000Mb/s" */
  speed?: string;
  /**
   * SupportedFECModes is a slice of strings containing the supported Forward
   * Error Correction (FEC) modes for this NIC.
   */
  supported_fec_modes?: string[];
  /**
   * SupportedLinkModes is a slice of strings containing the supported link
   * modes of this NIC, e.g. "10baseT/Half", "1000baseT/Full", etc.
   */
  supported_link_modes?: string[];
  /**
   * SupportedPorts is a slice of strings containing the supported physical
   * ports on this NIC, e.g. "Twisted Pair"
   */
  supported_ports?: string[];
}

export interface NetNICCapability {
  /**
   * CanEnable is true if the capability can be enabled on the NIC, false
   * otherwise.
   */
  can_enable?: boolean;
  /**
   * IsEnabled is true if the capability is currently enabled on the NIC,
   * false otherwise.
   */
  is_enabled?: boolean;
  /**
   * Name is the string name for the capability, e.g.
   * "tcp-segmentation-offload"
   */
  name?: string;
}

import type { AxiosInstance, AxiosRequestConfig, AxiosResponse, HeadersDefaults, ResponseType } from "axios";
import axios from "axios";

export type QueryParamsType = Record<string | number, any>;

export interface FullRequestParams extends Omit<AxiosRequestConfig, "data" | "params" | "url" | "responseType"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseType;
  /** request body */
  body?: unknown;
}

export type RequestParams = Omit<FullRequestParams, "body" | "method" | "query" | "path">;

export interface ApiConfig<SecurityDataType = unknown> extends Omit<AxiosRequestConfig, "data" | "cancelToken"> {
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<AxiosRequestConfig | void> | AxiosRequestConfig | void;
  secure?: boolean;
  format?: ResponseType;
}

export enum ContentType {
  Json = "application/json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
  Text = "text/plain",
}

export class HttpClient<SecurityDataType = unknown> {
  public instance: AxiosInstance;
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private secure?: boolean;
  private format?: ResponseType;

  constructor({ securityWorker, secure, format, ...axiosConfig }: ApiConfig<SecurityDataType> = {}) {
    this.instance = axios.create({ ...axiosConfig, baseURL: axiosConfig.baseURL || "" });
    this.secure = secure;
    this.format = format;
    this.securityWorker = securityWorker;
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  protected mergeRequestParams(params1: AxiosRequestConfig, params2?: AxiosRequestConfig): AxiosRequestConfig {
    const method = params1.method || (params2 && params2.method);

    return {
      ...this.instance.defaults,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...((method && this.instance.defaults.headers[method.toLowerCase() as keyof HeadersDefaults]) || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  protected stringifyFormItem(formItem: unknown) {
    if (typeof formItem === "object" && formItem !== null) {
      return JSON.stringify(formItem);
    } else {
      return `${formItem}`;
    }
  }

  protected createFormData(input: Record<string, unknown>): FormData {
    if (input instanceof FormData) {
      return input;
    }
    return Object.keys(input || {}).reduce((formData, key) => {
      const property = input[key];
      const propertyContent: any[] = property instanceof Array ? property : [property];

      for (const formItem of propertyContent) {
        const isFileType = formItem instanceof Blob || formItem instanceof File;
        formData.append(key, isFileType ? formItem : this.stringifyFormItem(formItem));
      }

      return formData;
    }, new FormData());
  }

  public request = async <T = any, _E = any>({
    secure,
    path,
    type,
    query,
    format,
    body,
    ...params
  }: FullRequestParams): Promise<AxiosResponse<T>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const responseFormat = format || this.format || undefined;

    if (type === ContentType.FormData && body && body !== null && typeof body === "object") {
      body = this.createFormData(body as Record<string, unknown>);
    }

    if (type === ContentType.Text && body && body !== null && typeof body !== "string") {
      body = JSON.stringify(body);
    }

    return this.instance.request({
      ...requestParams,
      headers: {
        ...(requestParams.headers || {}),
        ...(type ? { "Content-Type": type } : {}),
      },
      params: query,
      responseType: responseFormat,
      data: body,
      url: path,
    });
  };
}

/**
 * @title SRAT API
 * @version 1.0
 * @license Apache 2.0 (http://www.apache.org/licenses/LICENSE-2.0.html)
 * @contact Lucio Tarantino <lucio.tarantino@gmail.com> (https://github.com/dianlight)
 *
 * This are samba rest admin API
 */
export class Api<SecurityDataType extends unknown> extends HttpClient<SecurityDataType> {
  admin = {
    /**
     * @description get the admin user
     *
     * @tags user
     * @name UserList
     * @summary Get the admin user
     * @request GET:/admin/user
     */
    userList: (params: RequestParams = {}) =>
      this.request<ConfigUser, MainResponseError>({
        path: `/admin/user`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description update admin user
     *
     * @tags user
     * @name UserUpdate
     * @summary Update admin user
     * @request PUT:/admin/user
     */
    userUpdate: (user: ConfigUser, params: RequestParams = {}) =>
      this.request<ConfigUser, MainResponseError>({
        path: `/admin/user`,
        method: "PUT",
        body: user,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description update admin user
     *
     * @tags user
     * @name UserPartialUpdate
     * @summary Update admin user
     * @request PATCH:/admin/user
     */
    userPartialUpdate: (user: ConfigUser, params: RequestParams = {}) =>
      this.request<ConfigUser, MainResponseError>({
        path: `/admin/user`,
        method: "PATCH",
        body: user,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  config = {
    /**
     * @description Save dirty changes to the disk
     *
     * @tags samba
     * @name ConfigUpdate
     * @summary Persiste the current samba config
     * @request PUT:/config
     */
    configUpdate: (params: RequestParams = {}) =>
      this.request<ConfigConfig, MainResponseError>({
        path: `/config`,
        method: "PUT",
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Revert to the last saved samba config
     *
     * @tags samba
     * @name ConfigDelete
     * @summary Rollback the current samba config
     * @request DELETE:/config
     */
    configDelete: (params: RequestParams = {}) =>
      this.request<ConfigConfig, MainResponseError>({
        path: `/config`,
        method: "DELETE",
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Save dirty changes to the disk
     *
     * @tags samba
     * @name ConfigPartialUpdate
     * @summary Persiste the current samba config
     * @request PATCH:/config
     */
    configPartialUpdate: (params: RequestParams = {}) =>
      this.request<ConfigConfig, MainResponseError>({
        path: `/config`,
        method: "PATCH",
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  events = {
    /**
     * @description Return a list of available WSChannel events
     *
     * @tags system
     * @name EventsList
     * @summary WSChannelEventsList
     * @request GET:/events
     */
    eventsList: (params: RequestParams = {}) =>
      this.request<MainEventType[], string>({
        path: `/events`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  global = {
    /**
     * @description Get the configuration for the global samba settings
     *
     * @tags samba
     * @name GlobalList
     * @summary Get the configuration for the global samba settings
     * @request GET:/global
     */
    globalList: (params: RequestParams = {}) =>
      this.request<MainGlobalConfig, MainResponseError>({
        path: `/global`,
        method: "GET",
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Update the configuration for the global samba settings
     *
     * @tags samba
     * @name GlobalUpdate
     * @summary Update the configuration for the global samba settings
     * @request PUT:/global
     */
    globalUpdate: (config: MainGlobalConfig, params: RequestParams = {}) =>
      this.request<MainGlobalConfig, MainResponseError>({
        path: `/global`,
        method: "PUT",
        body: config,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Update the configuration for the global samba settings
     *
     * @tags samba
     * @name GlobalPartialUpdate
     * @summary Update the configuration for the global samba settings
     * @request PATCH:/global
     */
    globalPartialUpdate: (config: MainGlobalConfig, params: RequestParams = {}) =>
      this.request<MainGlobalConfig, MainResponseError>({
        path: `/global`,
        method: "PATCH",
        body: config,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  health = {
    /**
     * @description HealthCheck
     *
     * @tags system
     * @name HealthList
     * @summary HealthCheck
     * @request GET:/health
     */
    healthList: (params: RequestParams = {}) =>
      this.request<MainHealth, MainResponseError>({
        path: `/health`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  nics = {
    /**
     * @description Return all network interfaces
     *
     * @tags system
     * @name NicsList
     * @summary GetNICsHandler
     * @request GET:/nics
     */
    nicsList: (params: RequestParams = {}) =>
      this.request<NetInfo, MainResponseError>({
        path: `/nics`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  restart = {
    /**
     * @description Restart the server ( useful in development )
     *
     * @tags system
     * @name RestartUpdate
     * @summary RestartHandler
     * @request PUT:/restart
     */
    restartUpdate: (params: RequestParams = {}) =>
      this.request<void, MainResponseError>({
        path: `/restart`,
        method: "PUT",
        ...params,
      }),
  };
  samba = {
    /**
     * @description Get the generated samba config
     *
     * @tags samba
     * @name SambaList
     * @summary Get the generated samba config
     * @request GET:/samba
     */
    sambaList: (params: RequestParams = {}) =>
      this.request<string, MainResponseError>({
        path: `/samba`,
        method: "GET",
        type: ContentType.Json,
        ...params,
      }),

    /**
     * @description Write the samba config and send signal ro restart
     *
     * @tags samba
     * @name ApplyUpdate
     * @summary Write the samba config and send signal ro restart
     * @request PUT:/samba/apply
     */
    applyUpdate: (params: RequestParams = {}) =>
      this.request<void, MainResponseError>({
        path: `/samba/apply`,
        method: "PUT",
        type: ContentType.Json,
        ...params,
      }),

    /**
     * @description Get the current samba process status
     *
     * @tags samba
     * @name StatusList
     * @summary Get the current samba process status
     * @request GET:/samba/status
     */
    statusList: (params: RequestParams = {}) =>
      this.request<MainSambaProcessStatus, MainResponseError>({
        path: `/samba/status`,
        method: "GET",
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  share = {
    /**
     * @description create e new share
     *
     * @tags share
     * @name ShareCreate
     * @summary Create a share
     * @request POST:/share
     */
    shareCreate: (share: ConfigShare, params: RequestParams = {}) =>
      this.request<ConfigShare, MainResponseError>({
        path: `/share`,
        method: "POST",
        body: share,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description get share by Name
     *
     * @tags share
     * @name ShareDetail
     * @summary Get a share
     * @request GET:/share/{share_name}
     */
    shareDetail: (shareName: string, params: RequestParams = {}) =>
      this.request<ConfigShare, MainResponseError>({
        path: `/share/${shareName}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description update e new share
     *
     * @tags share
     * @name ShareUpdate
     * @summary Update a share
     * @request PUT:/share/{share_name}
     */
    shareUpdate: (shareName: string, share: ConfigShare, params: RequestParams = {}) =>
      this.request<ConfigShare, MainResponseError>({
        path: `/share/${shareName}`,
        method: "PUT",
        body: share,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description delere a share
     *
     * @tags share
     * @name ShareDelete
     * @summary Delere a share
     * @request DELETE:/share/{share_name}
     */
    shareDelete: (shareName: string, params: RequestParams = {}) =>
      this.request<void, MainResponseError>({
        path: `/share/${shareName}`,
        method: "DELETE",
        ...params,
      }),

    /**
     * @description update e new share
     *
     * @tags share
     * @name SharePartialUpdate
     * @summary Update a share
     * @request PATCH:/share/{share_name}
     */
    sharePartialUpdate: (shareName: string, share: ConfigShare, params: RequestParams = {}) =>
      this.request<ConfigShare, MainResponseError>({
        path: `/share/${shareName}`,
        method: "PATCH",
        body: share,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  shares = {
    /**
     * @description List all configured shares
     *
     * @tags share
     * @name SharesList
     * @summary List all configured shares
     * @request GET:/shares
     */
    sharesList: (params: RequestParams = {}) =>
      this.request<ConfigShares, MainResponseError>({
        path: `/shares`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  update = {
    /**
     * @description Start the update process
     *
     * @tags system
     * @name UpdateUpdate
     * @summary UpdateHandler
     * @request PUT:/update
     */
    updateUpdate: (params: RequestParams = {}) =>
      this.request<MainSRATReleaseAsset, MainResponseError>({
        path: `/update`,
        method: "PUT",
        format: "json",
        ...params,
      }),
  };
  user = {
    /**
     * @description create e new user
     *
     * @tags user
     * @name UserCreate
     * @summary Create a user
     * @request POST:/user
     */
    userCreate: (user: ConfigUser, params: RequestParams = {}) =>
      this.request<ConfigUser, MainResponseError>({
        path: `/user`,
        method: "POST",
        body: user,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description get user by Name
     *
     * @tags user
     * @name UserDetail
     * @summary Get a user
     * @request GET:/user/{username}
     */
    userDetail: (username: string, params: RequestParams = {}) =>
      this.request<ConfigUser, MainResponseError>({
        path: `/user/${username}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description update e user
     *
     * @tags user
     * @name UserUpdate
     * @summary Update a user
     * @request PUT:/user/{username}
     */
    userUpdate: (username: string, user: ConfigUser, params: RequestParams = {}) =>
      this.request<ConfigUser, MainResponseError>({
        path: `/user/${username}`,
        method: "PUT",
        body: user,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description delete a user
     *
     * @tags user
     * @name UserDelete
     * @summary Delete a user
     * @request DELETE:/user/{username}
     */
    userDelete: (username: string, params: RequestParams = {}) =>
      this.request<void, MainResponseError>({
        path: `/user/${username}`,
        method: "DELETE",
        ...params,
      }),

    /**
     * @description update e user
     *
     * @tags user
     * @name UserPartialUpdate
     * @summary Update a user
     * @request PATCH:/user/{username}
     */
    userPartialUpdate: (username: string, user: ConfigUser, params: RequestParams = {}) =>
      this.request<ConfigUser, MainResponseError>({
        path: `/user/${username}`,
        method: "PATCH",
        body: user,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  users = {
    /**
     * @description List all configured users
     *
     * @tags user
     * @name UsersList
     * @summary List all configured users
     * @request GET:/users
     */
    usersList: (params: RequestParams = {}) =>
      this.request<ConfigUser[], MainResponseError>({
        path: `/users`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  volume = {
    /**
     * @description mount an existing volume
     *
     * @tags volume
     * @name MountCreate
     * @summary mount an existing volume
     * @request POST:/volume/{volume_name}/mount
     */
    mountCreate: (volumeName: string, mount_data: MainMountPointData, params: RequestParams = {}) =>
      this.request<MainMountPointData, MainResponseError>({
        path: `/volume/${volumeName}/mount`,
        method: "POST",
        body: mount_data,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Umount the selected volume
     *
     * @tags volume
     * @name MountDelete
     * @summary Umount the selected volume
     * @request DELETE:/volume/{volume_name}/mount
     */
    mountDelete: (
      volumeName: string,
      query: {
        /** Umount forcefully - forces an unmount regardless of currently open or otherwise used files within the file system to be unmounted. */
        force: boolean;
        /** Umount lazily - disallows future uses of any files below path -- i.e. it hides the file system mounted at path, but the file system itself is still active and any currently open files can continue to be used. When all references to files from this file system are gone, the file system will actually be unmounted. */
        lazy: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<void, MainResponseError>({
        path: `/volume/${volumeName}/mount`,
        method: "DELETE",
        query: query,
        ...params,
      }),
  };
  volumes = {
    /**
     * @description List all available volumes
     *
     * @tags volume
     * @name VolumesList
     * @summary List all available volumes
     * @request GET:/volumes
     */
    volumesList: (params: RequestParams = {}) =>
      this.request<BlockInfo, MainResponseError>({
        path: `/volumes`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  ws = {
    /**
     * @description Open the WSChannel
     *
     * @tags system
     * @name GetWs
     * @summary WSChannel
     * @request GET:/ws
     */
    getWs: (params: RequestParams = {}) =>
      this.request<ConfigConfigSectionDirtySate, MainResponseError>({
        path: `/ws`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
}
