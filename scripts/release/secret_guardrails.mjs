import { execFileSync } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));

const REQUIRED_IGNORED_PATHS = [
  'app/android/key.properties',
  'app/android/app/upload-keystore.jks',
  'app/android/fastlane/report.xml',
  'app/android/fastlane/README.md',
  'app/android/play-service-account.json',
  'app/android/google-play-service-account.json',
  'app/build/app/outputs/bundle/release/app-release.aab',
  'app/build/app/outputs/flutter-apk/app-release.apk',
  'app/ios/AuthKey_ABCD1234.p8',
  'app/ios/fastlane/AuthKey_ABCD1234.p8',
  'app/ios/fastlane/app-store-connect-api-key.json',
  'app/ios/fastlane/app_store_connect_api_key.json',
  'app/ios/fastlane/report.xml',
  'app/ios/fastlane/README.md',
  'app/ios/Runner/Profile.mobileprovision',
  'app/build/ios/ipa/STONE SIGNATURE.ipa',
];

const FORBIDDEN_TRACKED_PATTERNS = [
  {
    code: 'release-secret-android-key-properties',
    pattern: /(^|\/)key\.properties$/i,
    message: 'Android signing key.properties must remain local or in CI secrets',
  },
  {
    code: 'release-secret-keystore',
    pattern: /\.(jks|keystore)$/i,
    message: 'Android keystore files must not be tracked',
  },
  {
    code: 'release-secret-google-play-json',
    pattern: /(^|\/).*((service[-_]account)|(play)).*\.json$/i,
    message: 'Google Play service account JSON must not be tracked',
  },
  {
    code: 'release-secret-apple-api-key',
    pattern: /(^|\/)(AuthKey[^/]*\.p8|.*api[-_]key.*\.json)$/i,
    message: 'App Store Connect API keys must not be tracked',
  },
  {
    code: 'release-secret-provisioning-profile',
    pattern: /\.(mobileprovision|provisionprofile)$/i,
    message: 'Apple provisioning profiles must not be tracked',
  },
  {
    code: 'release-secret-exported-binary',
    pattern: /\.(aab|apk|ipa)$/i,
    message: 'Exported release binaries must not be tracked',
  },
  {
    code: 'release-secret-fastlane-report',
    pattern: /(^|\/)fastlane\/(report\.xml|README\.md)$/i,
    message: 'Local fastlane reports and generated lane docs must not be tracked',
  },
];

/**
 * @typedef {Object} ReleaseSecretIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} ReleaseSecretReport
 * @property {boolean} ok
 * @property {ReleaseSecretIssue[]} issues
 * @property {string[]} ignored_paths
 * @property {string[]} tracked_files
 */

/**
 * @param {{ rootDir?: string, requiredIgnoredPaths?: string[] }} options
 * @returns {Promise<ReleaseSecretReport>}
 */
export async function validateReleaseSecretGuardrails({
  rootDir = REPO_ROOT,
  requiredIgnoredPaths = REQUIRED_IGNORED_PATHS,
} = {}) {
  const issues = [];
  const ignoredPaths = [];
  const trackedFiles = listTrackedFiles(rootDir);

  await requireIgnoreFiles(rootDir, issues);

  for (const relativePath of requiredIgnoredPaths) {
    if (isIgnored(rootDir, relativePath)) {
      ignoredPaths.push(relativePath);
      continue;
    }
    issues.push({
      code: 'release-secret-not-ignored',
      file: relativePath,
      key: null,
      message: 'release secret or local artifact path is not ignored by git',
    });
  }

  for (const relativePath of trackedFiles) {
    for (const rule of FORBIDDEN_TRACKED_PATTERNS) {
      if (!rule.pattern.test(relativePath)) {
        continue;
      }
      issues.push({
        code: rule.code,
        file: relativePath,
        key: null,
        message: rule.message,
      });
    }
  }

  return {
    ok: issues.length === 0,
    issues,
    ignored_paths: ignoredPaths.sort(),
    tracked_files: trackedFiles,
  };
}

async function requireIgnoreFiles(rootDir, issues) {
  for (const relativePath of ['.gitignore', 'app/android/.gitignore', 'app/ios/.gitignore']) {
    try {
      const contents = await readFile(join(rootDir, relativePath), 'utf8');
      if (contents.trim() === '') {
        issues.push({
          code: 'release-secret-empty-ignore-file',
          file: relativePath,
          key: null,
          message: 'ignore file must not be empty',
        });
      }
    } catch (error) {
      issues.push({
        code: 'release-secret-missing-ignore-file',
        file: relativePath,
        key: null,
        message: error?.code === 'ENOENT' ? 'required ignore file is missing' : error.message,
      });
    }
  }
}

function isIgnored(rootDir, relativePath) {
  try {
    execFileSync('git', ['check-ignore', '-q', '--', relativePath], {
      cwd: rootDir,
      stdio: 'ignore',
    });
    return true;
  } catch {
    return false;
  }
}

function listTrackedFiles(rootDir) {
  const output = execFileSync('git', ['ls-files', '-z'], {
    cwd: rootDir,
    encoding: 'utf8',
  });
  return output.split('\0').filter(Boolean).sort();
}

function formatIssue(issue) {
  const key = issue.key ? ` ${issue.key}` : '';
  return `- ${issue.file}${key}: ${issue.code}: ${issue.message}`;
}

function renderReport(report) {
  const lines = [
    '# Release secret guardrails',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Issues: ${report.issues.length}`,
    `Ignored paths checked: ${report.ignored_paths.length}`,
    `Tracked files checked: ${report.tracked_files.length}`,
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
  const report = await validateReleaseSecretGuardrails();
  process.stdout.write(renderReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
