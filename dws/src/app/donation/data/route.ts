import { HttpMethod, sendBaseRequest } from "@/utils/api/baseAPI";
import { DonationData, FailedResponse } from "@/utils/api/types";
import { NextResponse } from "next/server";

export async function GET(request: Request): Promise<Response> {
  const serverResponse = await getDonationData();
  return serverResponse;
}

async function getDonationData(): Promise<NextResponse> {
  const response = await sendBaseRequest("donation/data/", HttpMethod.GET);

  // Something is wrong with API setup
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

  return NextResponse.json<DonationData>(responseBody ?? {}, { status: 200 });
}

function getDefaultErrResponse(): NextResponse {
  return NextResponse.json<FailedResponse>(
    { code: "500", message: "Failed to fetch donation data" },
    { status: 500 }
  );
}
