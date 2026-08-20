/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Output standalone supaya image Docker kecil dan `docker compose up`
  // tidak perlu menyalin seluruh node_modules.
  output: 'standalone',
};

module.exports = nextConfig;
