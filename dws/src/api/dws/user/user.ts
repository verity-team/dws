import { Nullable } from "@/utils";
import { CustomError } from "@/utils/api/types";
import { swrFetcher, handleErrorRetry } from "@/utils/baseAPI";
import { useMemo, useState, useEffect } from "react";
import useSWRImmutable from "swr/immutable";
import { UserDonationData } from "./user.type";

// For data revalidation when sending new donations
export const getUserDonationDataKey = (account: string) =>
  `/api/donation/user/${account}`;

// For long-term use of data, have data refresh integrated
/**
 * Get user donation data from server with refresh
 *
 * Refresh interval are changed based on the response's body
 *
 * @param account [string]
 * @returns {{data, error, isLoading}} Return user's data, error (if any), and request status
 */
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
  >(getUserDonationDataKey(account), swrFetcher, {
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
    const response = await swrFetcher(getUserDonationDataKey(account));
    return response;
  } catch {
    return null;
  }
};
