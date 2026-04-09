(() => {
  if (window.__ATCODER_SYNC_CONTENT_SCRIPT__) {
    return;
  }
  window.__ATCODER_SYNC_CONTENT_SCRIPT__ = true;

  const ATCODER_ORIGIN = "https://atcoder.jp";
  const MAX_PAGES_PER_CONTEST = 20;

  chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    if (!message || !message.type) {
      return false;
    }

    if (message.type === "ATCODER_SYNC_PING") {
      sendResponse({ ok: true });
      return false;
    }

    if (message.type === "ATCODER_SYNC_SCRAPE") {
      collectAtCoderData(message.payload || {})
        .then((data) => sendResponse({ ok: true, data }))
        .catch((error) => sendResponse({ ok: false, error: error.message }));
      return true;
    }

    return false;
  });

  async function collectAtCoderData(payload) {
    const user = String(payload.user || "").trim();
    const recentContestCount = clampRecentContestCount(payload.recentContestCount);

    if (!user) {
      throw new Error("用户名不能为空。");
    }

    const history = await fetchHistory(user);
    const recentContests = selectRecentContests(history, recentContestCount);
    const contestResults = [];
    const solvedSet = new Set();
    let latestSubmission = null;

    for (const contest of recentContests) {
      const contestData = await scrapeContestSubmissions({
        contestId: contest.contest,
        contestName: contest.contest_name,
        user,
      });

      contestResults.push(contestData);

      for (const problemId of contestData.solved_problem_ids) {
        solvedSet.add(problemId);
      }

      if (
        contestData.latest_submission &&
        isNewerSubmission(contestData.latest_submission, latestSubmission)
      ) {
        latestSubmission = {
          user,
          contest: contestData.contest,
          contest_name: contestData.contest_name,
          ...contestData.latest_submission,
        };
      }
    }

    return {
      user,
      recent_contests_requested: recentContestCount,
      recent_contests_scanned: contestResults.length,
      history_count: history.length,
      history_sample: recentContests,
      latest_submission: latestSubmission,
      solved_problem_ids: Array.from(solvedSet).sort(),
      contests: contestResults,
      source_urls: {
        history: buildHistoryUrl(user),
      },
    };
  }

  async function fetchHistory(user) {
    const response = await fetch(buildHistoryUrl(user), {
      credentials: "include",
      cache: "no-store",
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      throw new Error(`获取比赛历史失败，状态码 ${response.status}。`);
    }

    const data = await response.json();
    if (!Array.isArray(data)) {
      throw new Error("比赛历史返回格式异常。");
    }

    return data;
  }

  async function scrapeContestSubmissions({ contestId, contestName, user }) {
    const firstPageUrl = buildSubmissionUrl({ contestId, user, page: 1 });
    const firstPage = await fetchSubmissionPage(firstPageUrl);
    const allRows = [...firstPage.rows];
    const detectedMaxPage = detectMaxPage(firstPage.document);
    const maxPage = Math.min(detectedMaxPage, MAX_PAGES_PER_CONTEST);
    let pagesScanned = 1;

    for (let page = 2; page <= maxPage; page += 1) {
      const pageResult = await fetchSubmissionPage(
        buildSubmissionUrl({ contestId, user, page })
      );

      if (pageResult.rows.length === 0) {
        break;
      }

      allRows.push(...pageResult.rows);
      pagesScanned = page;
    }

    const solvedSet = new Set();
    let latestSubmission = null;

    for (const row of allRows) {
      if (row.result === "AC" && row.problem_id) {
        solvedSet.add(row.problem_id);
      }

      if (isNewerSubmission(row, latestSubmission)) {
        latestSubmission = row;
      }
    }

    return {
      contest: contestId,
      contest_name: contestName,
      source_url: firstPageUrl,
      pages_scanned: pagesScanned,
      total_pages_detected: detectedMaxPage,
      submissions_scanned: allRows.length,
      solved_problem_ids: Array.from(solvedSet).sort(),
      latest_submission: latestSubmission,
    };
  }

  async function fetchSubmissionPage(url) {
    const response = await fetch(url, {
      credentials: "include",
      cache: "no-store",
      headers: {
        Accept: "text/html,application/xhtml+xml",
      },
    });

    const html = await response.text();

    if (!response.ok) {
      throw new Error(`获取提交页失败，状态码 ${response.status}。`);
    }

    if (response.url.includes("/login")) {
      throw new Error("AtCoder 提交页需要登录，但当前会话未通过验证。");
    }

    const documentNode = new DOMParser().parseFromString(html, "text/html");
    const loginMarker = documentNode.querySelector("form[action='/login']");
    if (loginMarker) {
      throw new Error("AtCoder 提交页返回了登录页面。");
    }

    return {
      document: documentNode,
      rows: parseSubmissionRows(documentNode),
    };
  }

  function parseSubmissionRows(documentNode) {
    const rows = Array.from(documentNode.querySelectorAll("table tbody tr"));

    return rows
      .map((row) => {
        const cells = Array.from(row.querySelectorAll("td"));
        if (cells.length < 7) {
          return null;
        }

        const taskLink = cells[1]?.querySelector("a[href*='/tasks/']");
        const detailLink = row.querySelector("a[href*='/submissions/']");
        const timeNode = cells[0]?.querySelector("time") || cells[0];
        const problemHref = taskLink?.getAttribute("href") || "";
        const problemId = problemHref.split("/").filter(Boolean).pop() || "";
        const submittedAt = normalizeAtCoderTime(timeNode?.textContent || "");
        const resultText = normalizeText(cells[6]?.textContent || "");

        return {
          problem_id: problemId,
          problem_name: normalizeText(taskLink?.textContent || ""),
          submitted_at: submittedAt,
          submitted_at_epoch: toEpoch(submittedAt),
          result: resultText,
          detail_url: detailLink
            ? new URL(detailLink.getAttribute("href"), ATCODER_ORIGIN).toString()
            : "",
        };
      })
      .filter(Boolean);
  }

  function detectMaxPage(documentNode) {
    const pageNumbers = Array.from(documentNode.querySelectorAll(".pagination a"))
      .map((link) => {
        const href = link.getAttribute("href");
        if (!href) {
          return null;
        }

        const page = new URL(href, ATCODER_ORIGIN).searchParams.get("page");
        const parsed = Number.parseInt(page || link.textContent, 10);
        return Number.isFinite(parsed) ? parsed : null;
      })
      .filter((value) => value !== null);

    return Math.max(1, ...pageNumbers);
  }

  function selectRecentContests(history, count) {
    const deduped = [];
    const seen = new Set();

    const sortedHistory = [...history].sort((left, right) => {
      return toEpoch(right.EndTime) - toEpoch(left.EndTime);
    });

    for (const item of sortedHistory) {
      const rawScreenName = normalizeText(item.ContestScreenName || "");
      const contest = normalizeContestId(rawScreenName);
      if (!contest || seen.has(contest)) {
        continue;
      }

      seen.add(contest);
      deduped.push({
        contest,
        raw_contest_screen_name: rawScreenName,
        contest_name: item.ContestName || contest,
        end_time: item.EndTime || "",
        rank: item.Place ?? null,
      });

      if (deduped.length >= count) {
        break;
      }
    }

    return deduped;
  }

  function buildHistoryUrl(user) {
    return `${ATCODER_ORIGIN}/users/${encodeURIComponent(user)}/history/json`;
  }

  function buildSubmissionUrl({ contestId, user, page }) {
    const url = new URL(`${ATCODER_ORIGIN}/contests/${contestId}/submissions`);
    url.searchParams.set("f.User", user);

    if (page > 1) {
      url.searchParams.set("page", String(page));
    }

    return url.toString();
  }

  function normalizeAtCoderTime(value) {
    const text = normalizeText(value);
    if (!text) {
      return "";
    }

    if (/ [+-]\d{4}$/.test(text)) {
      return text;
    }

    return text.replace(
      /(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})([+-]\d{4})$/,
      "$1 $2"
    );
  }

  function normalizeContestId(value) {
    const text = normalizeText(value);
    if (!text) {
      return "";
    }

    if (text.includes("://")) {
      try {
        return normalizeContestId(new URL(text).hostname);
      } catch (_error) {
        return "";
      }
    }

    if (text.includes("/")) {
      const segments = text.split("/").filter(Boolean);
      const contestIndex = segments.findIndex((segment) => segment === "contests");
      if (contestIndex >= 0 && segments[contestIndex + 1]) {
        return normalizeText(segments[contestIndex + 1]);
      }

      return normalizeText(segments[segments.length - 1] || "");
    }

    const hostMatch = text.match(/^([a-z0-9_-]+)\.contest\.atcoder\.jp$/i);
    if (hostMatch) {
      return normalizeText(hostMatch[1]);
    }

    return text;
  }

  function normalizeText(value) {
    return String(value || "").replace(/\s+/g, " ").trim();
  }

  function toEpoch(value) {
    const text = normalizeText(value);
    if (!text) {
      return 0;
    }

    const normalized = text
      .replace(" ", "T")
      .replace(/T(\d{2}:\d{2}:\d{2}) ([+-]\d{2})(\d{2})$/, "T$1$2:$3")
      .replace(/T(\d{2}:\d{2}:\d{2})([+-]\d{2})(\d{2})$/, "T$1$2:$3");

    const parsed = Date.parse(normalized);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function isNewerSubmission(candidate, current) {
    if (!candidate) {
      return false;
    }

    if (!current) {
      return true;
    }

    return (candidate.submitted_at_epoch || 0) > (current.submitted_at_epoch || 0);
  }

  function clampRecentContestCount(value) {
    const parsed = Number.parseInt(value, 10);

    if (!Number.isFinite(parsed)) {
      return 5;
    }

    return Math.min(Math.max(parsed, 1), 10);
  }
})();
