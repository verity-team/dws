import { Nullable, Maybe } from "@/utils";
import { Revalidator } from "swr";
import { getExponentialWaitTime } from "./utils";

export enum HttpMethod {
  GET = "GET",
  POST = "POST",
}

export interface RequestConfig {
  host: string;
  timeout: number;
}

export interface FailedResponse {
  code: string;
  message: string;
}

export interface CustomError extends Error {
  info: FailedResponse;
  status: number;
}

export const getDefaultHeaders = () => {
  const headers = new Headers();

  headers.append("Content-Type", "application/json");
  headers.append("Accept", "application/json");

  return headers;
};

export const baseRequest = async (
  url: string,
  method: HttpMethod,
  config: RequestConfig,
  body?: any,
  headers?: Headers
): Promise<Nullable<Response>> => {
  const { host, timeout } = config;

  headers = headers ?? getDefaultHeaders();
  const requestHeaders: Record<string, string> = {};
  for (let [key, value] of headers.entries()) {
    requestHeaders[key] = value;
  }

  // Use signal to avoid running the request for too long
  // Docs for canceling fetch API request
  // https://javascript.info/fetch-abort
  const controller = new AbortController();
  const timerId = setTimeout(() => controller.abort(), timeout);

  const requestConfig = {
    method,
    headers: requestHeaders,
    body: body == null ? null : JSON.stringify(body),
    signal: controller.signal,
  };

  console.log(`Requesting to ${host}${url}`);

  const response = await fetch(`${host}${url}`, requestConfig);
  clearTimeout(timerId);
  return response;
};

export const baseFormRequest = async (
  url: string,
  config: RequestConfig,
  body: FormData
): Promise<Maybe<Response>> => {
  const { host, timeout } = config;

  // Use signal to avoid running the request for too long
  // Docs for canceling fetch API request
  // https://javascript.info/fetch-abort
  const controller = new AbortController();
  const timerId = setTimeout(() => controller.abort(), timeout);

  const requestConfig = {
    method: "POST",
    headers: new Headers(),
    body,
    signal: controller.signal,
  };

  console.log(
    `Requesting to ${host}${url} with config ${JSON.stringify(requestConfig)}`
  );

  const response = await fetch(`${host}${url}`, requestConfig);
  clearTimeout(timerId);
  return response;
};

/**
 *
 * @param url
 * @param method
 * @param body
 * @description Base request for Next.js backend to make to real backend services
 *
 * @returns Promise<any> (should be object, whether the request is a success or a fail)
 */
export const serverBaseRequest = async (
  url: string,
  method: HttpMethod,
  body?: any,
  headers?: Headers
): Promise<Nullable<Response>> => {
  // Read configurations
  const apiHost = process.env.API_URL;
  if (apiHost == null) {
    console.log("API_URL not set");
    return null;
  }

  let timeout: number = Number(process.env.API_TIMEOUT);
  if (isNaN(timeout)) {
    console.log("API_TIMEOUT not set. Using fallback value");
    timeout = 10000;
  }

  return baseRequest(url, method, { host: apiHost, timeout }, body, headers);
};

export const serverFormRequest = async (url: string, body: FormData) => {
  // Read configurations
  const apiHost = process.env.API_URL;
  if (apiHost == null) {
    console.log("API_URL not set");
    return null;
  }

  let timeout: number = Number(process.env.API_TIMEOUT);
  if (isNaN(timeout)) {
    console.log("API_TIMEOUT not set. Using fallback value");
    timeout = 10000;
  }

  return baseFormRequest(url, { host: apiHost, timeout }, body);
};

export const clientBaseRequest = async (
  url: string,
  method: HttpMethod,
  body?: any,
  host?: string,
  auth?: boolean
): Promise<Nullable<Response>> => {
  // Read configurations

  // Endpoint should be the same with current host
  const apiHost = host ?? window.location.origin;
  if (apiHost == null) {
    console.log("API_URL not set");
    return null;
  }

  let timeout: number = Number(process.env.API_TIMEOUT);
  if (isNaN(timeout)) {
    console.log("API_TIMEOUT not set. Using fallback value");
    timeout = 10000;
  }

  if (auth) {
    const headers = getDefaultHeaders();
    const accessToken = localStorage.getItem("dws-at");
    if (!accessToken) {
      console.log("Cannot send auth request without access token");
      return null;
    }
    headers.append("Authorization", "Bearer " + accessToken);
    return baseRequest(url, method, { host: apiHost, timeout }, body, headers);
  }

  return baseRequest(url, method, { host: apiHost, timeout }, body);
};

export const clientFormRequest = async (
  url: string,
  body: FormData,
  host?: string
): Promise<Maybe<Response>> => {
  const apiHost = host ?? window.location.origin;
  if (apiHost == null) {
    console.log("API_URL not set");
    return null;
  }

  let timeout: number = Number(process.env.NEXT_PUBLIC_API_TIMEOUT);
  if (isNaN(timeout)) {
    console.log("API_TIMEOUT not set. Using fallback value");
    timeout = 10000;
  }

  return await baseFormRequest(url, { host: apiHost, timeout }, body);
};

export const swrFetcher = async (url: string) => {
  if (url == null || url.trim() === "") {
    return null;
  }

  const response = await clientBaseRequest(url, HttpMethod.GET);

  if (response == null) {
    throw new Error("Client setup is wrong. Check log for more info");
  }

  if (!response.ok) {
    const error = new Error() as any;
    try {
      error.info = await response.json();
    } catch {
      error.info = {
        code: "unknown",
        message: "Cannot parse response body. Check log for more info",
      };
    }

    error.status = response.status;
    throw error;
  }

  // Avoid error when parsing empty request body
  try {
    const result = await response.json();
    return result;
  } catch {
    // Ingore error and return empty object
    return null;
  }
};

export const handleErrorRetry = (
  err: CustomError,
  key: string,
  config: any,
  revalidate: Revalidator,
  { retryCount }: { retryCount: number }
) => {
  // Avoid retrying bad request or not found request
  if (err.status === 400 || err.status === 404) {
    return;
  }

  // Give up after 10 tries
  if (retryCount >= 5) {
    return;
  }

  setTimeout(
    () => revalidate({ retryCount }),
    getExponentialWaitTime(1000, retryCount)
  );
};
