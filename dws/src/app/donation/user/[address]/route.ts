import { HttpMethod, serverBaseRequest } from "@/utils/api/baseAPI";
import { FailedResponse, UserDonationData } from "@/utils/api/types";
import { NextResponse } from "next/server";
import { isAddress } from "web3-validator";

export async function GET(
  request: Request,
  { params }: { params: { address: string } }
): Promise<Response> {
  const walletAddr = params.address;
  console.log(walletAddr);

  if (!isAddress(walletAddr)) {
    return getBadRequestResponse("Invalid wallet address");
  }

  return getUserDonation(walletAddr);
}

// TODO: Need more testing
async function getUserDonation(walletAddr: string): Promise<NextResponse> {
  const response = await serverBaseRequest(
    `/user/data/${walletAddr}`,
    HttpMethod.GET
  );
  if (response == null) {
    return getDefaultErrResponse();
  }

  // Should exist a response body whether the request fail or not
  let responseBody = null;
  try {
    responseBody = await response.json();
  } catch {
    return getDefaultErrResponse();
  }

  // Redirect response to frontend with their respective statusCode
  if (!response.ok) {
    return NextResponse.json<FailedResponse>(responseBody ?? {}, {
      status: response.status,
    });
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
