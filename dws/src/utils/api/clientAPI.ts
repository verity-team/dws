"use client";

import useSWR, { Revalidator, RevalidatorOptions } from "swr";
import { HttpMethod, clientBaseRequest } from "./baseAPI";
import {
  AffiliateDonationInfo,
  CampaignStatus,
  DonationData,
  DonationStats,
  FailedResponse,
  TokenPrice,
  UserDonationData,
} from "./types";
import { useMemo } from "react";
import { getExponentialWaitTime } from "../utils";
import { Nullable } from "../types";
import useSWRImmutable from "swr/immutable";

interface CustomError extends Error {
  info: FailedResponse;
  status: number;
}

const fetcher = async (url: string) => {
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

const handleErrorRetry = (
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

export const useDonationData = () => {
  // Exponential backoff
  const { data, error, isLoading } = useSWRImmutable<DonationData, CustomError>(
    "/api/donation/data",
    fetcher,
    {
      // Refresh once per minute
      refreshInterval: 60000,
      onErrorRetry: handleErrorRetry,
    }
  );

  const tokenPrices: TokenPrice[] = useMemo(() => {
    if (data == null) {
      return [];
    }

    return data.prices;
  }, [data]);

  const donationStat: DonationStats = useMemo(() => {
    if (data == null) {
      return {
        total: "Not available",
        tokens: "Not available",
      };
    }

    return data.stats;
  }, [data]);

  const receivingWallet: string = useMemo(() => {
    if (data == null) {
      return "Unavailable";
    }
    return data.receiving_address;
  }, [data]);

  // Temporarily paused if cannot connect to server, or cannot get latest data
  const status: CampaignStatus = useMemo(() => {
    if (data == null) {
      return "paused";
    }

    return data.status;
  }, [data]);

  return {
    tokenPrices,
    donationStat,
    receivingWallet,
    status,
    error,
    isLoading,
  };
};

// For data revalidation when sending new donations
export const getUserDonationDataKey = (account: string) =>
  `/api/donation/user/${account}`;

export const useUserDonationData = (account: string) => {
  const { data, error, isLoading } = useSWR<UserDonationData, CustomError>(
    getUserDonationDataKey(account),
    fetcher,
    {
      revalidateOnFocus: false,
      onErrorRetry: handleErrorRetry,
    }
  );

  return { data, error, isLoading };
};
