import { Changa_One, Roboto } from "next/font/google";
import "./globals.css";
import type { Metadata } from "next";

export const changa_one = Changa_One({
  subsets: ["latin"],
  weight: "400",
  style: "normal",
  display: "swap",
  variable: "--font-changa",
});

export const roboto = Roboto({
  subsets: ["latin"],
  weight: ["100", "300", "400", "500", "700"],
  style: "normal",
  display: "swap",
  variable: "--font-roboto",
});

export const metadata: Metadata = {
  title: "Truth Memes",
  description: "We memefy counterculture - Join the Meme ®Evolution!",
  openGraph: {
    title: "Truth Memes",
    description: "We memefy counterculture. Join the Meme ®Evolution!",
    images: ["/images/social-media-banner.png"],
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Truth Memes",
    description: "Join the Truthmemes army and deliver the truth to everyone!",
    images: ["/images/social-media-banner.png"],
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
        className={`${changa_one.variable} ${roboto.variable} ${changa_one.className}`}
      >
        {children}
      </body>
    </html>
  );
}
