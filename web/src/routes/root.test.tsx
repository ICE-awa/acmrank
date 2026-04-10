import { fireEvent, render, screen } from "@testing-library/react";

import { RootPage } from "./root";

describe("RootPage", () => {
  it("renders the preview lab and switches between style directions", () => {
    render(<RootPage />);

    expect(
      screen.getByRole("heading", { name: "ACMRank Frontend Preview Lab" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Luogu Portal/i }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(
      screen.getByRole("button", { name: /Luogu Profile/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Luogu Rankings/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/保留洛谷这套蓝灰白颜色/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Luogu Profile/i }));

    expect(
      screen.getByRole("button", { name: /Luogu Profile/i }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(
      screen.getByText(/同一套洛谷配色，换成更正常的“个人页优先”布局/),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Luogu Rankings/i }));

    expect(
      screen.getByRole("button", { name: /Luogu Rankings/i }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(
      screen.getByText(/用洛谷这套颜色，直接把榜单和趋势做成站点主视觉/),
    ).toBeInTheDocument();
  });
});
