#!/usr/bin/env node

/**
 * postinstall script for larasense-limbo npm package.
 *
 * Downloads the correct pre-built Go binary from GitHub Releases
 * based on the current OS and architecture.
 */

"use strict";

const os = require("os");
const fs = require("fs");
const path = require("path");
const https = require("https");
const { execSync } = require("child_process");

const REPO = "Mattel-Limbo/larasense-limbo";
const VERSION = require("../package.json").version;

// GoReleaser naming: larasense-limbo_<version>_<os>_<arch>.<ext>
const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

function getBinaryName() {
  return os.platform() === "win32" ? "larasense-limbo.exe" : "larasense-limbo";
}

function getArchiveInfo() {
  const platform = PLATFORM_MAP[os.platform()];
  const arch = ARCH_MAP[os.arch()];

  if (!platform || !arch) {
    console.error(
      `Unsupported platform: ${os.platform()}-${os.arch()}\n` +
        "Supported: darwin/linux/win32 on x64/arm64"
    );
    process.exit(1);
  }

  const isWindows = os.platform() === "win32";
  const ext = isWindows ? "zip" : "tar.gz";
  const filename = `larasense-limbo_${VERSION}_${platform}_${arch}.${ext}`;
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${filename}`;

  return { url, filename, ext, isWindows };
}

function getBinDir() {
  return path.join(__dirname, "..", "bin");
}

/**
 * Follow redirects (GitHub releases redirect to S3/CDN).
 * Returns a Promise that resolves with the response stream.
 */
function download(url, maxRedirects = 5) {
  return new Promise((resolve, reject) => {
    if (maxRedirects <= 0) {
      return reject(new Error("Too many redirects"));
    }

    const proto = url.startsWith("https") ? https : require("http");
    proto
      .get(url, { headers: { "User-Agent": "larasense-limbo-npm" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return resolve(download(res.headers.location, maxRedirects - 1));
        }
        if (res.statusCode !== 200) {
          return reject(
            new Error(`Download failed: HTTP ${res.statusCode} for ${url}`)
          );
        }
        resolve(res);
      })
      .on("error", reject);
  });
}

/**
 * Download file to a temporary path.
 */
async function downloadToFile(url, destPath) {
  const stream = await download(url);
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(destPath);
    stream.pipe(file);
    file.on("finish", () => {
      file.close(resolve);
    });
    file.on("error", (err) => {
      fs.unlinkSync(destPath);
      reject(err);
    });
  });
}

/**
 * Extract archive and move binary to bin/.
 */
function extractBinary(archivePath, ext, isWindows, binDir) {
  const binaryName = getBinaryName();
  const binaryDest = path.join(binDir, binaryName);

  if (isWindows) {
    // Use PowerShell to extract zip
    const tempDir = path.join(os.tmpdir(), "larasense-limbo-extract");
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
    fs.mkdirSync(tempDir, { recursive: true });

    execSync(
      `powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${tempDir}' -Force"`,
      { stdio: "pipe" }
    );

    // Find the binary in extracted files
    const extracted = findBinary(tempDir, binaryName);
    if (!extracted) {
      throw new Error(`Binary ${binaryName} not found in archive`);
    }
    fs.copyFileSync(extracted, binaryDest);
    fs.rmSync(tempDir, { recursive: true });
  } else {
    // Use tar to extract
    const tempDir = path.join(os.tmpdir(), "larasense-limbo-extract");
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
    fs.mkdirSync(tempDir, { recursive: true });

    execSync(`tar -xzf "${archivePath}" -C "${tempDir}"`, { stdio: "pipe" });

    const extracted = findBinary(tempDir, binaryName);
    if (!extracted) {
      throw new Error(`Binary ${binaryName} not found in archive`);
    }
    fs.copyFileSync(extracted, binaryDest);
    fs.chmodSync(binaryDest, 0o755);
    fs.rmSync(tempDir, { recursive: true });
  }

  return binaryDest;
}

/**
 * Recursively find a binary file in a directory.
 */
function findBinary(dir, name) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      const found = findBinary(fullPath, name);
      if (found) return found;
    } else if (entry.name === name) {
      return fullPath;
    }
  }
  return null;
}

/**
 * Verify the binary works.
 */
function verifyBinary(binaryPath) {
  try {
    const output = execSync(`"${binaryPath}" version`, {
      encoding: "utf-8",
      timeout: 10000,
      stdio: ["pipe", "pipe", "pipe"],
    });
    return output.trim();
  } catch {
    return null;
  }
}

async function main() {
  const binDir = getBinDir();
  const binaryName = getBinaryName();
  const binaryPath = path.join(binDir, binaryName);

  // Skip if binary already exists and works (e.g. re-install)
  if (fs.existsSync(binaryPath)) {
    const version = verifyBinary(binaryPath);
    if (version) {
      console.log(`larasense-limbo already installed: ${version}`);
      return;
    }
  }

  const { url, filename, ext, isWindows } = getArchiveInfo();
  const tempArchive = path.join(os.tmpdir(), filename);

  console.log(`Downloading larasense-limbo v${VERSION}...`);
  console.log(`  Platform: ${os.platform()}-${os.arch()}`);
  console.log(`  From: ${url}`);

  try {
    await downloadToFile(url, tempArchive);
  } catch (err) {
    console.error(`\nFailed to download larasense-limbo v${VERSION}:`);
    console.error(`  ${err.message}`);
    console.error(`\nYou can install manually from:`);
    console.error(`  https://github.com/${REPO}/releases/tag/v${VERSION}`);
    process.exit(1);
  }

  console.log("Extracting...");

  try {
    // Ensure bin directory exists
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }

    const installedPath = extractBinary(tempArchive, ext, isWindows, binDir);

    // Clean up archive
    fs.unlinkSync(tempArchive);

    // Verify
    const version = verifyBinary(installedPath);
    if (version) {
      console.log(`\n  larasense-limbo installed successfully!`);
      console.log(`  ${version}\n`);
    } else {
      console.log(`\n  larasense-limbo binary installed to: ${installedPath}`);
      console.log(`  (could not verify version — binary may need execute permission)\n`);
    }
  } catch (err) {
    console.error(`\nFailed to extract larasense-limbo:`);
    console.error(`  ${err.message}`);
    console.error(`\nYou can install manually from:`);
    console.error(`  https://github.com/${REPO}/releases/tag/v${VERSION}`);

    // Clean up
    if (fs.existsSync(tempArchive)) {
      fs.unlinkSync(tempArchive);
    }
    process.exit(1);
  }
}

main();
