import type { NextConfig } from "next";

const proxyTarget = process.env.API_PROXY_TARGET ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  async rewrites() {
    return [
      { source: "/health/:path*", destination: `${proxyTarget}/health/:path*` },
      { source: "/openapi.yaml", destination: `${proxyTarget}/openapi.yaml` },
    ];
  },
};

export default nextConfig;
