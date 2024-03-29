import { WalletAffiliateRequest } from "@/api/dws/affiliate/affiliate.type";
import { Nullable } from "@/utils";
import { NextResponse } from "next/server";
import { isAddress } from "web3-validator";
import { baseNextServerRequest } from "@/utils/baseApiV2";
import { FailedResponse } from "@/utils/baseAPI";

export const runtime = "edge";

export async function POST(request: Request): Promise<Response> {
  let requestBody: Nullable<WalletAffiliateRequest> = null;
  try {
    requestBody = await request.json();
  } catch {
    return getDefaultErrResponse();
  }

  if (requestBody == null) {
    return getDefaultErrResponse();
  }

  const { code, address } = requestBody;
  if (code == null) {
    return getBadRequestResponse("Affiliate code is required");
  }
  if (address == null) {
    return getBadRequestResponse("Wallet address is required");
  }
  if (!isAddress(address)) {
    return getBadRequestResponse("Invalid user wallet address");
  }

  const serverResponse = await requestWalletConnection(requestBody);
  return serverResponse;
}

async function requestWalletConnection(
  donationInfo: WalletAffiliateRequest
): Promise<NextResponse> {
  const path = "/wallet/connection";
  const response = await baseNextServerRequest("POST", {
    path,
    payload: donationInfo,
    json: true,
  });

  if (response == null) {
    return getDefaultErrResponse();
  }

  if (!response.ok) {
    let errorMessage = null;
    try {
      errorMessage = await response.json();
    } catch {
      // ignore err, return failed response anyway
    }

    return NextResponse.json<FailedResponse>(errorMessage ?? {}, {
      status: response.status,
    });
  }

  // Return empty JSON object to avoid parsing errors
  return NextResponse.json({}, { status: 200 });
}

function getBadRequestResponse(message?: string): NextResponse {
  return NextResponse.json<FailedResponse>(
    { code: "400", message: message ?? "Invalid request body" },
    { status: 400 }
  );
}

function getDefaultErrResponse(): NextResponse {
  return NextResponse.json<FailedResponse>(
    { code: "500", message: "Failed to send confirmation. Please try again." },
    { status: 500 }
  );
}
