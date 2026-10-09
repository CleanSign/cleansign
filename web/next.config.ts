import type { NextConfig } from "next";

// output: "standalone"에서는 rewrites가 빌드 시점에 고정된다. 운영에서는 SERVER_URL을 빌드 인자로 준다.
const serverUrl = process.env.SERVER_URL ?? "http://localhost:4000";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [{ source: "/api/v1/:path*", destination: `${serverUrl}/api/v1/:path*` }];
  },
};

export default nextConfig;
