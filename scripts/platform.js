// SPDX-License-Identifier: MIT

"use strict";

const path = require("path");
const fs = require("fs");

const IS_WINDOWS = process.platform === "win32";
const BINARY_FILENAME = IS_WINDOWS ? "soulink.exe" : "soulink";

// process.platform-process.arch（Node 命名）→ 平台包名。
// 左侧是 Node 的 arch 命名（x64/arm64），平台包内的二进制仍用 Go 命名。
// 本表是平台归属的唯一真相，须与根 package.json 的 optionalDependencies 保持一致。
const PLATFORM_PKG = {
  "darwin-arm64": "@soulink9/soulink-darwin-arm64",
  "darwin-x64": "@soulink9/soulink-darwin-x64",
  "linux-arm64": "@soulink9/soulink-linux-arm64",
  "linux-x64": "@soulink9/soulink-linux-x64",
  "win32-arm64": "@soulink9/soulink-win32-arm64",
  "win32-x64": "@soulink9/soulink-win32-x64",
};

// 返回当前平台原生二进制的绝对路径；找不到返回 null。
// 先取随主包安装的平台包（正式安装路径），再兜底本机构建产物
// （make build → bin/soulink），便于从源码目录直接运行 launcher。
function resolveNativeBinary() {
  const pkgName = PLATFORM_PKG[`${process.platform}-${process.arch}`];
  if (pkgName) {
    try {
      const pkgDir = path.dirname(require.resolve(`${pkgName}/package.json`));
      const binPath = path.join(pkgDir, "bin", BINARY_FILENAME);
      if (fs.existsSync(binPath)) {
        return binPath;
      }
    } catch (err) {
      // 平台包未安装（如安装时带了 --omit=optional）属预期情况，静默回退；
      // 其它解析错误值得告警，否则会掩盖安装异常。
      if (err.code !== "MODULE_NOT_FOUND") {
        console.warn(`[WARN] 解析平台包 ${pkgName} 出错：${err.message}`);
      }
    }
  }

  // 兜底本机产物。Windows 上 `go build -o bin/soulink` 不会追加 .exe，但手工构建
  // 或经 Makefile 的 windows 目标时可能是 .exe——两个名字都试一遍。
  const localNames = IS_WINDOWS
    ? [BINARY_FILENAME, "soulink"]
    : [BINARY_FILENAME];
  for (const name of localNames) {
    const localPath = path.join(__dirname, "..", "bin", name);
    if (fs.existsSync(localPath)) {
      return localPath;
    }
  }
  return null;
}

module.exports = { resolveNativeBinary };
