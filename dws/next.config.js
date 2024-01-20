/** @type {import('next').NextConfig} */
const nextConfig = {
  webpack: (config) => {
    config.externals.push("pino-pretty", "lokijs", "encoding");
    return config;
  },
  compiler: {
    removeConsole: process.env.NODE_ENV === "production",
  },
  experimental: {
    outputFileTracingIncludes: {
      "/image": ["./node_modules/@img/**"],
    },
  },
  output: "standalone",
};

module.exports = nextConfig;
