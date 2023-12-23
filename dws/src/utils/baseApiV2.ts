import { method } from "lodash";
import path from "path";

export type HttpMethod = "GET" | "POST" | "PATCH" | "DELETE";

interface RequestConfig {
  path: string;
  headers?: Headers;
  fullPath?: boolean;
  json?: boolean;
  payload?: unknown;
}

export const getDefaultJsonHeaders = (): Headers => {
  const headers = new Headers();
  headers.append("Content-Type", "application/json");
  headers.append("Accept", "application/json");
  return headers;
};

export const baseRequest = async (
  method: HttpMethod,
  { path, headers, fullPath = false, payload, json }: RequestConfig
): Promise<Response> => {
  const host = process.env.NEXT_PUBLIC_API_HOST;
  if (host == null) {
    throw new Error("API host not set. Unable to make requests");
  }

  const requestPath = fullPath ? path : `${host}${path}`;
  let requestHeaders = null;
  if (headers == null && json) {
    requestHeaders = getDefaultJsonHeaders();
  } else {
    requestHeaders = new Headers();
  }

  let requestBody = undefined;
  if (json) {
    requestBody = JSON.stringify(payload);
  } else {
    requestBody = payload;
  }

  const controller = new AbortController();
  const requestConfig: RequestInit = {
    method,
    headers: requestHeaders,
    body: requestBody as BodyInit,
    signal: controller.signal,
  };

  const timeout = 10000;
  const timeoutId = setTimeout(() => {
    controller.abort();
  }, timeout);

  console.log(
    "Requesting to",
    requestPath,
    "with config",
    JSON.stringify(requestConfig, null, 2)
  );

  try {
    const response = await fetch(requestPath, requestConfig);
    return response;
  } catch (error) {
    console.log(error);
    throw error;
  } finally {
    clearTimeout(timeoutId);
  }
};

// Request to galactica server
export const baseGalacticaRequest = async (
  method: HttpMethod,
  config: RequestConfig
): Promise<Response> => {
  const host = process.env.NEXT_PUBLIC_GALACTICA_API_URL;
  const path = `${host}${config.path}`;

  return baseRequest(method, { ...config, path, fullPath: true });
};

// Request to Next.js route handler
export const baseNextClientRequest = async (
  method: HttpMethod,
  config: RequestConfig
): Promise<Response> => {
  if (config.fullPath) {
    throw new Error(
      "baseNextClientRequest function does not allow fullPath option"
    );
  }
  const host = window.location.host;
  const path = `${host}${config.path}`;

  return baseRequest(method, { ...config, path, fullPath: true });
};

// Request to Donation website (DWS) server
export const baseNextServerRequest = async (
  method: HttpMethod,
  config: RequestConfig
): Promise<Response> => {
  if (config.fullPath) {
    throw new Error(
      "baseNextServerRequest function does not allow fullPath option"
    );
  }
  const host = process.env.DWS_API_URL;
  const path = `${host}${config.path}`;

  return baseRequest(method, { ...config, path, fullPath: true });
};
