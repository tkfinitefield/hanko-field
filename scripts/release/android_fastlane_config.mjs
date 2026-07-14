import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const ANDROID_FASTLANE_FILES = {
  gemfile: 'app/android/Gemfile',
  appfile: 'app/android/fastlane/Appfile',
  fastfile: 'app/android/fastlane/Fastfile',
};

/**
 * @typedef {Object} AndroidFastlaneIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} AndroidFastlaneReport
 * @property {boolean} ok
 * @property {AndroidFastlaneIssue[]} issues
 * @property {string[]} parsed_files
 */

/**
 * @param {{ rootDir?: string }} options
 * @returns {Promise<AndroidFastlaneReport>}
 */
export async function validateAndroidFastlaneConfig({ rootDir = REPO_ROOT } = {}) {
  const issues = [];
  const parsedFiles = [];

  const gemfile = await readText(rootDir, ANDROID_FASTLANE_FILES.gemfile, issues);
  if (gemfile !== null) {
    parsedFiles.push(ANDROID_FASTLANE_FILES.gemfile);
    requirePattern(gemfile, /^source "https:\/\/rubygems\.org"$/m, ANDROID_FASTLANE_FILES.gemfile, 'source', issues);
    requirePattern(gemfile, /^gem "fastlane", "2\.228\.0"$/m, ANDROID_FASTLANE_FILES.gemfile, 'fastlane', issues);
  }

  const appfile = await readText(rootDir, ANDROID_FASTLANE_FILES.appfile, issues);
  if (appfile !== null) {
    parsedFiles.push(ANDROID_FASTLANE_FILES.appfile);
    requirePattern(
      appfile,
      /^package_name\("org\.finitefield\.hankofield"\)$/m,
      ANDROID_FASTLANE_FILES.appfile,
      'package_name',
      issues,
    );
    rejectPattern(appfile, /json_key|key\.properties|keystore|\.jks/i, ANDROID_FASTLANE_FILES.appfile, 'secret', issues);
  }

  const fastfile = await readText(rootDir, ANDROID_FASTLANE_FILES.fastfile, issues);
  if (fastfile !== null) {
    parsedFiles.push(ANDROID_FASTLANE_FILES.fastfile);
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
      code: 'android-fastlane-missing-file',
      file: relativePath,
      key: null,
      message: error?.code === 'ENOENT' ? 'required Android fastlane file is missing' : error.message,
    });
    return null;
  }
}

