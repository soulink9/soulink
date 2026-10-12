// SPDX-License-Identifier: MIT

"use strict";

const path = require("path");
const fs = require("fs");

const IS_WINDOWS = process.platform === "win32";
const BINARY_FILENAME = IS_WINDOWS ? "soulink.exe" : "soulink";

// process.platform-process.arch（Node 命名）→ 平台包名。
// 注意 x64/arm64 是 Node 的 arch 命名，平台包内的二进制仍用 Go 命名（amd64/arm64）。
const PLATFORM_PKG = {
  "darwin-arm64": "@soulink9/soulink-darwin-arm64",
  "darwin-x64": "@soulink9/soulink-darwin-x64",
  "linux-arm64": "@soulink9/soulink-linux-arm64",
  "linux-x64": "@soulink9/soulink-linux-x64",
  "win32-arm64": "@soulink9/soulink-win32-arm64",
  "win32-x64": "@soulink9/soulink-win32-x64",
};

// 以 package.json 的 optionalDependencies 为单一真相反查平台包名，
// 避免平台包清单在 package.json 与本文件两处维护。
function getPlatformPackageName() {
  const key = `${process.platform}-${process.arch}`;

  try {
    const parentPkg = JSON.parse(
      fs.readFileSync(path.join(__dirname, "..", "package.json"), "utf8")
    );
    for (const name of Object.keys(parentPkg.optionalDependencies || {})) {
      if (name.endsWith(`-${key}`)) {
        return name;
      }
    }
  } catch (_) {}

  return PLATFORM_PKG[key] || null;
}

function resolveNativeBinary() {
  const pkgName = getPlatformPackageName();
  if (pkgName) {
    try {
      const pkgDir = path.dirname(require.resolve(`${pkgName}/package.json`));
      const binPath = path.join(pkgDir, "bin", BINARY_FILENAME);
      if (fs.existsSync(binPath)) {
        return { path: binPath, fromPlatformPkg: true };
      }
    } catch (err) {
      if (err.code !== "MODULE_NOT_FOUND") {
        console.warn(`[WARN] 解析平台包 ${pkgName} 出错：${err.message}`);
      }
    }
  }

  // 兜底：本机构建产物（make build → bin/soulink），便于从源码目录直接运行 launcher
  const localPath = path.join(__dirname, "..", "bin", BINARY_FILENAME);
  if (fs.existsSync(localPath)) {
    return { path: localPath, fromPlatformPkg: false };
  }

  return null;
}

module.exports = {
  IS_WINDOWS,
  BINARY_FILENAME,
  PLATFORM_PKG,
  getPlatformPackageName,
  resolveNativeBinary,
};
