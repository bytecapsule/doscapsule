import { defineConfig } from "vite";
import solid from "vite-plugin-solid";

export default defineConfig({
  plugins: [solid()],
  clearScreen: false,
  build: {
    target: "es2021",
    outDir: "../cmd/doscapsule/dist",
    emptyOutDir: true,
  },
});
