/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  
  // Image configuration to handle external images
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'torecacamp-pokemon.com',
        port: '',
        pathname: '/**',
      },
      {
        protocol: 'https',
        hostname: '*.torecacamp-pokemon.com',
        port: '',
        pathname: '/**',
      },
    ],
    // Allow unoptimized images for external sources
    unoptimized: true,
  },

  // API rewrites for backend communication
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://backend:8080/api/:path*', // Backend API via Docker service name
      },
    ];
  },

  // Headers configuration to handle CORS and image loading
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'Cross-Origin-Embedder-Policy',
            value: 'unsafe-none',
          },
          {
            key: 'Cross-Origin-Opener-Policy',
            value: 'same-origin-allow-popups',
          },
        ],
      },
    ];
  },
};

export default nextConfig;
