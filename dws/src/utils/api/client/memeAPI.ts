import mime from "mime/lite";

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

  return true;
};
