import { Nullable } from "@/utils";
import { swrFetcher, handleErrorRetry, CustomError } from "@/utils/baseAPI";
import { useMemo, useState } from "react";
import useSWRImmutable from "swr/immutable";
import { UserDonationData } from "./user.type";

// The donation history served by /user/data/{address} is paginated and the
// backend caps a page at 100 records; ask for one screenful at a time.
export const DONATION_PAGE_SIZE = 20;

// For data revalidation when sending new donations
export const getUserDonationDataKey = (account: string, page: number = 0) => {
  if (!account) {
    return null;
  }

  const offset = Math.max(0, page) * DONATION_PAGE_SIZE;
  return `/api/donation/user/${account}?limit=${DONATION_PAGE_SIZE}&offset=${offset}`;
};

/**
 * SWR key filter matching every cached page of the given address.
 *
 * A new donation lands at the end of the history, so revalidating the first
 * page alone would leave the page the user is looking at stale.
 *
 * @param account [string]
 * @returns a predicate usable as the key argument of SWR's global `mutate`
 */
export const getUserDonationDataKeyFilter = (account: string) => {
  return (key: unknown): boolean =>
    typeof key === "string" && key.startsWith(`/api/donation/user/${account}?`);
};

// For long-term use of data, have data refresh integrated
/**
 * Get user donation data from server with refresh
 *
 * Refresh interval are changed based on the response's body
 *
 * @param account [string]
 * @param page [number] which page of the donation history to fetch
 * @returns {{data, error, isLoading}} Return user's data, error (if any), and request status
 */
export const useUserDonationData = (account: string, page: number = 0) => {
  const unconfirmWaitTime = useMemo(() => {
    const time = Number(process.env.NEXT_PUBLIC_UDATA_REFRESH_TIME_UNCONFIRMED);

    if (isNaN(time)) {
      return 60 * 1000;
    }

    return time;
  }, []);

  const [waitTime, setWaitTime] = useState(unconfirmWaitTime);

  const { data, error, isLoading } = useSWRImmutable<
    UserDonationData,
    CustomError
  >(getUserDonationDataKey(account, page), swrFetcher, {
    refreshInterval: waitTime,
    onErrorRetry: handleErrorRetry,
    // paging must not blank the history out while the next page is in flight
    keepPreviousData: true,
  });

  return { data, error, isLoading };
};

// For one-off request
export const getUserDonationData = async (
  account: string,
  page: number = 0
): Promise<Nullable<UserDonationData>> => {
  try {
    const fetchKey = getUserDonationDataKey(account, page);
    if (fetchKey == null) {
      return null;
    }

    const response = await swrFetcher(fetchKey);
    return response;
  } catch {
    return null;
  }
};
