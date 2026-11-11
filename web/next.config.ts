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
  images: {
    // Company logos are hosted on Cloudinary; placehold.co is used by the local seed.
    remotePatterns: [
      { protocol: "https", hostname: "res.cloudinary.com" },
      { protocol: "https", hostname: "placehold.co" },
    ],
  },
  async redirects() {
    // Keep links to the old React app's URLs working.
    return [{ source: "/apply-job/:id", destination: "/jobs/:id", permanent: true }];
  },
};

export default nextConfig;
