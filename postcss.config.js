import tailwind from "@tailwindcss/postcss";
import autoprefixer from "autoprefixer";
import cssnano from "cssnano";

export default {
  plugins: [
    tailwind(),
    autoprefixer,
    cssnano({ preset: "default" }),
  ],
};
