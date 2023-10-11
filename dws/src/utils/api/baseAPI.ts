import { Nullable } from "../types";

export enum HttpMethod {
  GET = "GET",
  POST = "POST",
}

const getDefaultHeaders = () => {
  const headers = new Headers();

  headers.append("Content-Type", "application/json");
  headers.append("Accept", "application/json");

  return headers;
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
export const sendBaseRequest = async (
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
    `Requesting to ${apiHost}${url} with config ${JSON.stringify(
      requestConfig
    )}`
  );

  const response = await fetch(`${apiHost}${url}`, requestConfig);
  clearTimeout(timerId);
  return response;
};
