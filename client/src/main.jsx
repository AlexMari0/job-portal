import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.jsx";
import { BrowserRouter } from "react-router-dom";
import { AuthContextProvider } from "./context/AuthContext.jsx";
import { JobsContextProvider } from "./context/JobsContext.jsx";
import { SearchContextProvider } from "./context/SearchContext.jsx";
import { ClerkProvider } from "@clerk/react";

const PUBLISHABLE_KEY = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY;

if (!PUBLISHABLE_KEY) {
  throw new Error("Missing Publishable Key");
}

createRoot(document.getElementById("root")).render(
  <ClerkProvider publishableKey={PUBLISHABLE_KEY} afterSignOutUrl="/">
    <BrowserRouter>
      <AuthContextProvider>
        <JobsContextProvider>
          <SearchContextProvider>
            <App />
          </SearchContextProvider>
        </JobsContextProvider>
      </AuthContextProvider>
    </BrowserRouter>
  </ClerkProvider>,
);
