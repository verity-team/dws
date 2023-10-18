import type { Config } from "tailwindcss";

export const customColors = {
  cblack: "#1a1b1f",
  cred: "#ee382d",
  cgreen: "#FDEEA9",
  cblue: "#F4F4F4",
};

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  plugins: [],
  theme: {
    extend: {
      colors: customColors,
      lineHeight: {
        "loose-xl": "3.75rem",
        "loose-2xl": "5rem",
      },
    },
  },
};
export default config;
