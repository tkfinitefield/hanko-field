import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const IOS_FASTLANE_FILES = {
  gemfile: 'app/ios/Gemfile',
  appfile: 'app/ios/fastlane/Appfile',
  fastfile: 'app/ios/fastlane/Fastfile',
};

/**
 * @typedef {Object} IosFastlaneIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} IosFastlaneReport
 * @property {boolean} ok
 * @property {IosFastlaneIssue[]} issues
 * @property {string[]} parsed_files
 */

/**
 * @param {{ rootDir?: string }} options
 * @returns {Promise<IosFastlaneReport>}
 */
export async function validateIosFastlaneConfig({ rootDir = REPO_ROOT } = {}) {
  const issues = [];
  const parsedFiles = [];

  const gemfile = await readText(rootDir, IOS_FASTLANE_FILES.gemfile, issues);
  if (gemfile !== null) {
    parsedFiles.push(IOS_FASTLANE_FILES.gemfile);
    requirePattern(gemfile, /^source "https:\/\/rubygems\.org"$/m, IOS_FASTLANE_FILES.gemfile, 'source', issues);
    requirePattern(gemfile, /^gem "fastlane", "2\.228\.0"$/m, IOS_FASTLANE_FILES.gemfile, 'fastlane', issues);
  }

  const appfile = await readText(rootDir, IOS_FASTLANE_FILES.appfile, issues);
  if (appfile !== null) {
    parsedFiles.push(IOS_FASTLANE_FILES.appfile);
    requirePattern(
      appfile,
      /^app_identifier\("org\.finitefield\.hankofield"\)$/m,
      IOS_FASTLANE_FILES.appfile,
      'app_identifier',
      issues,
    );
    rejectPattern(appfile, /api_key|issuer_id|key_id|private_key|\.p8/i, IOS_FASTLANE_FILES.appfile, 'secret', issues);
  }

  const fastfile = await readText(rootDir, IOS_FASTLANE_FILES.fastfile, issues);
  if (fastfile !== null) {
    parsedFiles.push(IOS_FASTLANE_FILES.fastfile);
    validateFastfile(fastfile, issues);
  }

  return {
    ok: issues.length === 0,
    issues,
    parsed_files: parsedFiles.sort(),
  };
}

async function readText(rootDir, relativePath, issues) {
  try {
    return await readFile(join(rootDir, relativePath), 'utf8');
  } catch (error) {
    issues.push({
      code: 'ios-fastlane-missing-file',
      file: relativePath,
      key: null,
      message: error?.code === 'ENOENT' ? 'required iOS fastlane file is missing' : error.message,
    });
    return null;
  }
}

