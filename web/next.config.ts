import type { NextConfig } from "next";

// With output: "standalone", rewrites are fixed at build time. In production, pass SERVER_URL as a build arg.
const serverUrl = process.env.SERVER_URL ?? "http://localhost:4000";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [{ source: "/api/v1/:path*", destination: `${serverUrl}/api/v1/:path*` }];
  },
};

export default nextConfig;
