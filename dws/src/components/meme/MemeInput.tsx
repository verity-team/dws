"use client";

import { useMemo, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { Maybe } from "@/utils/types";

import Image from "next/image";
import CloseIcon from "@mui/icons-material/Close";
import MemeDropArea from "./MemeDropArea";
import MemeToolbar from "./MemeToolbar";

interface MemeFormData {
  caption: string;
}

const MemeInput = () => {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<MemeFormData>({
    values: {
      caption: "",
    },
  });

  const memeInputRef = useRef<HTMLInputElement>(null);

  const [meme, setMeme] = useState<Maybe<File>>(null);

  const handleMemeChange = (meme: Maybe<File>) => {
    setMeme(meme);
  };

  const handleMemeRemove = () => {
    setMeme(null);
  };

  const handleImageBtnClick = () => {
    if (memeInputRef == null || memeInputRef.current == null) {
      return;
    }

    memeInputRef.current.click();
  };

  const handlePostBtnClick = () => {
    handleSubmit(handleMemeUpload);
  };

  const handleMemeUpload = (data: MemeFormData) => {};

  const canPost = useMemo(() => {
    // No images uploaded
    if (meme == null) {
      return false;
    }

    return true;
  }, [meme]);

  return (
    <div>
      <div className="w-full p-2 border-2 border-gray-100">
        <form onSubmit={handleSubmit(handleMemeUpload)}></form>
        <MemeDropArea
          onMemeChange={handleMemeChange}
          fileInputRef={memeInputRef}
        >
          <div className="flex items-center p-2">
            <Image src="/images/logo.png" width={48} height={48} alt="avatar" />
            <input
              className="px-4 py-2 w-full outline-none my-2 text-lg"
              placeholder="Unveil the truth ?!"
              {...register("caption", {
                required: true,
              })}
            />
          </div>
        </MemeDropArea>
        {meme && (
          <div className="container mx-auto mt-4 px-20 relative">
            <button
              onClick={handleMemeRemove}
              className="absolute top-2 right-2 rounded-full bg-gray-800 p-1"
            >
              <CloseIcon fontSize="medium" htmlColor="white" />
            </button>
            <Image
              src={URL.createObjectURL(meme)}
              alt="meme upload image"
              width={4000}
              height={0}
              className="object-cover w-full h-full"
            />
          </div>
        )}
        <div className="mt-4 p-2">
          <MemeToolbar
            canSubmit={canPost}
            onImageBtnClick={handleImageBtnClick}
            onPostBtnClick={handlePostBtnClick}
          />
        </div>
      </div>
    </div>
  );
};

export default MemeInput;
