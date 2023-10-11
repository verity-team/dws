import { Nullable } from "../types";

export enum HttpMethod {
  GET = "GET",
  POST = "POST",
}

export interface RequestConfig {
  host: string;
  timeout: number;
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
  body?: any
): Promise<Nullable<Response>> => {
  const { host, timeout } = config;

  // Generate request headers
  const headers = getDefaultHeaders();

  // Use signal to avoid running the request for too long
  // Docs for canceling fetch API request
  // https://javascript.info/fetch-abort
  const controller = new AbortController();
  const timerId = setTimeout(() => controller.abort(), timeout);

  const requestConfig = {
    method,
    headers,
    body: body == null ? null : JSON.stringify(body),
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
  body?: any
): Promise<Nullable<Response>> => {
  // Read configurations

  // Endpoint should be the same with current host
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

  return baseRequest(url, method, { host: apiHost, timeout }, body);
};

export const clientBaseRequest = async (
  url: string,
  method: HttpMethod,
  body?: any
): Promise<Nullable<Response>> => {
  // Read configurations

  // Endpoint should be the same with current host
  const apiHost = window.location.origin;
  if (apiHost == null) {
    console.log("API_URL not set");
    return null;
  }

  let timeout: number = Number(process.env.NEXT_PUBLIC_API_TIMEOUT);
  if (isNaN(timeout)) {
    console.log("API_TIMEOUT not set. Using fallback value");
    timeout = 10000;
  }

  return baseRequest(url, method, { host: apiHost, timeout }, body);
};
