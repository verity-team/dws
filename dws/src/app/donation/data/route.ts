import { HttpMethod, sendBaseRequest } from "@/utils/api/baseAPI";
import { DonationData, FailedResponse } from "@/utils/api/types";
import { NextResponse } from "next/server";

export async function GET(request: Request): Promise<Response> {
  const serverResponse = await getDonationData();
  return serverResponse;
}

async function getDonationData(): Promise<Response> {
  const response = await sendBaseRequest("donation/data/", HttpMethod.GET);

  // Something is wrong with API setup
  if (response == null) {
    return getDefaultErrResponse();
  }

  // Avoid parsing errors
  let responseBody = null;
  try {
    responseBody = await response.json();
  } catch {
    return getDefaultErrResponse();
  }

  console.log(responseBody);

  // Redirect response to frontend with their respective statusCode
  if (!response.ok) {
    return Response.json(responseBody ?? {}, {
      status: response.status,
    });
  }

  return Response.json(responseBody ?? {}, { status: 200 });
}

function getDefaultErrResponse(): Response {
  return Response.json(
    { code: "unknown", message: "Failed to fetch donation data" },
    { status: 500 }
  );
}
