"use client";

import { getExponentialWaitTime } from "@/utils/utils";
import { Revalidator } from "swr";
import { clientBaseRequest, HttpMethod } from "../baseAPI";
import { CustomError } from "../types";

export const fetcher = async (url: string) => {
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
