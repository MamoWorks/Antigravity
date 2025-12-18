/**
 * Antigravity 授权服务 - Deno 单文件应用
 * 使用 Express + Tailwind CSS 实现 Google OAuth 授权链接生成和 Token 获取
 */

import express from "npm:express@4.18.2";
import axios from "npm:axios@1.6.2";
import { randomBytes } from "node:crypto";

const app = express();
const PORT = 3001;

// Antigravity OAuth 配置
const ANTIGRAVITY_CLIENT_ID = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com";
const ANTIGRAVITY_CLIENT_SECRET = "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";
const ANTIGRAVITY_CALLBACK_PORT = 51121;

const ANTIGRAVITY_SCOPES = [
  "https://www.googleapis.com/auth/cloud-platform",
  "https://www.googleapis.com/auth/userinfo.email",
  "https://www.googleapis.com/auth/userinfo.profile",
  "https://www.googleapis.com/auth/cclog",
  "https://www.googleapis.com/auth/experimentsandconfigs",
];

/**
 * Base64 URL 安全编码
 */
function base64urlEncode(data) {
  return btoa(String.fromCharCode(...data))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

/**
 * 生成随机状态字符串
 */
function generateRandomState() {
  return base64urlEncode(randomBytes(32));
}

// 存储配置信息
let authConfig = null;

app.use(express.json());
app.use(express.urlencoded({ extended: true }));

/**
 * 主页路由 - 返回 HTML 界面
 */
app.get("/", (_req, res) => {
  res.send(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Antigravity 授权服务</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script>
    tailwind.config = {
      darkMode: 'class'
    }
  </script>
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Inter', 'SF Pro Display', 'Segoe UI', sans-serif;
    }
    .glass {
      backdrop-filter: blur(40px) saturate(180%);
      -webkit-backdrop-filter: blur(40px) saturate(180%);
    }
    .glass-strong {
      backdrop-filter: blur(60px) saturate(200%);
      -webkit-backdrop-filter: blur(60px) saturate(200%);
    }
    .bg-pattern {
      background-image:
        radial-gradient(circle at 20% 30%, rgba(234, 88, 12, 0.1) 0%, transparent 50%),
        radial-gradient(circle at 80% 70%, rgba(147, 51, 234, 0.08) 0%, transparent 50%),
        radial-gradient(circle at 40% 80%, rgba(234, 88, 12, 0.06) 0%, transparent 40%);
    }
    .dark .bg-pattern {
      background-image:
        radial-gradient(circle at 20% 30%, rgba(234, 88, 12, 0.15) 0%, transparent 50%),
        radial-gradient(circle at 80% 70%, rgba(147, 51, 234, 0.12) 0%, transparent 50%),
        radial-gradient(circle at 40% 80%, rgba(234, 88, 12, 0.1) 0%, transparent 40%);
    }
  </style>
</head>
<body class="bg-gray-100 dark:bg-gray-900 min-h-screen py-12 px-4 transition-colors duration-300 bg-pattern relative overflow-x-hidden">
  <!-- 夜间模式切换按钮 -->
  <div class="fixed bottom-6 right-6 z-50">
    <button id="themeToggle" class="glass bg-white/80 dark:bg-gray-800/80 p-2.5 rounded-full shadow-lg hover:shadow-xl transition-all duration-200 border border-gray-200/50 dark:border-gray-700/50">
      <svg id="sunIcon" class="w-5 h-5 text-gray-700 dark:text-gray-300 hidden dark:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path>
      </svg>
      <svg id="moonIcon" class="w-5 h-5 text-gray-700 dark:text-gray-300 block dark:hidden" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path>
      </svg>
    </button>
  </div>

  <div class="max-w-4xl mx-auto">
    <div class="glass-strong bg-white/70 dark:bg-gray-800/70 rounded-2xl shadow-2xl p-10 mb-8 border border-white/20 dark:border-gray-700/30 transition-colors duration-300">
      <h1 class="text-3xl font-semibold text-gray-900 dark:text-white mb-3 text-center">Antigravity 授权服务</h1>
      <p class="text-center text-gray-500 dark:text-gray-400 mb-10">Google Cloud Code Assist OAuth 授权</p>

      <!-- 步骤 1: 生成授权链接 -->
      <div id="step1" class="mb-10">
        <h2 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">1. 生成授权链接</h2>
        <button id="generateBtn" class="w-full bg-orange-500 hover:bg-orange-600 dark:bg-orange-600 dark:hover:bg-orange-700 text-white font-medium py-3 px-6 rounded-lg transition-all duration-200 shadow-sm hover:shadow active:scale-[0.98]">
          生成授权链接
        </button>
        <div id="authUrlResult" class="mt-4 hidden">
          <div class="glass bg-gray-50/80 dark:bg-gray-700/50 border border-gray-200/50 dark:border-gray-600/50 rounded-lg p-4">
            <p class="text-xs text-gray-500 dark:text-gray-400 mb-2 uppercase tracking-wide font-medium">授权链接</p>
            <div class="glass bg-white/80 dark:bg-gray-800/50 p-3 rounded border border-gray-200/30 dark:border-gray-600/30 break-all text-sm mb-3">
              <a id="authUrl" href="" target="_blank" class="text-orange-600 dark:text-orange-400 hover:underline"></a>
            </div>
            <button id="copyUrlBtn" class="w-full bg-white hover:bg-gray-50 dark:bg-gray-600 dark:hover:bg-gray-500 text-gray-800 dark:text-white text-sm font-medium py-2.5 px-5 rounded-lg transition-all duration-200 shadow-md hover:shadow-lg active:scale-[0.98] border border-gray-200 dark:border-gray-500">
              复制链接
            </button>
          </div>
        </div>
      </div>

      <!-- 步骤 2: 提取 Token (初始隐藏) -->
      <div id="step2" class="hidden">
        <h2 class="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">2. 提取并交换 Token</h2>
        <div class="glass bg-gray-50/80 dark:bg-gray-700/50 border border-gray-200/50 dark:border-gray-600/50 rounded-lg p-4 mb-4">
          <p class="text-sm text-gray-600 dark:text-gray-300 leading-relaxed">
            <span class="font-medium text-gray-900 dark:text-white">操作步骤：</span><br>
            * 点击上方生成的授权链接<br>
            * 使用 Google 账号登录并授权<br>
            * 授权后会跳转到无法访问的页面（正常现象）<br>
            * 复制浏览器地址栏中的完整 URL 并粘贴到下方
          </p>
        </div>
        <textarea
          id="callbackUrl"
          placeholder="粘贴授权后跳转的完整 URL..."
          class="w-full h-24 p-3.5 glass bg-white/80 dark:bg-gray-700/50 text-gray-900 dark:text-gray-100 border border-gray-200/50 dark:border-gray-600/50 rounded-lg focus:ring-2 focus:ring-orange-500/50 dark:focus:ring-orange-500/50 focus:border-transparent mb-4 transition-all duration-200 placeholder-gray-400 dark:placeholder-gray-500 text-sm"
        ></textarea>
        <button id="extractBtn" class="w-full bg-orange-500 hover:bg-orange-600 dark:bg-orange-600 dark:hover:bg-orange-700 text-white font-medium py-3 px-6 rounded-lg transition-all duration-200 shadow-sm hover:shadow active:scale-[0.98]">
          提取 Token
        </button>
        <div id="tokenResult" class="mt-4 hidden">
          <div class="glass bg-gray-50/80 dark:bg-gray-700/50 border border-gray-200/50 dark:border-gray-600/50 rounded-lg p-4">
            <p class="text-xs text-gray-500 dark:text-gray-400 mb-3 uppercase tracking-wide font-medium">凭证信息</p>
            <div class="space-y-2.5">
              <div class="glass bg-white/80 dark:bg-gray-800/50 p-3 rounded border border-gray-200/30 dark:border-gray-600/30">
                <p class="text-xs text-gray-500 dark:text-gray-400 mb-1.5">Email</p>
                <p id="email" class="text-xs font-mono text-gray-900 dark:text-gray-100 whitespace-nowrap overflow-x-auto"></p>
              </div>
              <div class="glass bg-white/80 dark:bg-gray-800/50 p-3 rounded border border-gray-200/30 dark:border-gray-600/30">
                <p class="text-xs text-gray-500 dark:text-gray-400 mb-1.5">Project ID</p>
                <p id="projectId" class="text-xs font-mono text-gray-900 dark:text-gray-100 whitespace-nowrap overflow-x-auto"></p>
              </div>
              <div class="glass bg-white/80 dark:bg-gray-800/50 p-3 rounded border border-gray-200/30 dark:border-gray-600/30">
                <p class="text-xs text-gray-500 dark:text-gray-400 mb-1.5">Refresh Token</p>
                <p id="refreshToken" class="text-xs font-mono text-gray-900 dark:text-gray-100 break-all"></p>
              </div>
            </div>
            <button id="copyRtBtn" class="w-full mt-3 bg-white hover:bg-gray-50 dark:bg-gray-600 dark:hover:bg-gray-500 text-gray-800 dark:text-white text-sm font-medium py-2.5 px-5 rounded-lg transition-all duration-200 shadow-md hover:shadow-lg active:scale-[0.98] border border-gray-200 dark:border-gray-500">
              复制 Refresh Token
            </button>
            <p class="text-xs text-gray-500 dark:text-gray-400 text-center mt-2">Refresh Token 用作 Antigravity Proxy 的身份校验 KEY</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 错误提示 -->
    <div id="errorMsg" class="hidden glass bg-white/70 dark:bg-gray-800/70 border border-gray-300/50 dark:border-gray-600/50 rounded-lg p-4 mb-4">
      <p class="text-gray-800 dark:text-gray-200"></p>
    </div>

    <!-- 加载提示 -->
    <div id="loading" class="hidden fixed inset-0 bg-black/10 dark:bg-black/40 backdrop-blur flex items-center justify-center z-50">
      <div class="glass-strong bg-white/90 dark:bg-gray-800/90 rounded-xl p-6 flex items-center space-x-3 border border-gray-200/30 dark:border-gray-700/30 shadow-2xl">
        <div class="animate-spin rounded-full h-7 w-7 border-2 border-gray-300 dark:border-gray-600 border-t-orange-500 dark:border-t-orange-500"></div>
        <p class="text-gray-900 dark:text-gray-100 font-medium">处理中...</p>
      </div>
    </div>
  </div>

  <script>
    // 夜间模式切换
    const themeToggle = document.getElementById('themeToggle')
    const htmlElement = document.documentElement

    // 检查本地存储中的主题设置
    const currentTheme = localStorage.getItem('theme') ||
                        (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')

    if (currentTheme === 'dark') {
      htmlElement.classList.add('dark')
    }

    themeToggle.addEventListener('click', () => {
      htmlElement.classList.toggle('dark')
      const theme = htmlElement.classList.contains('dark') ? 'dark' : 'light'
      localStorage.setItem('theme', theme)
    })
    const showError = (msg) => {
      const errorDiv = document.getElementById('errorMsg')
      errorDiv.querySelector('p').textContent = msg
      errorDiv.classList.remove('hidden')
      setTimeout(() => errorDiv.classList.add('hidden'), 5000)
    }

    const showLoading = (show) => {
      document.getElementById('loading').classList.toggle('hidden', !show)
    }

    // 生成授权链接
    document.getElementById('generateBtn').addEventListener('click', async () => {
      showLoading(true)
      try {
        const response = await fetch('/api/generate-auth', { method: 'POST' })
        const data = await response.json()

        if (!response.ok) {
          throw new Error(data.error || '生成失败')
        }

        document.getElementById('authUrl').href = data.authUrl
        document.getElementById('authUrl').textContent = data.authUrl
        document.getElementById('authUrlResult').classList.remove('hidden')

        // 显示步骤2
        setTimeout(() => {
          document.getElementById('step2').classList.remove('hidden')
        }, 300)
      } catch (error) {
        showError('生成授权链接失败: ' + error.message)
      } finally {
        showLoading(false)
      }
    })

    // 复制按钮通用函数
    const copyWithFeedback = (btn, text) => {
      navigator.clipboard.writeText(text).then(() => {
        const originalText = btn.textContent
        btn.textContent = '已复制'
        btn.classList.add('bg-green-500', 'text-white', 'border-green-500')
        btn.classList.remove('bg-white', 'hover:bg-gray-50', 'text-gray-800', 'border-gray-200')
        setTimeout(() => {
          btn.textContent = originalText
          btn.classList.remove('bg-green-500', 'text-white', 'border-green-500')
          btn.classList.add('bg-white', 'hover:bg-gray-50', 'text-gray-800', 'border-gray-200')
        }, 1500)
      })
    }

    // 复制授权链接
    document.getElementById('copyUrlBtn').addEventListener('click', function() {
      const url = document.getElementById('authUrl').textContent
      copyWithFeedback(this, url)
    })

    // 提取并交换 Token
    document.getElementById('extractBtn').addEventListener('click', async () => {
      const callbackUrl = document.getElementById('callbackUrl').value.trim()

      if (!callbackUrl) {
        showError('请输入回调 URL')
        return
      }

      showLoading(true)
      try {
        const response = await fetch('/api/exchange-token', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ callbackUrl })
        })
        const data = await response.json()

        if (!response.ok) {
          throw new Error(data.error || '交换失败')
        }

        document.getElementById('email').textContent = data.email || '(未获取)'
        document.getElementById('projectId').textContent = data.projectId || '(未获取, 该账户可能没有使用资格)'
        document.getElementById('refreshToken').textContent = data.refreshToken
        document.getElementById('tokenResult').classList.remove('hidden')
      } catch (error) {
        showError('Token 交换失败: ' + error.message)
      } finally {
        showLoading(false)
      }
    })

    // 复制 Refresh Token
    document.getElementById('copyRtBtn').addEventListener('click', function() {
      const rt = document.getElementById('refreshToken').textContent
      copyWithFeedback(this, rt)
    })
  </script>
</body>
</html>
  `);
});

/**
 * API: 生成授权链接
 */
app.post("/api/generate-auth", async (_req, res) => {
  try {
    const state = generateRandomState();
    const redirectUri = `http://localhost:${ANTIGRAVITY_CALLBACK_PORT}/oauth-callback`;

    const params = new URLSearchParams({
      access_type: "offline",
      client_id: ANTIGRAVITY_CLIENT_ID,
      prompt: "consent",
      redirect_uri: redirectUri,
      response_type: "code",
      scope: ANTIGRAVITY_SCOPES.join(" "),
      state: state,
    });

    const authUrl = `https://accounts.google.com/o/oauth2/v2/auth?${params.toString()}`;

    authConfig = {
      state,
      redirectUri,
    };

    res.json({ authUrl });
  } catch (error) {
    console.error("生成授权链接失败:", error);
    res.status(500).json({ error: error.message || "生成授权链接失败" });
  }
});

