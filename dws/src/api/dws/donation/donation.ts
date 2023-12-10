"use client";

import { useMemo, useState, useEffect } from "react";
import useSWRImmutable from "swr/immutable";
import { Nullable } from "@/utils";
import { swrFetcher, handleErrorRetry } from "@/utils/baseAPI";
import { CustomError, UserDonationData } from "@/utils/api/types";
import {
  DonationData,
  TokenPrice,
  DonationStats,
  CampaignStatus,
} from "./donation.type";

export const useDonationData = () => {
  // Exponential backoff
  const { data, error, isLoading } = useSWRImmutable<DonationData, CustomError>(
    "/api/donation/data",
    swrFetcher,
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
