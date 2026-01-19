/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:1317/:path*', // Cosmos SDK REST API
      },
    ]
  },
}

module.exports = nextConfig