function validateFastfile(fastfile, issues) {
  const file = IOS_FASTLANE_FILES.fastfile;
  requirePattern(fastfile, /^default_platform\(:ios\)$/m, file, 'default_platform', issues);
  requirePattern(fastfile, /APP_IDENTIFIER = "org\.finitefield\.hankofield"/, file, 'APP_IDENTIFIER', issues);
  requirePattern(fastfile, /APP_STORE_METADATA_PATH = File\.join\(REPO_ROOT, "release\/store_metadata\/app_store"\)/, file, 'APP_STORE_METADATA_PATH', issues);
  requirePattern(fastfile, /lane :metadata_check do[\s\S]*make app-store-metadata-check[\s\S]*end/, file, 'metadata_check', issues);
  requirePattern(fastfile, /lane :metadata do[\s\S]*deliver\(/, file, 'metadata', issues);
  requirePattern(fastfile, /lane :testflight_upload do[\s\S]*flutter build ipa --release[\s\S]*upload_to_testflight\(/, file, 'testflight_upload', issues);

  const metadataLane = extractLane(fastfile, 'metadata');
  if (metadataLane === null) {
    issues.push({
      code: 'ios-fastlane-missing-lane',
      file,
      key: 'metadata',
      message: 'metadata lane is missing',
    });
  } else {
    requirePattern(metadataLane, /metadata_check/, file, 'metadata.metadata_check', issues);
    requirePattern(metadataLane, /api_key_path: required_env\("APP_STORE_CONNECT_API_KEY_PATH"\)/, file, 'metadata.api_key_path', issues);
    requirePattern(metadataLane, /metadata_path: APP_STORE_METADATA_PATH/, file, 'metadata.metadata_path', issues);
    requirePattern(metadataLane, /skip_binary_upload: true/, file, 'metadata.skip_binary_upload', issues);
    requirePattern(metadataLane, /skip_screenshots: true/, file, 'metadata.skip_screenshots', issues);
    requirePattern(metadataLane, /submit_for_review: false/, file, 'metadata.submit_for_review', issues);
    rejectPattern(metadataLane, /^\s*(ipa|pkg):/m, file, 'metadata.binary_upload', issues);
    rejectPattern(metadataLane, /upload_to_testflight|pilot\(/, file, 'metadata.testflight_upload', issues);
  }

  const testflightLane = extractLane(fastfile, 'testflight_upload');
  if (testflightLane === null) {
    issues.push({
      code: 'ios-fastlane-missing-lane',
      file,
      key: 'testflight_upload',
      message: 'testflight_upload lane is missing',
    });
  } else {
    requirePattern(testflightLane, /api_key_path: required_env\("APP_STORE_CONNECT_API_KEY_PATH"\)/, file, 'testflight_upload.api_key_path', issues);
    requirePattern(testflightLane, /ipa: latest_ipa_path/, file, 'testflight_upload.ipa', issues);
    requirePattern(testflightLane, /skip_submission: ENV\.fetch\("TESTFLIGHT_SKIP_SUBMISSION", "true"\) != "false"/, file, 'testflight_upload.skip_submission_default', issues);
  }

  rejectPattern(fastfile, /BEGIN PRIVATE KEY|issuer_id|key_id|private_key|\.p8/i, file, 'secret', issues);
}

function extractLane(fastfile, laneName) {
  const startPattern = new RegExp(`^\\s*lane :${laneName} do\\s*$`, 'm');
  const startMatch = startPattern.exec(fastfile);
  if (!startMatch) {
    return null;
  }
  const start = startMatch.index;
  const lines = fastfile.slice(start).split('\n');
  let depth = 0;
  const laneLines = [];
  for (const line of lines) {
    laneLines.push(line);
    if (/\bdo\b/.test(line)) {
      depth += 1;
    }
    if (/^\s*end\s*$/.test(line)) {
      depth -= 1;
      if (depth === 0) {
        break;
      }
    }
  }
  return laneLines.join('\n');
}

function requirePattern(contents, pattern, file, key, issues) {
  if (pattern.test(contents)) {
    return;
  }
  issues.push({
    code: 'ios-fastlane-required-config',
    file,
    key,
    message: 'required iOS fastlane configuration is missing',
  });
}

function rejectPattern(contents, pattern, file, key, issues) {
  if (!pattern.test(contents)) {
    return;
  }
  issues.push({
    code: 'ios-fastlane-forbidden-config',
    file,
    key,
    message: 'iOS fastlane configuration includes forbidden release or secret material',
  });
}

function formatIssue(issue) {
  const key = issue.key ? ` ${issue.key}` : '';
  return `- ${issue.file}${key}: ${issue.code}: ${issue.message}`;
}

function renderReport(report) {
  const lines = [
    '# iOS fastlane config check',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Issues: ${report.issues.length}`,
    `Parsed files: ${report.parsed_files.length}`,
  ];
  if (report.issues.length > 0) {
    lines.push('', '## Issues', '');
    for (const issue of report.issues) {
      lines.push(formatIssue(issue));
    }
  }
  return `${lines.join('\n')}\n`;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const report = await validateIosFastlaneConfig();
  process.stdout.write(renderReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
