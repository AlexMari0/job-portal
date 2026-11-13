import type { NextConfig } from "next";

// Uploads stored on disk by the Go API (LOCAL_UPLOAD_DIR) in development and
// e2e tests, served at <origin>/uploads/.
const localUploads = (() => {
  const origin = process.env.LOCAL_UPLOADS_ORIGIN;
  if (!origin) return { patterns: [] };
  const url = new URL(origin);
  return {
    patterns: [
      {
        protocol: url.protocol.replace(":", "") as "http" | "https",
        hostname: url.hostname,
        port: url.port,
        pathname: "/uploads/**",
      },
    ],
  };
})();

const nextConfig: NextConfig = {
  cacheComponents: true,
  partialPrefetching: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
  experimental: {
    // Resume (2 MB) and logo (1 MB) uploads pass through Server Actions to the
    // Go API; the default limit is 1 MB. Vercel's own cap is 4.5 MB.
    serverActions: { bodySizeLimit: "3mb" },
  },
  images: {
    // Company logos are hosted on Cloudinary; placehold.co is used by the local seed.
    remotePatterns: [
      { protocol: "https", hostname: "res.cloudinary.com" },
      { protocol: "https", hostname: "placehold.co" },
      { protocol: "https", hostname: "img.clerk.com" },
      ...localUploads.patterns,
    ],
    // Only when LOCAL_UPLOADS_ORIGIN is set (development / e2e): the Go API
    // on localhost is a private address, which the optimizer refuses by
    // default to prevent SSRF. Never set in production.
    dangerouslyAllowLocalIP: localUploads.patterns.length > 0,
  },
  async redirects() {
    // Keep links to the old React app's URLs working.
    return [{ source: "/apply-job/:id", destination: "/jobs/:id", permanent: true }];
  },
};

export default nextConfig;
