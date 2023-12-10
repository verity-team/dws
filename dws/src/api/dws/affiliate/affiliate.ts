"use client";

import { sleep, getExponentialWaitTime } from "@/utils/utils";
import { clientBaseRequest, HttpMethod } from "../../../utils/baseAPI";
import { UserDonationData } from "../../../utils/api/types";
import {
  WalletAffiliateRequest,
  GenAffiliateRequest,
  GenAffiliateResponse,
} from "../../../utils/api/types/affiliate.type";
import { Nullable, Maybe } from "@/utils";

export const connectWalletWithAffiliate = async (
  walletAffiliateRequest: WalletAffiliateRequest
): Promise<Nullable<UserDonationData>> => {
  const response = await withErrorRetry(
    async () => {
      return await clientBaseRequest(
        "/api/affiliate/connect",
        HttpMethod.POST,
        walletAffiliateRequest
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
): Promise<Maybe<GenAffiliateResponse>> => {
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
