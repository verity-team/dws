import { serverFormRequest, FailedResponse } from "@/utils/baseAPI";
import { NextResponse } from "next/server";

export async function POST(request: Request): Promise<Response> {
  const payload = await request.formData();
  if (payload == null) {
    return getDefaultErrResponse();
  }

  const response = await serverFormRequest("/meme", payload);
  if (response == null || !response.ok) {
    return getDefaultErrResponse();
  }

  try {
    const result = await response.json();
    return NextResponse.json(result, { status: 200 });
  } catch {
    return getDefaultErrResponse();
  }
}

function getDefaultErrResponse(): NextResponse {
  return NextResponse.json<FailedResponse>(
    { code: "500", message: "Failed to upload meme. Please try again." },
    { status: 500 }
  );
}
