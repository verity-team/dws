"use client";

import Image from "next/image";
import SlideshowItem from "./SlideshowItem";
import { useEffect, useState } from "react";

const memes = [
  "/memes/truth-meme-1.jpg",
  "/memes/truth-meme-2.jpg",
  "/memes/truth-meme-3.jpg",
  "/memes/truth-meme-4.jpg",
  "/memes/truth-meme-5.jpg",
  "/memes/truth-meme-6.jpg",
  "/memes/truth-meme-7.jpg",
  "/memes/truth-meme-8.jpg",
  "/memes/truth-meme-9.jpg",
  "/memes/truth-meme-10.jpg",
  "/memes/truth-meme-11.jpg",
];

const MemeSlideshow = () => {
  const [currentItem, setCurrentItem] = useState(0);

  useEffect(() => {
    const timerId = setInterval(() => {
      setCurrentItem((current) => (current + 1) % memes.length);
    }, 2000);

    return () => clearInterval(timerId);
  }, []);

  return (
    <div className="bg-cgreen">
      <div className="flex flex-1 items-center justify-center overflow-hidden">
        <div className="static whitespace-nowrap overflow-visible">
          <div className="relative w-[400px] h-[400px] left-0">
            {memes.map((src, index) => (
              <SlideshowItem
                isActive={index === currentItem}
                currentItem={currentItem}
                key={src}
              >
                <Image
                  src={src}
                  alt={`meme number ${index + 1}`}
                  width={0}
                  height={0}
                  sizes="100vw"
                  className="border-4 border-black box-border rounded-lg w-auto h-auto max-h-full object-contain"
                />
              </SlideshowItem>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

export default MemeSlideshow;
