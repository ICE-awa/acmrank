import { render, screen } from "@testing-library/react";

import { RootPage } from "./root";

describe("RootPage", () => {
  it("renders the ACMRank dashboard preview", () => {
    render(<RootPage />);

    expect(
      screen.getByRole("heading", { name: "我的主页" }),
    ).toBeInTheDocument();
    expect(screen.getByText("搜索用户、题号或比赛 ID")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "SCNU Rating" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Daily New AC" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "最新动态" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "排行榜" })).toBeInTheDocument();
  });
});
