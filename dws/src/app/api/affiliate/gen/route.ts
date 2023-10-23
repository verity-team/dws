import {
  HttpMethod,
  getDefaultHeaders,
  serverBaseRequest,
} from "@/utils/api/baseAPI";
import { FailedResponse } from "@/utils/api/types";
import {
  GenAffiliateRequest,
  GenAffliateResponse,
} from "@/utils/api/types/affliate.type";
import { Nullable } from "@/utils/types";
import { timeStamp } from "console";
import { NextResponse } from "next/server";
import { isAddress } from "web3-validator";

export const runtime = "edge";

export async function POST(request: Request): Promise<Response> {
  let requestBody: Nullable<GenAffiliateRequest> = null;
  try {
    requestBody = await request.json();
  } catch {
    return getDefaultErrResponse();
  }

  if (requestBody == null) {
    return getDefaultErrResponse();
  }

  const { address, timestamp, signature } = requestBody;
  if (address == null) {
    return getBadRequestResponse("Wallet address is required");
  }
  if (!isAddress(address)) {
    return getBadRequestResponse("Invalid user wallet address");
  }

  if (timestamp == null) {
    return getBadRequestResponse("Timestamp is required");
  }

  if (signature == null) {
    return getBadRequestResponse("User signature is required");
  }

  return await requestNewAffiliateCode(requestBody);
}

async function requestNewAffiliateCode(
  request: GenAffiliateRequest
): Promise<NextResponse> {
  const headers = getDefaultHeaders();
  headers.append("delphi-key", request.address);
  headers.append("delphi-ts", request.timestamp);
  headers.append("delphi-signature", request.signature);

  const response = await serverBaseRequest(
    "/affiliate/code",
    HttpMethod.POST,
    {},
    headers
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

  try {
    const result = await response.json();
    return NextResponse.json(result, { status: 200 });
  } catch {
    return getDefaultErrResponse();
  }
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
