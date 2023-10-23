import { HttpMethod, serverBaseRequest } from "@/utils/api/baseAPI";
import { FailedResponse } from "@/utils/api/types";
import { WalletAffliateRequest } from "@/utils/api/types/affliate.type";
import { Nullable } from "@/utils/types";
import { NextResponse } from "next/server";
import { isAddress } from "web3-validator";

export const runtime = "edge";

export async function POST(request: Request): Promise<Response> {
  let requestBody: Nullable<WalletAffliateRequest> = null;
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
  donationInfo: WalletAffliateRequest
): Promise<NextResponse> {
  const response = await serverBaseRequest(
    "/wallet/connection",
    HttpMethod.POST,
    donationInfo
  );

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