/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: '#040b16', // Deep almost-black blue
        surface: '#0a192f',    // Deep slate blue
        surfaceHighlight: '#112240', // Lighter slate for borders/hovers
        primary: '#64ffda',    // Electric teal accent
        secondary: '#0070f3',  // Deep blue accent
        textMain: '#ccd6f6',   // Soft white/blue text
        textMuted: '#8892b0',  // Muted grey/blue text
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
      },
      animation: {
        'gradient-x': 'gradient-x 15s ease infinite',
        aurora: "aurora 60s linear infinite",
        shimmer: "shimmer 2s linear infinite",
        meteor: "meteor 5s linear infinite",
      },
      keyframes: {
        'gradient-x': {
          '0%, 100%': { 'background-size': '200% 200%', 'background-position': 'left center' },
          '50%': { 'background-size': '200% 200%', 'background-position': 'right center' }
        },
        aurora: {
          from: { backgroundPosition: "50% 50%, 50% 50%" },
          to: { backgroundPosition: "350% 50%, 350% 50%" },
        },
        shimmer: {
          from: { backgroundPosition: "0 0" },
          to: { backgroundPosition: "-200% 0" },
        },
        meteor: {
          "0%": { transform: "rotate(215deg) translateX(0)", opacity: "1" },
          "70%": { opacity: "1" },
          "100%": { transform: "rotate(215deg) translateX(-500px)", opacity: "0" },
        },
      }
    },
  },
  plugins: [],
}
