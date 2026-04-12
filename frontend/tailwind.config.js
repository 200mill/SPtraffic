/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        bus: '#3b82f6',   // blue-500
        rail: '#ef4444',  // red-500
      },
    },
  },
  plugins: [],
}
