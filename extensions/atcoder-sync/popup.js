const usernameInput = document.getElementById("username");
const recentContestCountInput = document.getElementById("recentContestCount");
const fetchButton = document.getElementById("fetchButton");
const copyJsonButton = document.getElementById("copyJsonButton");
const copySolvedButton = document.getElementById("copySolvedButton");
const refreshLoginButton = document.getElementById("refreshLoginButton");
const loginBadge = document.getElementById("loginBadge");
const loginText = document.getElementById("loginText");
const messageNode = document.getElementById("message");
const summaryNode = document.getElementById("summary");
const jsonOutputNode = document.getElementById("jsonOutput");

let latestResult = null;

document.addEventListener("DOMContentLoaded", async () => {
  await restoreFormState();
  await refreshLoginState();
});

fetchButton.addEventListener("click", async () => {
  const user = usernameInput.value.trim();
  const recentContestCount = clampRecentContestCount(recentContestCountInput.value);

  if (!user) {
    setMessage("请输入 AtCoder 用户名。", "error");
    usernameInput.focus();
    return;
  }

  recentContestCountInput.value = String(recentContestCount);
  await saveFormState({ user, recentContestCount });

  setLoading(true);
  setMessage("正在抓取 AtCoder 历史与提交页……");

  try {
    const response = await chrome.runtime.sendMessage({
      type: "ATCODER_SYNC_FETCH_DATA",
      payload: {
        user,
        recentContestCount,
      },
    });

    if (!response || !response.ok) {
      throw new Error(response?.error || "扩展后台未返回有效结果。");
    }

    latestResult = response.data;
    renderResult(latestResult);
    setMessage("抓取完成。", "success");
  } catch (error) {
    latestResult = null;
    renderResult(null);
    setMessage(error.message || "抓取失败。", "error");
  } finally {
    setLoading(false);
  }
});

copyJsonButton.addEventListener("click", async () => {
  if (!latestResult) {
    return;
  }

  try {
    await copyText(JSON.stringify(latestResult, null, 2));
    setMessage("JSON 已复制。", "success");
  } catch (error) {
    setMessage(error.message || "复制 JSON 失败。", "error");
  }
});

copySolvedButton.addEventListener("click", async () => {
  if (!latestResult) {
    return;
  }

  try {
    const solved = Array.isArray(latestResult.solved_problem_ids)
      ? latestResult.solved_problem_ids.join("\n")
      : "";
    await copyText(solved);
    setMessage("AC 题号已复制。", "success");
  } catch (error) {
    setMessage(error.message || "复制 AC 题号失败。", "error");
  }
});

refreshLoginButton.addEventListener("click", async () => {
  await refreshLoginState();
});

async function restoreFormState() {
  const { atcoderSyncFormState } = await chrome.storage.local.get("atcoderSyncFormState");

  if (!atcoderSyncFormState) {
    return;
  }

  if (atcoderSyncFormState.user) {
    usernameInput.value = atcoderSyncFormState.user;
  }

  if (atcoderSyncFormState.recentContestCount) {
    recentContestCountInput.value = String(
      clampRecentContestCount(atcoderSyncFormState.recentContestCount)
    );
  }
}

async function saveFormState({ user, recentContestCount }) {
  await chrome.storage.local.set({
    atcoderSyncFormState: {
      user,
      recentContestCount,
    },
  });
}

async function refreshLoginState() {
  setMessage("正在检查 AtCoder 登录状态……");

  try {
    const response = await chrome.runtime.sendMessage({
      type: "ATCODER_SYNC_CHECK_LOGIN",
    });

    if (!response || !response.ok) {
      throw new Error(response?.error || "无法检查登录状态。");
    }

    const loggedIn = Boolean(response.loggedIn);
    loginBadge.classList.toggle("is-success", loggedIn);
    loginBadge.classList.toggle("is-error", !loggedIn);
    loginText.textContent = loggedIn ? "已检测到登录态" : "未登录";

    if (loggedIn) {
      setMessage("已检测到 AtCoder 登录态。", "success");
    } else {
      setMessage("未检测到 AtCoder 登录态，请先登录 atcoder.jp。", "error");
    }
  } catch (error) {
    loginBadge.classList.remove("is-success");
    loginBadge.classList.add("is-error");
    loginText.textContent = "检测失败";
    setMessage(error.message || "登录状态检查失败。", "error");
  }
}

function renderResult(result) {
  if (!result) {
    summaryNode.textContent = "尚未抓取。";
    jsonOutputNode.textContent = "{}";
    copyJsonButton.disabled = true;
    copySolvedButton.disabled = true;
    return;
  }

  const latest = result.latest_submission;
  const latestText = latest
    ? [
        `用户: ${result.user}`,
        `最新提交比赛: ${latest.contest || "-"}`,
        `最新题号: ${latest.problem_id || "-"}`,
        `题目名: ${latest.problem_name || "-"}`,
        `结果: ${latest.result || "-"}`,
        `时间: ${latest.submitted_at || "-"}`,
        `详情: ${latest.detail_url || "-"}`,
        `最近扫描比赛数: ${result.recent_contests_scanned}`,
        `去重 AC 题号数: ${(result.solved_problem_ids || []).length}`,
      ].join("\n")
    : [
        `用户: ${result.user}`,
        "最近扫描比赛里没有解析到提交记录。",
        `最近扫描比赛数: ${result.recent_contests_scanned}`,
        `去重 AC 题号数: ${(result.solved_problem_ids || []).length}`,
      ].join("\n");

  summaryNode.textContent = latestText;
  jsonOutputNode.textContent = JSON.stringify(result, null, 2);
  copyJsonButton.disabled = false;
  copySolvedButton.disabled = false;
}

function setLoading(isLoading) {
  fetchButton.disabled = isLoading;
  refreshLoginButton.disabled = isLoading;
  usernameInput.disabled = isLoading;
  recentContestCountInput.disabled = isLoading;
}

function setMessage(text, type = "") {
  messageNode.textContent = text;
  messageNode.classList.remove("is-error", "is-success");

  if (type === "error") {
    messageNode.classList.add("is-error");
  }

  if (type === "success") {
    messageNode.classList.add("is-success");
  }
}

function clampRecentContestCount(value) {
  const parsed = Number.parseInt(value, 10);

  if (!Number.isFinite(parsed)) {
    return 5;
  }

  return Math.min(Math.max(parsed, 1), 10);
}

async function copyText(text) {
  if (!text) {
    throw new Error("没有可复制的内容。");
  }

  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }

  const fallbackInput = document.createElement("textarea");
  fallbackInput.value = text;
  document.body.appendChild(fallbackInput);
  fallbackInput.select();
  document.execCommand("copy");
  fallbackInput.remove();
}
