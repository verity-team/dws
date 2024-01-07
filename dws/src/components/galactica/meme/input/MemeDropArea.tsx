"use client";

import {
  ChangeEvent,
  DragEvent,
  ReactElement,
  ReactNode,
  Ref,
  useMemo,
  useState,
} from "react";
import MemeInputToolbar from "./MemeInputToolbar";
import toast from "react-hot-toast";

type FileStatus = "TOO_BIG" | "INVALID_MIME" | "OK";

interface MemeDropAreaProps {
  onMemeChange: (file: File) => void;
  children: ReactNode;
  fileInputRef?: Ref<HTMLInputElement>;
}

const allowedFileTypes = ["image/png", "image/jpeg", "image/gif"];

const validateUploadFile = (file: File): FileStatus => {
  // Default max file size to 1MB ~ 1.000.000 bytes
  let maxFileSize = Number(process.env.NEXT_PUBLIC_FILE_MAX);
  if (isNaN(maxFileSize)) {
    maxFileSize = 1_000_000;
  }

  if (file.size > maxFileSize) {
    return "TOO_BIG";
  }

  const isMimeValid = allowedFileTypes.some((type) => type === file.type);
  if (!isMimeValid) {
    return "INVALID_MIME";
  }

  return "OK";
};

const MemeDropArea = ({
  onMemeChange,
  children,
  fileInputRef,
}: MemeDropAreaProps): ReactElement<MemeDropAreaProps> => {
  const [dragActive, setDragActive] = useState(false);

  const handleImageDrag = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();

    if (event.type === "dragenter" || event.type === "dragover") {
      setDragActive(true);
    } else {
      setDragActive(false);
    }
  };

  const handleImageDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();

    setDragActive(false);

    if (event.dataTransfer.files && event.dataTransfer.files[0]) {
      // use callback to add file to state and to preview file
      onMemeChange(event.dataTransfer.files[0]);
    }
  };

  const handleImageChange = (event: ChangeEvent<HTMLInputElement>) => {
    if (!event.target.files?.[0]) {
      return;
    }

    const uploadedFile = event.target.files[0];
    const validationResult = validateUploadFile(uploadedFile);
    if (validationResult === "TOO_BIG") {
      toast.error("Your meme is too big. Best we can do is 1MB");
      return;
    }

    if (validationResult === "INVALID_MIME") {
      toast.error("Meme should be an image. We support PNG, JPG, and GIF");
      return;
    }

    onMemeChange(event.target.files[0]);
  };

  const overlayStyle = useMemo(() => {
    const baseStyle =
      "absolute w-full h-full top-0 bottom-0 left-0 right-0 z-10";
    if (!dragActive) {
      return baseStyle;
    }

    return `${baseStyle} border-2 border-blue-400 border-dashed`;
  }, [dragActive]);

  return (
    <div onDragEnter={handleImageDrag} className="w-full h-full relative">
      {children}
      <input
        type="file"
        id="meme-upload"
        onChange={handleImageChange}
        hidden
        ref={fileInputRef}
      />
      {dragActive && (
        <div
          onDragEnter={handleImageDrag}
          onDragLeave={handleImageDrag}
          onDragOver={handleImageDrag}
          onDrop={handleImageDrop}
          className={overlayStyle}
        ></div>
      )}
    </div>
  );
};

export default MemeDropArea;
