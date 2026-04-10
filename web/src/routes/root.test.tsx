import { fireEvent, render, screen } from "@testing-library/react";

import { RootPage } from "./root";

describe("RootPage", () => {
  it("renders the preview lab and switches between style directions", () => {
    render(<RootPage />);

    expect(
      screen.getByRole("heading", { name: "ACMRank Frontend Preview Lab" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Ink Stone/i })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(
      screen.getByRole("button", { name: /Paper Column/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Steel Frame/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/黑白灰的训练档案页/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Paper Column/i }));

    expect(
      screen.getByRole("button", { name: /Paper Column/i }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText(/白底黑字，像公开档案册/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Steel Frame/i }));

    expect(
      screen.getByRole("button", { name: /Steel Frame/i }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText(/中性、规整、克制/)).toBeInTheDocument();
  });
});
