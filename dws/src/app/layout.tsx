import "./globals.css";
import type { Metadata } from "next";
import clsx from "clsx";
import { changa_one, roboto } from "./fonts";

export const metadata: Metadata = {
  title: "Truth Memes",
  description: "We memefy counterculture - Join the Meme ®Evolution!",
  openGraph: {
    title: "Truth Memes",
    description: "We memefy counterculture. Join the Meme ®Evolution!",
    images: ["/dws-images/social-media-banner.png"],
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Truth Memes",
    description: "Join the Truthmemes army and deliver the truth to everyone!",
    images: ["/dws-images/social-media-banner.png"],
  },
  metadataBase: process.env.NEXT_PUBLIC_HOST_URL
    ? new URL(process.env.NEXT_PUBLIC_HOST_URL)
    : new URL("https://truthmemes.io"),
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body
        className={clsx(
          changa_one.variable,
          roboto.variable,
          changa_one.className
        )}
      >
        {children}
      </body>
    </html>
  );
}
