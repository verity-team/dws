"use client";

import TextButton from "@/components/common/TextButton";
import { Dialog, DialogContent } from "@mui/material";
import { ChangeEvent, useCallback, useState } from "react";
import AddIcon from "@mui/icons-material/Add";
import CloseIcon from "@mui/icons-material/Close";
import { useForm } from "react-hook-form";
import Image from "next/image";
import { uploadMeme } from "@/api/memeAPI";
import toast from "react-hot-toast";
import { Maybe } from "@/utils";

interface UploadMemeFormData {
  caption: string;
}

const UploadMemeButton = () => {
  const [uploadFormOpen, setUploadFormOpen] = useState(false);

  const [currentMeme, setCurrentMeme] = useState<Maybe<File>>(null);

  const { register, handleSubmit, reset } = useForm<UploadMemeFormData>();

  const handleOpenUploadForm = useCallback(() => {
    setUploadFormOpen(true);
  }, []);

  const handleCloseUploadForm = useCallback(() => {
    setUploadFormOpen(false);

    // Clear form data
    setCurrentMeme(null);
    reset();
  }, []);

  const handleMemeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const newMeme = event.target.files?.item(0);
    if (newMeme == null) {
      return;
    }
    setCurrentMeme(newMeme);
  };

  const handleMemeRemove = () => {
    setCurrentMeme(null);
  };

  const handleMemeUpload = async (data: UploadMemeFormData) => {
    if (currentMeme == null) {
      return;
    }

    const isUploaded = await uploadMeme(data.caption, currentMeme);
    if (!isUploaded) {
      // Toast failed but don't close the form yet
      toast.error("Upload failed");
      return;
    }

    toast.success("Your meme have been uploaded");
    handleCloseUploadForm();
  };

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
          <form onSubmit={handleSubmit(handleMemeUpload)}>
            <div className="flex items-center justify-between text-2xl mb-8">
              <div>Upload meme</div>
              <button onClick={handleCloseUploadForm}>
                <CloseIcon fontSize="medium" />
              </button>
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
                  <div className="flex items-center justify-center">
                    <Image
                      src={URL.createObjectURL(currentMeme)}
                      alt="meme"
                      width={400}
                      height={400}
                      className="max-h-[400px]"
                    />
                  </div>
                </>
              ) : (
                <>
                  <label htmlFor="meme">
                    <div className="p-16 flex flex-col items-center justify-center border-2 border-black rounded-lg cursor-pointer">
                      <AddIcon fontSize="large" />
                      <div className="mt-2">*Add your meme here</div>
                    </div>
                  </label>
                  <input
                    type="file"
                    accept="image/jpeg, image/png, image/gif"
                    id="meme"
                    className="hidden"
                    autoComplete="off"
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
                type="text"
                className="my-2 py-2 w-full outline-none text-xl border-b focus:border-black"
                placeholder="What's the meme about?"
                autoComplete="off"
                {...register("caption", { required: false, maxLength: 128 })}
              />
            </div>
            <button
              type="submit"
              className="w-full px-4 py-2 border-2 border-black text-cred bg-white rounded-lg disabled:opacity-70 disabled:cursor-not-allowed"
              disabled={!currentMeme}
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
