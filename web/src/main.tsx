import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { EssayApp } from "@/app/EssayApp";
import "@/theme/index.css";

const root = document.getElementById("root");

if (!root) {
  throw new Error("root container not found");
}

createRoot(root).render(
  <StrictMode>
    <EssayApp />
  </StrictMode>
);
