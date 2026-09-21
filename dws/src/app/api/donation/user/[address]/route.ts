import { UserDonationData } from "@/api/dws/user/user.type";
import { FailedResponse } from "@/utils/baseAPI";
import { baseNextServerRequest } from "@/utils/baseApiV2";
import { NextResponse } from "next/server";
import { isAddress } from "web3-validator";

export const runtime = "edge";
export const revalidate = 60; // seconds

// The backend caps a page of the donation history at 100 records; reject
// anything outside the accepted range here as well instead of forwarding it.
const MAX_LIMIT = 100;
const DEFAULT_LIMIT = 50;

export async function GET(
  request: Request,
  { params }: { params: { address: string } }
): Promise<Response> {
  const walletAddr = params.address;
  if (!isAddress(walletAddr)) {
    return getBadRequestResponse("Invalid wallet address");
  }

  const { searchParams } = new URL(request.url);

  const limit = parsePositiveInt(searchParams.get("limit"), DEFAULT_LIMIT);
  if (limit == null || limit < 1 || limit > MAX_LIMIT) {
    return getBadRequestResponse(
      `"limit" must be an integer between 1 and ${MAX_LIMIT}`
    );
  }

  const offset = parsePositiveInt(searchParams.get("offset"), 0);
  if (offset == null || offset < 0) {
    return getBadRequestResponse('"offset" must be a non-negative integer');
  }

  return getUserDonation(walletAddr, limit, offset);
}

function parsePositiveInt(raw: string | null, fallback: number): number | null {
  if (raw == null || raw.trim() === "") {
    return fallback;
  }
  if (!/^\d+$/.test(raw.trim())) {
    return null;
  }
  const value = Number(raw);
  return Number.isSafeInteger(value) ? value : null;
}

async function getUserDonation(
  walletAddress: string,
  limit: number,
  offset: number
): Promise<NextResponse> {
  const path = `/user/data/${walletAddress}?limit=${limit}&offset=${offset}`;
  const response = await baseNextServerRequest("GET", { path });
  if (response == null) {
    return getDefaultErrResponse();
  }

  // Redirect response to frontend with their respective statusCode
  if (!response.ok) {
    return new NextResponse(undefined, { status: response.status });
  }

  // Should exist a response body whether the request fail or not
  let responseBody = null;
  try {
    responseBody = await response.json();
  } catch {
    return getDefaultErrResponse();
  }

  return NextResponse.json<UserDonationData>(responseBody ?? {}, {
    status: 200,
  });
}

function getBadRequestResponse(message: string): NextResponse {
  return NextResponse.json<FailedResponse>(
    { code: "400", message },
    { status: 400 }
  );
}

function getDefaultErrResponse(): NextResponse {
  return NextResponse.json<FailedResponse>(
    {
      code: "500",
      message: "Cannot fetch user donation data. Please try again later",
    },
    { status: 500 }
  );
}
