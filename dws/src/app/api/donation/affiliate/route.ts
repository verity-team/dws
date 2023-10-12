import { HttpMethod, serverBaseRequest } from "@/utils/api/baseAPI";
import { AffiliateDonationInfo, FailedResponse } from "@/utils/api/types";
import { Nullable } from "@/utils/types";
import { NextResponse } from "next/server";

export const runtime = "edge";

export async function POST(request: Request): Promise<Response> {
  let requestBody: Nullable<AffiliateDonationInfo> = null;
  try {
    requestBody = await request.json();
  } catch {
    return getDefaultErrResponse();
  }

  if (requestBody == null) {
    return getDefaultErrResponse();
  }

  const { code, tx_hash } = requestBody;
  if (code == null) {
    return getBadRequestResponse("Affiliate code is required");
  }
  if (tx_hash == null) {
    return getBadRequestResponse("Transaction hash is required");
  }

  const serverResponse = await requestAffiliateDonation(requestBody);
  return serverResponse;
}

async function requestAffiliateDonation(
  donationInfo: AffiliateDonationInfo
): Promise<NextResponse> {
  const response = await serverBaseRequest(
    "/donation/affiliate",
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
