import type { NextConfig } from "next";
import { resolveApiBaseUrl } from "./src/lib/config/api-base-url";

const apiBaseUrl = resolveApiBaseUrl(process.env);

const nextConfig: NextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  async rewrites() {
    return [
      { source: "/health/:path*", destination: `${apiBaseUrl}/health/:path*` },
      { source: "/openapi.yaml", destination: `${apiBaseUrl}/openapi.yaml` },
      { source: "/ws/message", destination: `${apiBaseUrl}/ws/message` },
    ];
  },
};

export default nextConfig;
