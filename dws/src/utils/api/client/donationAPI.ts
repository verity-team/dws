"use client";

import { Nullable } from "@/utils/types";
import { useMemo, useState, useEffect } from "react";
import useSWRImmutable from "swr/immutable";
import {
  UserDonationData,
  CustomError,
  CampaignStatus,
  DonationData,
  DonationStats,
  TokenPrice,
} from "../types";
import { fetcher, handleErrorRetry } from "./clientAPI";

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
  const defaultWaitTime = useMemo(() => {
    const time = Number(process.env.NEXT_PUBLIC_UDATA_REFRESH_TIME_DEFAULT);

    if (isNaN(time)) {
      // 5 minutes
      return 5 * 60 * 1000;
    }

    return time;
  }, []);

  const unconfirmWaitTime = useMemo(() => {
    const time = Number(process.env.NEXT_PUBLIC_UDATA_REFRESH_TIME_UNCONFIRMED);

    if (isNaN(time)) {
      return 60 * 1000;
    }

    return time;
  }, []);

  const [waitTime, setWaitTime] = useState(defaultWaitTime);

  const { data, error, isLoading } = useSWRImmutable<
    UserDonationData,
    CustomError
  >(getUserDonationDataKey(account), fetcher, {
    refreshInterval: waitTime,
    onErrorRetry: handleErrorRetry,
  });

  useEffect(() => {
    if (data == null || data.donations == null) {
      return;
    }

    // If there are any unconfirmed donation, reduce the wait time
    if (data.donations.some((donation) => donation.status === "unconfirmed")) {
      setWaitTime(unconfirmWaitTime);
      return;
    }

    // If none are unconfirmed, revert back to default wait time
    if (waitTime !== defaultWaitTime) {
      setWaitTime(defaultWaitTime);
    }
  }, [data]);

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
