import { getSingleMeme } from "@/api/galactica/meme/meme";
import { roboto } from "@/app/fonts";
import ItemToolbar from "@/components/galactica/meme/list/toolbar/ItemToolbar";
import { getTimeElapsedString } from "@/utils/utils";
import { getWalletShorthand } from "@/utils/wallet/wallet";
import Avatar from "boring-avatars";
import Image from "next/image";
import { ReactElement } from "react";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import { Metadata } from "next";

const prepareMetadata: Metadata = {
  twitter: {
    card: "summary_large_image",
    description: "Join the Truthmemes army and deliver the truth to everyone!",
    images: [],
  },
};

interface SingleMemePageProps {
  params: {
    id: string;
  };
}

const SingleMemePage = async ({
  params,
}: SingleMemePageProps): Promise<ReactElement<SingleMemePageProps>> => {
  const memeData = await getSingleMeme(params.id);
  if (memeData == null) {
    throw new Error("Cannot find meme");
  }

  const { fileId, userId, createdAt, caption } = memeData;

  const memeImage = `${process.env.GALACTICA_API_URL}/meme/image/${fileId}`;

  if (prepareMetadata.twitter) {
    prepareMetadata.twitter.images = [memeImage];
    prepareMetadata.twitter.title = caption;
    prepareMetadata.twitter.site = process.env.HOST_URL;
  }

  return (
    <div className="grid grid-cols-12">
      <div className="col-span-3"></div>
      <div className="col-span-6">
        <div className={roboto.className}>
          <a
            className="flex items-center my-8 hover:text-blue-500 hover:underline cursor-pointer space-x-4"
            href="/meme"
          >
            <ArrowBackIcon fontSize="medium" />
            <p className="text-lg">Back to timeline</p>
          </a>
          <div className="w-full mt-6 border-b border-gray-100">
            <div className="flex items-center justify-start space-x-4">
              <Avatar size={40} name={userId} variant="marble" />
              <div>
                <div className="font-semibold text-sm">
                  {getWalletShorthand(userId)}
                  <span className="text-sm font-light text-gray-500">
                    {" "}
                    • {getTimeElapsedString(createdAt)}
                  </span>
                </div>
                {caption && <div className="text-base mt-0.5">{caption}</div>}
              </div>
            </div>
            <div className="flex items-center justify-center mt-4">
              <Image
                src={memeImage}
                width={1000}
                height={1000}
                className="object-contain w-auto max-w-[80%] max-h-[40vh]"
                alt={caption}
                priority={true}
              />
            </div>
            <div className="mt-2">
              <ItemToolbar />
            </div>
          </div>
        </div>
      </div>
      <div className="col-span-3"></div>
    </div>
  );
};

export const metadata = prepareMetadata;
export default SingleMemePage;
