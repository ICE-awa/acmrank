import { createBrowserRouter } from "react-router-dom";

import { RootPage } from "@/routes/root";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <RootPage />,
  },
]);
