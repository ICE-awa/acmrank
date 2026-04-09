const ATCODER_ORIGIN = "https://atcoder.jp";
const ATCODER_TAB_PATTERN = `${ATCODER_ORIGIN}/*`;
const HOME_URL = `${ATCODER_ORIGIN}/`;
const LOGIN_COOKIE_NAME = "REVEL_SESSION";

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (!message || !message.type) {
    return false;
  }

  if (message.type === "ATCODER_SYNC_CHECK_LOGIN") {
    handleCheckLogin()
      .then((result) => sendResponse({ ok: true, ...result }))
      .catch((error) => sendResponse({ ok: false, error: error.message }));
    return true;
  }

  if (message.type === "ATCODER_SYNC_FETCH_DATA") {
    handleFetchData(message.payload || {})
      .then((result) => sendResponse({ ok: true, data: result }))
      .catch((error) => sendResponse({ ok: false, error: error.message }));
    return true;
  }

  return false;
});

async function handleCheckLogin() {
  const cookie = await chrome.cookies.get({
    url: ATCODER_ORIGIN,
    name: LOGIN_COOKIE_NAME,
  });

  return {
    loggedIn: Boolean(cookie),
    checkedAt: new Date().toISOString(),
  };
}

async function handleFetchData(payload) {
  const user = String(payload.user || "").trim();
  const recentContestCount = clampRecentContestCount(payload.recentContestCount);

  if (!user) {
    throw new Error("请输入 AtCoder 用户名。");
  }

  const loginState = await handleCheckLogin();
  if (!loginState.loggedIn) {
    throw new Error("未检测到 AtCoder 登录态，请先在当前浏览器中登录 atcoder.jp。");
  }

  const tabState = await ensureAtCoderTab();

  try {
    await waitForTabComplete(tabState.tab.id);
    await ensureContentScript(tabState.tab.id);

    const response = await chrome.tabs.sendMessage(tabState.tab.id, {
      type: "ATCODER_SYNC_SCRAPE",
      payload: {
        user,
        recentContestCount,
      },
    });

    if (!response || !response.ok) {
      throw new Error(response?.error || "AtCoder 页面未返回有效数据。");
    }

    return {
      ...response.data,
      fetched_at: new Date().toISOString(),
      tab_reused: !tabState.created,
    };
  } finally {
    if (tabState.created && tabState.tab.id) {
      try {
        await chrome.tabs.remove(tabState.tab.id);
      } catch (_error) {
        // Ignore cleanup failure for temporary tabs.
      }
    }
  }
}

async function ensureAtCoderTab() {
  const existingTabs = await chrome.tabs.query({
    url: [ATCODER_TAB_PATTERN],
  });

  const reusableTab = existingTabs.find((tab) => tab.id);
  if (reusableTab) {
    return { tab: reusableTab, created: false };
  }

  const tab = await chrome.tabs.create({
    url: HOME_URL,
    active: false,
  });

  return { tab, created: true };
}

async function waitForTabComplete(tabId, timeoutMs = 15000) {
  const tab = await chrome.tabs.get(tabId);
  if (tab.status === "complete") {
    return;
  }

  await new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      chrome.tabs.onUpdated.removeListener(onUpdated);
      reject(new Error("等待 AtCoder 页签加载超时。"));
    }, timeoutMs);

    function onUpdated(updatedTabId, changeInfo) {
      if (updatedTabId !== tabId || changeInfo.status !== "complete") {
        return;
      }

      clearTimeout(timeout);
      chrome.tabs.onUpdated.removeListener(onUpdated);
      resolve();
    }

    chrome.tabs.onUpdated.addListener(onUpdated);
  });
}

async function ensureContentScript(tabId) {
  try {
    const ping = await chrome.tabs.sendMessage(tabId, {
      type: "ATCODER_SYNC_PING",
    });

    if (ping && ping.ok) {
      return;
    }
  } catch (_error) {
    // Fall through to manual injection.
  }

  await chrome.scripting.executeScript({
    target: { tabId },
    files: ["content-script.js"],
  });

  const ping = await chrome.tabs.sendMessage(tabId, {
    type: "ATCODER_SYNC_PING",
  });

  if (!ping || !ping.ok) {
    throw new Error("无法在 AtCoder 页中初始化抓取脚本。");
  }
}

function clampRecentContestCount(value) {
  const parsed = Number.parseInt(value, 10);

  if (!Number.isFinite(parsed)) {
    return 5;
  }

  return Math.min(Math.max(parsed, 1), 10);
}
