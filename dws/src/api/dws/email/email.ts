import { baseEmailServerRequest } from "@/utils/baseApiV2";

export const subscribeEmail = async (email: string): Promise<boolean> => {
  const payload = { email };
  const path = "/subscribe.php";

  try {
    const response = await baseEmailServerRequest("POST", {
      path,
      payload,
      json: true,
    });

    if (response == null || !response.ok) {
      console.error("Failed to subscribe your email. Please try again later");
      return false;
    }
  } catch (error) {
    console.error(error);
    return false;
  }

  return true;
};
