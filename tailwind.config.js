/** @type {import('tailwindcss').Config} */
export default {
  content: ["./server/**/*.{html,templ}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        accent: {
          light: "#2563eb", // or keep tailwindcss/colors if you want
          dark: "#60a5fa",
        },
      },
    },
  },
  plugins: [],
};
