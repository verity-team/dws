"use client";

import {
  ReactElement,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
} from "react";
import Image from "next/image";
import CloseIcon from "@mui/icons-material/Close";
import MemeDropArea from "./MemeDropArea";
import MemeInputToolbar from "./MemeInputToolbar";
import toast from "react-hot-toast";
import { Maybe } from "@/utils";
import { uploadMeme } from "@/api/galactica/meme/meme";
import { ClientWallet } from "@/components/ClientRoot";
import { useForm } from "react-hook-form";
import { OptimisticMemeUpload } from "@/api/galactica/meme/meme.type";

interface MemeInputProps {
  onUpload: (meme: OptimisticMemeUpload) => void;
}

interface MemeInputFormData {
  caption: string;
}

const MemeInput = ({
  onUpload,
}: MemeInputProps): ReactElement<MemeInputProps> => {
  const walletAddress = useContext(ClientWallet);
  const memeInputRef = useRef<HTMLInputElement>(null);
  const [meme, setMeme] = useState<Maybe<File>>(null);
  const {
    register,
    getValues,
    reset: resetForm,
  } = useForm<MemeInputFormData>({
    defaultValues: { caption: "" },
  });

  const handleMemeChange = useCallback((meme: Maybe<File>) => {
    setMeme(meme);
  }, []);

  const handleMemeRemove = useCallback(() => {
    setMeme(null);
  }, []);

  const handleImageBtnClick = useCallback(() => {
    if (memeInputRef == null || memeInputRef.current == null) {
      return;
    }

    memeInputRef.current.click();
  }, []);

  const handleMemeUpload = async () => {
    if (meme == null) {
      toast.error("You have not upload any image");
      return;
    }

    if (walletAddress == null || walletAddress === "") {
      // TODO: Open pop-up to prompt user to sign in
      toast.error("You need to sign-in first");
      return;
    }

    // Use API to upload meme to server
    const caption = getValues("caption");
    const uploaded = await uploadMeme({
      meme,
      caption,
      userId: walletAddress,
    });
    if (!uploaded) {
      toast.error("Failed to upload. Please try again later");
      return;
    }

    onUpload({
      userId: walletAddress,
      fileId: URL.createObjectURL(meme),
      caption,
      createdAt: new Date().toISOString(),
    });

    // Clear form
    resetForm();
    setMeme(null);

    // Toast
    toast.success("Post uploaded");
  };

  const canPost = useMemo(() => {
    // No images uploaded
    if (meme == null) {
      return false;
    }

    return true;
  }, [meme]);

  return (
    <div>
      <div className="w-full p-2">
        <div className="p-2 border-2 border-black rounded-lg bg-white">
          <MemeDropArea
            onMemeChange={handleMemeChange}
            fileInputRef={memeInputRef}
          >
            <div className="flex items-center space-x-2 p-2">
              <Image
                src="/images/logo.png"
                width={48}
                height={48}
                alt="avatar"
              />
              <input
                className="px-4 py-2 w-full outline-none my-2 text-lg rounded-lg border-2 border-black"
                placeholder="Unveil the truth ?!"
                {...register("caption")}
              />
            </div>
          </MemeDropArea>
          {meme && (
            <div className="container mx-auto mt-4 relative">
              <button
                onClick={handleMemeRemove}
                className="absolute top-2 right-2 rounded-full bg-gray-800 p-1"
              >
                <CloseIcon fontSize="medium" htmlColor="white" />
              </button>
              <Image
                src={URL.createObjectURL(meme)}
                alt="meme upload image"
                width={0}
                height={0}
                className="object-contain w-auto max-w-full max-h-full mx-auto"
              />
            </div>
          )}
          <div className="mt-4 p-2">
            <MemeInputToolbar
              canSubmit={canPost}
              onImageBtnClick={handleImageBtnClick}
              onPostBtnClick={handleMemeUpload}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

export default MemeInput;
