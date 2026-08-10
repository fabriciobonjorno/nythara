import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const backend = loadEnv(mode, ".", "VITE_").VITE_API_TARGET || "http://localhost:8080";
  return {
    plugins: [react()],
    server: {
      proxy: {
        "/v1": { target: backend, ws: true },
        "/healthz": backend,
        "/readyz": backend,
        "/version": backend,
      },
    },
  };
});
