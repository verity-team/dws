/** @type {import('next').NextConfig} */
const nextConfig = {
  // webpack: (config) => {
  //   config.module.rules?.push({
  //     test: /src\/app\/api/,
  //     loader: "ignore-loader",
  //   });
  //   return config;
  // },
  webpack: (config) => {
    config.externals.push("pino-pretty", "lokijs", "encoding");
    return config;
  },
  images: {
    domains: ["localhost"],
  },
  compiler: {
    removeConsole: process.env.NODE_ENV === "production",
  },
  output: "standalone",
};

module.exports = nextConfig;
