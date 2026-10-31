"use client";

import { sleep, getExponentialWaitTime } from "@/utils/utils";
import { Maybe } from "@/utils";
import { clientBaseRequest, HttpMethod } from "@/utils/baseAPI";
import {
  WalletAffiliateRequest,
  WalletConnectionResult,
  GenAffiliateRequest,
  GenAffiliateResponse,
} from "./affiliate.type";

export const connectWalletWithAffiliate = async (
  walletAffiliateRequest: WalletAffiliateRequest
): Promise<WalletConnectionResult> => {
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

      if (
        response.ok ||
        response.status === 400 ||
        response.status === 401 ||
        response.status === 404 ||
        response.status === 422
      ) {
        // a stale/invalid signature will not become valid on a retry, nor
        // will a turned down affiliate code
        return false;
      }

      return true;
    },
    5
  );

  if (response == null) {
    return "failed";
  }

  if (response.ok) {
    return "connected";
  }

  if (response.status !== 422) {
    return "failed";
  }

  try {
    const { code } = await response.json();
    if (code === 119) {
      return "unknown_code";
    }
    if (code === 120) {
      return "self_referral";
    }
  } catch {
    // no readable error body
  }
  return "failed";
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
