"use client";

import useSWR from "swr";
import { HttpMethod, clientBaseRequest } from "./baseAPI";
import {
  CampaignStatus,
  DonationData,
  DonationStats,
  FailedResponse,
  TokenPrice,
} from "./types";
import { useMemo } from "react";

interface CustomError {
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

export const useDonationData = () => {
  // Exponential backoff
  const { data, error, isLoading } = useSWR<any, CustomError>(
    "/api/donation/data",
    fetcher,
    {
      // Refresh once per minute
      refreshInterval: 60000,
      revalidateOnFocus: false,
      onErrorRetry: (err, key, config, revalidate, { retryCount }) => {
        if (error?.status === 404) {
          return;
        }

        if (retryCount >= 5) {
          return;
        }

        setTimeout(() => revalidate({ retryCount }), 3000);
      },
    }
  );

  const tokenPrices: TokenPrice[] = useMemo(() => {
    if (data == null) {
      return [];
    }

    return (data as DonationData).prices;
  }, [data]);

  const donationStat: DonationStats = useMemo(() => {
    if (data == null) {
      return {
        total: "Not available",
        tokens: "Not available",
      };
    }

    return (data as DonationData).stats;
  }, [data]);

  const receivingWallet: string = useMemo(() => {
    if (data == null) {
      return "Unavailable";
    }

    return (data as DonationData).receiving_address;
  }, [data]);

  // Temporarily paused if cannot connect to server, or cannot get latest data
  const status: CampaignStatus = useMemo(() => {
    if (data == null) {
      return "paused";
    }

    return (data as DonationData).status;
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
