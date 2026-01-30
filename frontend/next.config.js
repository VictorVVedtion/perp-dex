/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Only enable API rewrites if REST_URL is configured (local development)
  async rewrites() {
    const restUrl = process.env.NEXT_PUBLIC_REST_URL
    if (restUrl && restUrl !== '') {
      return [
        {
          source: '/api/:path*',
          destination: `${restUrl}/:path*`, // Cosmos SDK REST API
        },
      ]
    }
    return []
  },
  // Vercel deployment optimization
  output: 'standalone',
}

module.exports = nextConfig
