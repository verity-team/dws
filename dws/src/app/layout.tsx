import { changa_one } from "./fonts";
import "./globals.css";
import type { Metadata } from "next";

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
      <body className={changa_one.className}>{children}</body>
    </html>
  );
}
