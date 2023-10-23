"use client";

import useSWR, { Revalidator } from "swr";
import { HttpMethod, clientBaseRequest } from "./baseAPI";
import {
  CampaignStatus,
  CustomError,
  DonationData,
  DonationStats,
  TokenPrice,
  UserDonationData,
} from "./types";
import { useMemo } from "react";
import { getExponentialWaitTime, sleep } from "../utils";
import { Maybe, Nullable } from "../types";
import useSWRImmutable from "swr/immutable";
import {
  GenAffiliateRequest,
  GenAffliateResponse,
  WalletAffiliateResponse,
  WalletAffliateRequest,
} from "./types/affliate.type";

const fetcher = async (url: string) => {
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

// TODO: Add data refresh interval
// For long-term use of data, have data refresh integrated
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

// For one-off request
export const getUserDonationData = async (
  account: string
): Promise<Nullable<UserDonationData>> => {
  try {
    const response = await fetcher(getUserDonationDataKey(account));
    return response;
  } catch {
    return null;
  }
};

export const connectWalletWithAffliate = async (
  walletAffliateRequest: WalletAffliateRequest
): Promise<Nullable<UserDonationData>> => {
  const response = await withErrorRetry(
    async () => {
      return await clientBaseRequest(
        "/api/donation/affliate",
        HttpMethod.POST,
        walletAffliateRequest
      );
    },
    (response: Maybe<Response>) => {
      if (response == null) {
        return false;
      }

      if (response.ok || response.status === 400 || response.status === 404) {
        return false;
      }

      return true;
    },
    5
  );

  if (response == null) {
    return null;
  }

  // TODO: Add extra logic to handle 404 logic if needed, or else just ignore
  if (!response.ok) {
    return null;
  }

  try {
    // There should be a body in response
    const result = await response.json();
    return result;
  } catch {
    return null;
  }
};

export const requestNewAffiliateCode = async (
  request: GenAffiliateRequest
): Promise<Maybe<GenAffliateResponse>> => {
  const response = await clientBaseRequest(
    "/api/affiliate/gen",
    HttpMethod.POST,
    request
  );

  if (response == null) {
    return null;
  }

  if (!response.ok) {
    return null;
  }

  try {
    // There should be a body in response
    const result = await response.json();
    return result;
  } catch {
    return null;
  }
};

const withErrorRetry = async (
  request: () => Promise<Maybe<Response>>,
  shouldRetry: (response: Maybe<Response>) => boolean,
  limit: number
): Promise<Maybe<Response>> => {
  let counter = 0;
  while (counter < limit) {
    const response = await request();

    if (!shouldRetry(response)) {
      return response;
    }

    await sleep(getExponentialWaitTime(1000, counter));
    counter++;
  }

  return null;
};
