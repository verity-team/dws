import { HttpMethod, baseRequest, getDefaultHeaders } from "../baseAPI";

export const subscribeEmail = async (email: string): Promise<void> => {
  const emailAPIHost = process.env.NEXT_PUBLIC_EMAIL_API_URL;
  if (emailAPIHost == null || emailAPIHost.trim() === "") {
    console.warn("Email subscription endpoint not set");
    return;
  }

  const payload = { email };
  const response = await baseRequest(
    "/subscribe.php",
    HttpMethod.POST,
    { host: emailAPIHost, timeout: 10000 },
    payload,
    getDefaultHeaders()
  );

  if (response == null) {
    throw new Error("No response");
  }

  if (!response.ok) {
    throw new Error("Failed to subscribe your email. Please try again later");
  }
};
