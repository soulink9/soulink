#!/usr/bin/env node
// SPDX-License-Identifier: MIT

"use strict";

const { spawn } = require("child_process");
const os = require("os");

const { resolveNativeBinary } = require("../scripts/platform");
// 安装说明链接取自 package.json 的 homepage：与包元数据共用一处真相，换域名时
// 不会漏掉错误提示里的这一份。
const { homepage } = require("../package.json");

// ── 可测试的纯逻辑 ────────────────────────────────────────────────────

// 子进程被信号杀死时 status 为 null，不能落回 0：否则 OOM 或任意信号退出
// 会被下游流水线读成成功。按惯例映射为 128+signo。
function launcherExitCode(result) {
  if (result.signal) {
    return 128 + (os.constants.signals[result.signal] ?? 1);
  }
  return result.status ?? (result.error ? 1 : 0);
}

function forwardSignal(child, signal) {
  return () => {
    child.kill(signal);
  };
}

function installSignalHandlers(child, platform, signalTarget) {
  const handlers = [];
  function add(signal, handler) {
    handlers.push([signal, handler]);
    signalTarget.on(signal, handler);
  }

  // Windows 把控制台 Ctrl+C 同时投递给两个进程。保留空 handler 以等待子进程自行
  // 结束；此处若调 child.kill("SIGINT")，Node 会映射成 TerminateProcess 强杀。
  // 已知代价：子进程收到 Ctrl+C 后若不退出，父进程会一直等下去。正常 Go 程序不会
  // 这样（信号由 runtime 转成 os.Interrupt，默认终止进程）。加超时强杀需要引入
  // 定时器与竞态处理，且区分不了「卡死」与「正在优雅收尾」，收益不抵复杂度。
  if (platform === "win32") {
    add("SIGINT", () => {});
  } else {
    for (const signal of ["SIGINT", "SIGTERM"]) {
      add(signal, forwardSignal(child, signal));
    }
  }

  return () => {
    for (const [signal, handler] of handlers) {
      signalTarget.removeListener(signal, handler);
    }
  };
}

module.exports = { launcherExitCode, forwardSignal, installSignalHandlers };

// ── CLI 入口 ──────────────────────────────────────────────────────────

function main() {
  const resolved = resolveNativeBinary();
  if (!resolved) {
    console.error(
      "[ERROR] 未找到 soulink 二进制。\n" +
        `        当前平台：${process.platform}/${process.arch}\n` +
        "        常见原因：安装时带了 --omit=optional（或 --no-optional），\n" +
        "        导致平台二进制包被跳过。请重新安装：\n" +
        "          npm install -g soulink\n" +
        `        其他安装方式见 ${homepage}`
    );
    process.exit(1);
  }

  const child = spawn(resolved, process.argv.slice(2), {
    stdio: "inherit",
    env: process.env,
  });

  const removeSignalHandlers = installSignalHandlers(
    child,
    process.platform,
    process
  );

  let settled = false;
  function settle(result) {
    if (settled) return;
    settled = true;
    removeSignalHandlers();
    process.exitCode = launcherExitCode(result);
  }

  // spawn 失败只经 "error" 事件暴露，不会触发 "close"。置 exitCode 并让 wrapper
  // 自然退出，使已写入 stderr 的诊断信息刷完。
  child.on("error", (err) => {
    console.error(`[ERROR] 启动 ${resolved} 失败：${err.message}`);
    settle({ error: err });
  });

  child.on("close", (code, signal) => {
    settle({ status: code, signal: signal });
  });
}

// 被 require（如测试）时只取上面的纯函数，不启动子进程
if (require.main === module) {
  main();
}
