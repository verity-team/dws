export type HttpMethod = "GET" | "POST" | "PATCH" | "DELETE";

interface RequestConfig {
  path: string;
  headers?: Headers;
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
  { path, headers, payload, json }: RequestConfig
): Promise<Response> => {
  let requestHeaders = headers;
  if (requestHeaders == null && json) {
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
    path,
    "with config",
    JSON.stringify(requestConfig, null, 2)
  );

  try {
    const response = await fetch(path, requestConfig);
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

  return baseRequest(method, { ...config, path });
};

// Request to Next.js route handler
export const baseNextClientRequest = async (
  method: HttpMethod,
  config: RequestConfig
): Promise<Response> => {
  const host = window.location.host;
  const path = `${host}${config.path}`;

  return baseRequest(method, { ...config, path });
};

// Request to Donation website (DWS) server
export const baseNextServerRequest = async (
  method: HttpMethod,
  config: RequestConfig
): Promise<Response> => {
  const host = process.env.DWS_API_URL;
  const path = `${host}${config.path}`;

  return baseRequest(method, { ...config, path });
};
