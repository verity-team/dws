import type { Config } from "tailwindcss";

export const customColors = {
  cblack: "1a1b1f",
  cred: "#ee382d",
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
    },
  },
};
export default config;