/**
 * API: 交换 Token
 */
app.post("/api/exchange-token", async (req, res) => {
  try {
    if (!authConfig) {
      return res.status(400).json({ error: "请先生成授权链接" });
    }

    const { callbackUrl } = req.body;

    if (!callbackUrl) {
      return res.status(400).json({ error: "缺少回调 URL" });
    }

    const url = new URL(callbackUrl);
    const code = url.searchParams.get("code");
    const returnedState = url.searchParams.get("state");

    if (!code) {
      return res.status(400).json({ error: "回调 URL 中未找到授权码" });
    }

    if (returnedState !== authConfig.state) {
      console.warn("State 验证失败，但继续尝试交换令牌...");
    }

    // 交换 Token
    const tokenResponse = await axios.post(
      "https://oauth2.googleapis.com/token",
      new URLSearchParams({
        code: code,
        client_id: ANTIGRAVITY_CLIENT_ID,
        client_secret: ANTIGRAVITY_CLIENT_SECRET,
        redirect_uri: authConfig.redirectUri,
        grant_type: "authorization_code",
      }).toString(),
      {
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
      }
    );

    const tokens = tokenResponse.data;
    const accessToken = tokens.access_token;
    const refreshToken = tokens.refresh_token;

    if (!refreshToken) {
      return res.status(500).json({ error: "未获取到 refresh_token" });
    }

    // 获取用户信息
    let email = "";
    try {
      const userInfoResponse = await axios.get(
        "https://www.googleapis.com/oauth2/v1/userinfo?alt=json",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
        }
      );
      email = userInfoResponse.data.email || "";
    } catch (e) {
      console.warn("获取用户信息失败:", e);
    }

    // 获取 Project ID
    let projectId = "";
    try {
      const loadResponse = await axios.post(
        "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist",
        {
          metadata: {
            ideType: "IDE_UNSPECIFIED",
            platform: "PLATFORM_UNSPECIFIED",
            pluginType: "GEMINI",
          },
        },
        {
          headers: {
            Authorization: `Bearer ${accessToken}`,
            "Content-Type": "application/json",
            "User-Agent": "google-api-nodejs-client/9.15.1",
            "X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
            "Client-Metadata": '{"ideType":"IDE_UNSPECIFIED","platform":"PLATFORM_UNSPECIFIED","pluginType":"GEMINI"}',
          },
        }
      );

      const loadData = loadResponse.data;
      if (typeof loadData.cloudaicompanionProject === "string") {
        projectId = loadData.cloudaicompanionProject;
      } else if (loadData.cloudaicompanionProject?.id) {
        projectId = loadData.cloudaicompanionProject.id;
      }
    } catch (e) {
      console.warn("获取 Project ID 失败");
    }

    res.json({
      email,
      projectId,
      refreshToken,
      accessToken,
    });
  } catch (error) {
    console.error("交换 Token 失败:", error);
    res.status(500).json({
      error: error.response?.data?.error_description || error.message || "交换 Token 失败",
    });
  }
});

/**
 * 启动服务器
 */
app.listen(PORT, () => {
  console.log(`\nAntigravity 授权服务已启动: http://localhost:${PORT}`);
  console.log(`\n请在浏览器中打开上述地址开始使用`);
});
