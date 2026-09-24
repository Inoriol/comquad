import React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app";
import "@patternfly/patternfly/patternfly.css";
import "@patternfly/patternfly/utilities/Accessibility/accessibility.css";

const root = createRoot(document.getElementById("app")!);
root.render(<App />);