function validateFastfile(fastfile, issues) {
  const file = ANDROID_FASTLANE_FILES.fastfile;
  requirePattern(fastfile, /^default_platform\(:android\)$/m, file, 'default_platform', issues);
  requirePattern(fastfile, /PACKAGE_NAME = "org\.finitefield\.hankofield"/, file, 'PACKAGE_NAME', issues);
  requirePattern(fastfile, /GOOGLE_PLAY_METADATA_PATH = File\.join\(REPO_ROOT, "release\/store_metadata\/google_play"\)/, file, 'GOOGLE_PLAY_METADATA_PATH', issues);
  requirePattern(fastfile, /SIGNOFF_CONFIRMATION_PHRASE = "I confirm the Stone Signature production release"/, file, 'SIGNOFF_CONFIRMATION_PHRASE', issues);
  requirePattern(fastfile, /def require_release_signoff\(platform\)[\s\S]*RELEASE_SIGNOFF_PATH[\s\S]*RELEASE_SIGNOFF_CONFIRMATION[\s\S]*SIGNOFF_CONFIRMATION_PHRASE[\s\S]*File\.file\?\(signoff_path\)[\s\S]*JSON\.parse\(File\.read\(signoff_path\)\)[\s\S]*manual_confirmation[\s\S]*required_phrase[\s\S]*approval[\s\S]*approved_for_m10_t07_execution[\s\S]*end/, file, 'require_release_signoff', issues);
  requirePattern(fastfile, /lane :metadata_check do[\s\S]*make google-play-metadata-check[\s\S]*end/, file, 'metadata_check', issues);
  requirePattern(fastfile, /lane :metadata do[\s\S]*upload_to_play_store\(/, file, 'metadata', issues);
  requirePattern(fastfile, /lane :internal do[\s\S]*flutter build appbundle --release[\s\S]*upload_to_play_store\(/, file, 'internal', issues);
  requirePattern(fastfile, /lane :production do[\s\S]*require_release_signoff\("android"\)[\s\S]*flutter build appbundle --release[\s\S]*upload_to_play_store\(/, file, 'production', issues);

  const metadataLane = extractLane(fastfile, 'metadata');
  if (metadataLane === null) {
    issues.push({
      code: 'android-fastlane-missing-lane',
      file,
      key: 'metadata',
      message: 'metadata lane is missing',
    });
  } else {
    requirePattern(metadataLane, /metadata_check/, file, 'metadata.metadata_check', issues);
    requirePattern(metadataLane, /json_key: required_env\("SUPPLY_JSON_KEY"\)/, file, 'metadata.json_key', issues);
    requirePattern(metadataLane, /metadata_path: GOOGLE_PLAY_METADATA_PATH/, file, 'metadata.metadata_path', issues);
    requirePattern(metadataLane, /skip_upload_apk: true/, file, 'metadata.skip_upload_apk', issues);
    requirePattern(metadataLane, /skip_upload_aab: true/, file, 'metadata.skip_upload_aab', issues);
    requirePattern(metadataLane, /skip_upload_images: true/, file, 'metadata.skip_upload_images', issues);
    requirePattern(metadataLane, /skip_upload_screenshots: true/, file, 'metadata.skip_upload_screenshots', issues);
    requirePattern(metadataLane, /validate_only: true/, file, 'metadata.validate_only', issues);
    rejectPattern(metadataLane, /^\s*(apk|aab):/m, file, 'metadata.binary_upload', issues);
  }

  const internalLane = extractLane(fastfile, 'internal');
  if (internalLane === null) {
    issues.push({
      code: 'android-fastlane-missing-lane',
      file,
      key: 'internal',
      message: 'internal lane is missing',
    });
  } else {
    requirePattern(internalLane, /track: "internal"/, file, 'internal.track', issues);
    requirePattern(internalLane, /aab: RELEASE_AAB_PATH/, file, 'internal.aab', issues);
    requirePattern(internalLane, /validate_only: ENV\.fetch\("SUPPLY_VALIDATE_ONLY", "true"\) != "false"/, file, 'internal.validate_only_default', issues);
  }

  const productionLane = extractLane(fastfile, 'production');
  if (productionLane === null) {
    issues.push({
      code: 'android-fastlane-missing-lane',
      file,
      key: 'production',
      message: 'production lane is missing',
    });
  } else {
    requirePattern(productionLane, /require_release_signoff\("android"\)/, file, 'production.signoff', issues);
    requirePattern(productionLane, /track: "production"/, file, 'production.track', issues);
    requirePattern(productionLane, /aab: RELEASE_AAB_PATH/, file, 'production.aab', issues);
    requirePattern(productionLane, /release_status: ENV\.fetch\("SUPPLY_PRODUCTION_RELEASE_STATUS", "draft"\)/, file, 'production.release_status_default', issues);
  }

  rejectPattern(fastfile, /BEGIN PRIVATE KEY|private_key|client_secret|upload-keystore|key\.properties/i, file, 'secret', issues);
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
    code: 'android-fastlane-required-config',
    file,
    key,
    message: 'required Android fastlane configuration is missing',
  });
}

function rejectPattern(contents, pattern, file, key, issues) {
  if (!pattern.test(contents)) {
    return;
  }
  issues.push({
    code: 'android-fastlane-forbidden-config',
    file,
    key,
    message: 'Android fastlane configuration includes forbidden release or secret material',
  });
}

function formatIssue(issue) {
  const key = issue.key ? ` ${issue.key}` : '';
  return `- ${issue.file}${key}: ${issue.code}: ${issue.message}`;
}

function renderReport(report) {
  const lines = [
    '# Android fastlane config check',
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
  const report = await validateAndroidFastlaneConfig();
  process.stdout.write(renderReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
