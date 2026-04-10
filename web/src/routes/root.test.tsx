import { fireEvent, render, screen } from "@testing-library/react";

import { RootPage } from "./root";

describe("RootPage", () => {
  it("renders the preview lab and switches between style directions", () => {
    render(<RootPage />);

    expect(
      screen.getByRole("heading", { name: "ACMRank 风格预览" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /门户布局/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(
      screen.getByRole("button", { name: /个人布局/ }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /榜单布局/ }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/门户布局：总览 \+ 公告 \+ 榜单/),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /个人布局/ }));

    expect(screen.getByRole("button", { name: /个人布局/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(
      screen.getByText(/个人布局：信息卡 \+ 曲线 \+ 过题/),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /榜单布局/ }));

    expect(screen.getByRole("button", { name: /榜单布局/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByText(/榜单布局：排行榜优先/)).toBeInTheDocument();
  });
});
