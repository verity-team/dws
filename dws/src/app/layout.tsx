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
  title: "TruthMemes",
  description: "United by Meme",
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
