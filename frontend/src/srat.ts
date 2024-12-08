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

export interface DiskUsageStat {
  free?: number;
  fstype?: string;
  inodesFree?: number;
  inodesTotal?: number;
  inodesUsed?: number;
  inodesUsedPercent?: number;
  path?: string;
  total?: number;
  used?: number;
  usedPercent?: number;
}

export interface LsblkDevice {
  alignment?: number;
  children?: LsblkDevice[];
  fsavail?: number;
  fssize?: number;
  fstype?: string;
  /** percent that was used */
  fsusage?: number;
  fsused?: number;
  group?: string;
  hctl?: string;
  hotplug?: boolean;
  label?: string;
  model?: string;
  mountpoint?: string;
  name?: string;
  path?: string;
  pttype?: string;
  rev?: string;
  rm?: boolean;
  serial?: string;
  state?: string;
  subsystems?: string;
  tran?: string;
  type?: string;
  uuid?: string;
  vendor?: string;
  wwn?: string;
}

export interface MainHealth {
  alive?: boolean;
}

export interface MainResponseError {
  body?: any;
  code?: number;
  error?: string;
}

export interface MainShare {
  fs?: string;
  path?: string;
}

export type MainShares = Record<string, MainShare>;

export interface MainUser {
  password?: string;
  username?: string;
}

export interface MainVolume {
  device?: string;
  fstype?: string;
  label?: string;
  lsbk?: LsblkDevice;
  mountpoint?: string;
  opts?: string[];
  serial_number?: string;
  stats?: DiskUsageStat;
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
 * @contact Lucio Tarantino <lucio.tarantino@gmail.com>
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
      this.request<MainUser, MainResponseError>({
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
    userUpdate: (user: MainUser, params: RequestParams = {}) =>
      this.request<MainUser, MainResponseError>({
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
    userPartialUpdate: (user: MainUser, params: RequestParams = {}) =>
      this.request<MainUser, MainResponseError>({
        path: `/admin/user`,
        method: "PATCH",
        body: user,
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
  samba = {
    /**
     * @description Write the samba config and send signal ro restart
     *
     * @tags samba
     * @name ApplyUpdate
     * @summary Write the samba config and send signal ro restart
     * @request PUT:/samba/apply
     */
    applyUpdate: (params: RequestParams = {}) =>
      this.request<number[], MainResponseError>({
        path: `/samba/apply`,
        method: "PUT",
        type: ContentType.Json,
        format: "json",
        ...params,
      }),
  };
  share = {
    /**
     * @description get share by Name
     *
     * @tags share
     * @name ShareDetail
     * @summary Get a share
     * @request GET:/share/{share_name}
     */
    shareDetail: (shareName: string, params: RequestParams = {}) =>
      this.request<MainShare, MainResponseError>({
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
    shareUpdate: (shareName: string, share: MainShare, params: RequestParams = {}) =>
      this.request<MainShare, MainResponseError>({
        path: `/share/${shareName}`,
        method: "PUT",
        body: share,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description create e new share
     *
     * @tags share
     * @name ShareCreate
     * @summary Create a share
     * @request POST:/share/{share_name}
     */
    shareCreate: (shareName: string, share: MainShare, params: RequestParams = {}) =>
      this.request<MainShare, MainResponseError>({
        path: `/share/${shareName}`,
        method: "POST",
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
    sharePartialUpdate: (shareName: string, share: MainShare, params: RequestParams = {}) =>
      this.request<MainShare, MainResponseError>({
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
      this.request<MainShares, MainResponseError>({
        path: `/shares`,
        method: "GET",
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
    userCreate: (user: MainUser, params: RequestParams = {}) =>
      this.request<MainUser, MainResponseError>({
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
      this.request<MainUser, MainResponseError>({
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
    userUpdate: (username: string, user: MainUser, params: RequestParams = {}) =>
      this.request<MainUser, MainResponseError>({
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
    userPartialUpdate: (username: string, user: MainUser, params: RequestParams = {}) =>
      this.request<MainUser, MainResponseError>({
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
      this.request<MainUser[], MainResponseError>({
        path: `/users`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
  volume = {
    /**
     * @description get a volume by Name
     *
     * @tags volume
     * @name VolumeDetail
     * @summary Get a volume
     * @request GET:/volume/{volume_name}
     */
    volumeDetail: (volumeName: string, params: RequestParams = {}) =>
      this.request<MainVolume, MainResponseError>({
        path: `/volume/${volumeName}`,
        method: "GET",
        format: "json",
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
      this.request<MainVolume[], MainResponseError>({
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
      this.request<void, MainResponseError>({
        path: `/ws`,
        method: "GET",
        ...params,
      }),
  };
}
