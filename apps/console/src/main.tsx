import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./app/App";
import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-sans-kr/400.css";
import "@fontsource/ibm-plex-sans-arabic/400.css";
import "@fontsource/ibm-plex-mono/400.css";
import "../../../packages/design-system/styles/tokens.css";
import "../../../packages/design-system/styles/console.css";
import "./shared/styles.css";
import "./shared/screen-layout.css";
import "./app/header.css";
import "./app/entry.css";
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
