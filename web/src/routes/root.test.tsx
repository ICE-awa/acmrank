import { fireEvent, render, screen } from "@testing-library/react";

import { RootPage } from "./root";

describe("RootPage", () => {
  it("renders the preview lab and switches between style directions", () => {
    render(<RootPage />);

    expect(
      screen.getByRole("heading", { name: "ACMRank Frontend Preview Lab" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Signal Lab/i })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(
      screen.getByRole("button", { name: /Archive Ledger/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Trackside Pulse/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Contest Signals, Clean Facts/),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Archive Ledger/i }));

    expect(
      screen.getByRole("button", { name: /Archive Ledger/i }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText(/训练档案像校史馆一样可靠/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Trackside Pulse/i }));

    expect(
      screen.getByRole("button", { name: /Trackside Pulse/i }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(
      screen.getByText(/把训练曲线做成一张有速度感的竞赛战报/),
    ).toBeInTheDocument();
  });
});
