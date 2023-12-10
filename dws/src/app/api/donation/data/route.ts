import { serverBaseRequest, HttpMethod } from "@/utils/baseAPI";
import { NextRequest, NextResponse } from "next/server";

export const revalidate = 60; // seconds
export const runtime = "edge";

export async function GET(request: NextRequest) {
  const response = await serverBaseRequest("/donation/data", HttpMethod.GET);

  // Something is wrong with API setup
  if (response == null) {
    return getDefaultErrResponse();
  }

  // Should exist a response body whether the request fail or not
  let responseBody = {};
  try {
    responseBody = await response.json();
    return NextResponse.json(responseBody, { status: response.status });
  } catch {
    return getDefaultErrResponse();
  }
}

function getDefaultErrResponse(): NextResponse {
  return NextResponse.json(
    { code: "500", message: "Failed to fetch donation data" },
    { status: 500 }
  );
}
