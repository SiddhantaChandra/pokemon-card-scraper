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
    // Determine backend URL based on environment
    const isDevelopment = process.env.NODE_ENV === 'development';
    const backendUrl = process.env.BACKEND_URL || 
                      (isDevelopment ? 'http://localhost:8080' : 'http://backend:8080');
    
    console.log(`Environment: ${process.env.NODE_ENV}, Backend URL: ${backendUrl}`);
    
    return [
      {
        source: '/api/:path*',
        destination: `${backendUrl}/api/:path*`,
      },
      {
        source: '/health',
        destination: `${backendUrl}/health`,
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

  // Optimize for production
  poweredByHeader: false,
  compress: true,
};

export default nextConfig;
