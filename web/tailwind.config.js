/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', '"Noto Sans SC"', '"Microsoft YaHei"', 'system-ui', 'sans-serif'],
      },
      colors: {
        accent: {
          DEFAULT: '#10b981',
          soft: '#34d399',
          deep: '#059669',
        },
      },
      boxShadow: {
        glass: '0 18px 50px -20px rgba(28, 36, 56, 0.28)',
        card: '0 10px 30px -18px rgba(28, 36, 56, 0.35)',
      },
    },
  },
  plugins: [],
}
