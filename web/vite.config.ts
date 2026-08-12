import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

const documentSecurityHeaders = {
  "Content-Security-Policy": "frame-ancestors 'none'",
  "X-Frame-Options": "DENY",
};

export default defineConfig(({ mode }) => {
  const backend = loadEnv(mode, ".", "VITE_").VITE_API_TARGET || "http://localhost:18080";
  return {
    plugins: [react()],
    server: {
      headers: documentSecurityHeaders,
      proxy: {
        "/v1": { target: backend, ws: true },
        "/healthz": backend,
        "/readyz": backend,
        "/version": backend,
      },
    },
    preview: { headers: documentSecurityHeaders },
  };
});
