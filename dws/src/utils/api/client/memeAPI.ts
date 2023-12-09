import mime from "mime/lite";
import { clientFormRequest } from "../baseAPI";

export const uploadMeme = async (
  caption: string,
  image: File
): Promise<boolean> => {
  const payload = new FormData();
  payload.append("caption", caption);
  payload.append("meme", {
    name: "meme",
    uri: image,
    type: mime.getType(image.name),
  } as any);

  const response = await clientFormRequest("/meme", payload);
  if (response == null || !response.ok) {
    return false;
  }

  return true;
};