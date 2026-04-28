import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";

const Root = () => <main className="p-8 text-2xl">Prometheus V2 — Skeleton</main>;

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>
);
