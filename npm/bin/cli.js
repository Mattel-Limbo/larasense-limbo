#!/usr/bin/env node

/**
 * CLI wrapper for larasense-limbo.
 *
 * This script finds and executes the platform-specific binary
 * that was downloaded during `npm install` (postinstall).
 */

"use strict";

const os = require("os");
const path = require("path");
const { execFileSync } = require("child_process");
const fs = require("fs");

const REPO = "Mattel-Limbo/larasense-limbo";

function getBinaryName() {
  return os.platform() === "win32" ? "larasense-limbo.exe" : "larasense-limbo";
}

function getBinaryPath() {
  const binaryName = getBinaryName();

  // 1. Check in the same bin/ directory (installed by postinstall)
  const localBin = path.join(__dirname, binaryName);
  if (fs.existsSync(localBin)) {
    return localBin;
  }

  // 2. Check if available in PATH (installed via go install or manual)
  try {
    const which = os.platform() === "win32" ? "where" : "which";
    const result = require("child_process")
      .execSync(`${which} larasense-limbo`, { encoding: "utf-8", stdio: ["pipe", "pipe", "pipe"] })
      .trim()
      .split("\n")[0]
      .trim();
    if (result && fs.existsSync(result)) {
      return result;
    }
  } catch {
    // Not in PATH
  }

  return null;
}

function main() {
  const binaryPath = getBinaryPath();

  if (!binaryPath) {
    console.error("Error: larasense-limbo binary not found.");
    console.error("");
    console.error("The binary should have been downloaded during installation.");
    console.error("Try reinstalling:");
    console.error("  npm install -g larasense-limbo");
    console.error("");
    console.error("Or install manually from:");
    console.error(`  https://github.com/${REPO}/releases`);
    process.exit(1);
  }

  // Forward all arguments to the Go binary
  const args = process.argv.slice(2);

  try {
    execFileSync(binaryPath, args, {
      stdio: "inherit",
      env: process.env,
    });
  } catch (err) {
    // execFileSync throws on non-zero exit code — forward it
    if (err.status !== null && err.status !== undefined) {
      process.exit(err.status);
    }
    // Actual execution error
    console.error(`Failed to execute larasense-limbo: ${err.message}`);
    process.exit(1);
  }
}

main();
