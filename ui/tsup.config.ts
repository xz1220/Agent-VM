import {defineConfig} from "tsup";

export default defineConfig({
  entry: {
    "avm-ui": "src/main.tsx"
  },
  outDir: "dist",
  outExtension: () => ({js: ".js"}),
  clean: true,
  format: ["esm"],
  platform: "node",
  target: "node22",
  splitting: false,
  sourcemap: false,
  minify: false
});
