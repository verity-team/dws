import { Changa_One, Roboto } from "next/font/google";

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
