import { render, screen } from "@testing-library/react";

import { RootPage } from "./root";

describe("RootPage", () => {
  it("renders the bootstrap headline and product highlights", () => {
    render(<RootPage />);

    expect(
      screen.getByRole("heading", { name: "ACMRank" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/SCNU Rating 折线图与每日新 AC 热力图入口/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Codeforces \/ AtCoder \/ 洛谷 聚合 AC 结果/),
    ).toBeInTheDocument();
  });
});
