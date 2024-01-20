"use client"; // Error components must be Client Components

import Image from "next/image";
import { useEffect } from "react";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Log the error to an error reporting service
    console.error(error);
  }, [error]);

  const handleRetry = () => {
    reset();
  };

  return (
    <div className="container mx-auto flex flex-col items-center justify-center space-y-4">
      <Image
        src="/dws-images/logo.png"
        alt="eye of truth"
        width={300}
        height={300}
      />
      <h2 className="text-4xl">Something went wrong!</h2>
      <button
        onClick={handleRetry}
        className="px-4 py-2 rounded-lg bg-red-500 hover:bg-red-600 text-white"
      >
        Try again
      </button>
    </div>
  );
}
