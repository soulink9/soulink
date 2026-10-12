// SPDX-License-Identifier: MIT
//
// launcher 纯逻辑测试：进程退出码映射与信号转发。
// 运行：node --test（从 cwd 自动发现；给显式文件路径亦可）
//
// 只测从 launcher 主体里抽出来的这几个函数——它们是 bin/soulink.js 中唯一带分支
// 的部分。spawn 的真实行为不在这里测：需要真起子进程，易 flaky，收益也低。

"use strict";

const test = require("node:test");
const assert = require("node:assert");
const { EventEmitter } = require("node:events");

const {
  launcherExitCode,
  forwardSignal,
  installSignalHandlers,
  // 必须带 .js 后缀：本机 make build 会产出裸二进制 bin/soulink，Node 的 require
  // 解析会优先命中那个无扩展名的文件，把二进制当 JS 读而抛 SyntaxError。
} = require("../bin/soulink.js");

// 造一个只记录被 kill 过哪些信号的假子进程
function fakeChild() {
  return {
    kills: [],
    kill(signal) {
      this.kills.push(signal);
    },
  };
}

test("launcherExitCode：正常退出直接用 status", () => {
  assert.strictEqual(launcherExitCode({ status: 0 }), 0);
  assert.strictEqual(launcherExitCode({ status: 3 }), 3);
});

test("launcherExitCode：被信号杀死不落回 0，按 128+signo 映射", () => {
  assert.strictEqual(launcherExitCode({ status: null, signal: "SIGINT" }), 130);
  assert.strictEqual(launcherExitCode({ status: null, signal: "SIGTERM" }), 143);
  assert.strictEqual(launcherExitCode({ status: null, signal: "SIGKILL" }), 137);
});

test("launcherExitCode：未知信号回落 1，不产生 128+undefined=NaN", () => {
  assert.strictEqual(launcherExitCode({ status: null, signal: "SIGWEIRD" }), 129);
});

test("launcherExitCode：spawn 失败（只有 error）返回 1", () => {
  assert.strictEqual(launcherExitCode({ error: new Error("ENOENT") }), 1);
});

test("forwardSignal：把信号原样转给子进程", () => {
  const child = fakeChild();
  forwardSignal(child, "SIGTERM")();
  assert.deepStrictEqual(child.kills, ["SIGTERM"]);
});

test("installSignalHandlers：非 win32 转发 SIGINT/SIGTERM，注销后不再转发", () => {
  const child = fakeChild();
  const target = new EventEmitter();
  const remove = installSignalHandlers(child, "linux", target);

  target.emit("SIGINT");
  target.emit("SIGTERM");
  assert.deepStrictEqual(child.kills, ["SIGINT", "SIGTERM"]);

  remove();
  target.emit("SIGTERM");
  assert.deepStrictEqual(
    child.kills,
    ["SIGINT", "SIGTERM"],
    "注销后不应再转发信号"
  );
});

test("installSignalHandlers：win32 不主动 kill 子进程（避免 TerminateProcess）", () => {
  const child = fakeChild();
  const target = new EventEmitter();
  const remove = installSignalHandlers(child, "win32", target);

  target.emit("SIGINT");
  assert.deepStrictEqual(child.kills, [], "win32 下不应主动 kill 子进程");

  remove();
});
