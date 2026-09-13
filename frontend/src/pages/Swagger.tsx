import { Box } from "@mui/material";
import normalizeUrl from "normalize-url";
import type React from "react";
import { useEffect, useState } from "react";
import { apiUrl } from "../store/emptyApi";

// Allow the custom web component <openapi-explorer> in TSX
const OpenApiExplorer = "openapi-explorer" as unknown as React.ElementType;

export function Swagger() {
  const [loaded, setLoaded] = useState(false);

  // Wait for custom element to be defined
  useEffect(() => {
    if ((globalThis as unknown as { __TEST__?: boolean }).__TEST__) {
      // In tests, mark as loaded immediately to show the overview
      setLoaded(true);
      return;
    }
    if (typeof window === "undefined" || !window.customElements) {
      setLoaded(true);
      return;
    }
    // Load Prism and openapi-explorer lazily on mount only. This module is
    // statically imported by the NavBar, so importing at module top level
    // would pull the Lit bundle (and its dev-mode console warning) into
    // every page of the app.
    let cancelled = false;
    import("prismjs").then((Prism) => {
      if (cancelled) return;
      (window as unknown as { Prism?: unknown }).Prism = Prism.default || Prism;
      // Load the openapi-explorer after Prism is available
      return import("openapi-explorer");
    });
    window.customElements
      .whenDefined("openapi-explorer")
      .then(() => {
        if (cancelled) return;
        console.debug("openapi-explorer custom element is ready");
        setLoaded(true);
      })
      .catch((err) => {
        if (cancelled) return;
        console.error("Error waiting for openapi-explorer:", err);
        setLoaded(true);
      });

    // Fallback timeout in case whenDefined doesn't resolve
    const timeout = setTimeout(() => {
      console.debug("openapi-explorer timeout, marking as loaded anyway");
      setLoaded(true);
    }, 2000);

    return () => {
      cancelled = true;
      clearTimeout(timeout);
    };
  }, []);

  return (
    <Box
      sx={{
        height: "100%",
        width: "100%",
        display: "flex",
        flexDirection: "column",
      }}
    >
      {loaded ? (
        <OpenApiExplorer
          data-testid="openapi-explorer"
          spec-url={normalizeUrl(`${apiUrl}/openapi.yaml`)}
          style={{ flex: 1, minHeight: 0 }}
        >
          <div slot="overview">
            <h1>API Documentation</h1>
            <p>
              <a href={normalizeUrl(`${apiUrl}/openapi.json`)}>JSON</a> |{" "}
              <a href={normalizeUrl(`${apiUrl}/openapi.yaml`)}>YAML</a>
            </p>
          </div>
        </OpenApiExplorer>
      ) : (
        <Box sx={{ p: 2 }}>
          <h1>API Documentation</h1>
          <p>Loading OpenAPI explorer...</p>
        </Box>
      )}
    </Box>
  );
}
