import type { NextConfig } from "next";

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
    ],
  },
  async redirects() {
    // Keep links to the old React app's URLs working.
    return [{ source: "/apply-job/:id", destination: "/jobs/:id", permanent: true }];
  },
};

export default nextConfig;
