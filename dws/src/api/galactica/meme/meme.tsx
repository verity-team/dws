import { clientFormRequest } from "@/utils/baseAPI";
import { MemeUploadDTO } from "./meme.type";

export const uploadMeme = async ({
  meme,
  caption,
  userId,
}: MemeUploadDTO): Promise<boolean> => {
  const formData = new FormData();

  // Construct formData
  formData.append("language", "en");
  formData.append("tags", "#meme");

  formData.append("userId", userId);
  formData.append("caption", caption);

  formData.append("fileName", meme);

  const response = await clientFormRequest(
    "/meme",
    formData,
    process.env.NEXT_PUBLIC_GALACTICA_API_URL
  );
  if (response == null || !response.ok) {
    return false;
  }

  return true;
};
