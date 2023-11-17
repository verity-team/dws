"use client";

import TextButton from "@/components/common/TextButton";
import { Dialog, DialogContent, DialogTitle } from "@mui/material";
import { ChangeEvent, FormEvent, useCallback, useState } from "react";
import AddIcon from "@mui/icons-material/Add";
import CloseIcon from "@mui/icons-material/Close";
import { useForm } from "react-hook-form";
import { Maybe } from "@/utils/types";
import Image from "next/image";

interface UploadMemeFormData {
  caption: string;
  meme: File;
}

const UploadMemeButton = () => {
  const [uploadFormOpen, setUploadFormOpen] = useState(false);

  const [currentMeme, setCurrentMeme] = useState<string>("");

  const { register, handleSubmit } = useForm<UploadMemeFormData>();

  const handleOpenUploadForm = useCallback(() => {
    setUploadFormOpen(true);
  }, []);

  const handleCloseUploadForm = useCallback(() => {
    setUploadFormOpen(false);
  }, []);

  const handleMemeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const newMeme = event.target.files?.item(0);
    if (newMeme == null) {
      return;
    }

    const newMemeUrl = URL.createObjectURL(newMeme);
    setCurrentMeme(newMemeUrl);
  };

  const handleMemeRemove = () => {
    setCurrentMeme("");
  };

  const handleUploadMeme = (data: UploadMemeFormData) => {};

  return (
    <div className="flex justify-center items-center">
      <div className="text-2xl">
        <TextButton onClick={handleOpenUploadForm}>
          Upload your meme!
        </TextButton>
      </div>
      <Dialog
        open={uploadFormOpen}
        onClose={handleCloseUploadForm}
        maxWidth="sm"
        fullWidth
      >
        <DialogContent>
          <form onSubmit={handleSubmit(handleUploadMeme)}>
            <div className="flex justify-center items-center text-2xl mb-4">
              Upload your meme
            </div>
            <div className="mt-2 mb-4 relative">
              {!!currentMeme ? (
                <>
                  <button
                    onClick={handleMemeRemove}
                    className="absolute top-2 right-2 rounded-full bg-gray-800 p-1"
                  >
                    <CloseIcon fontSize="medium" htmlColor="white" />
                  </button>
                  <Image
                    src={currentMeme}
                    alt="meme"
                    width={100}
                    height={100}
                    className="w-full"
                  />
                </>
              ) : (
                <>
                  <label htmlFor="meme">
                    <div className="p-16 flex flex-col items-center justify-center border-2 border-black rounded-lg cursor-pointer">
                      <AddIcon fontSize="medium" />
                      <div>Add your meme here</div>
                    </div>
                  </label>
                  <input
                    type="file"
                    accept="image/jpeg, image/png, image/gif"
                    id="meme"
                    className="hidden"
                    {...register("meme", {
                      required: true,
                    })}
                    onChange={handleMemeChange}
                  />
                </>
              )}
            </div>
            <div className="my-2">
              <label htmlFor="caption" className="text-xl">
                Caption (optional)
              </label>
              <input
                id="caption"
                name="caption"
                type="text"
                className="my-2 py-2 w-full outline-none text-xl border-b focus:border-black"
                placeholder="What's the meme about?"
              />
            </div>
            <button
              type="submit"
              className="w-full px-4 py-2 border-2 border-black text-cred bg-white rounded-lg"
            >
              Upload
            </button>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default UploadMemeButton;
