import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { validateReleaseSecretGuardrails } from './secret_guardrails.mjs';

test('accepts the checked-in release secret guardrails', async () => {
  const report = await validateReleaseSecretGuardrails();

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.ok(report.ignored_paths.includes('app/android/key.properties'));
  assert.ok(report.ignored_paths.includes('app/ios/AuthKey_ABCD1234.p8'));
});

test('fails when a required private release path is not ignored', async () => {
  const rootDir = await createTempRepo({ rootGitignore: '# empty release rules\n' });

  const report = await validateReleaseSecretGuardrails({
    rootDir,
    requiredIgnoredPaths: ['app/ios/AuthKey_ABCD1234.p8'],
  });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'release-secret-not-ignored' &&
        issue.file === 'app/ios/AuthKey_ABCD1234.p8',
    ),
  );
});

test('fails when forbidden release material is already tracked', async () => {
  const rootDir = await createTempRepo();
  await mkdir(join(rootDir, 'secrets'), { recursive: true });
  await writeFile(join(rootDir, 'secrets/AuthKey_ABCD1234.p8'), 'private key fixture\n');
  execFileSync('git', ['add', '-f', 'secrets/AuthKey_ABCD1234.p8'], { cwd: rootDir });

  const report = await validateReleaseSecretGuardrails({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'release-secret-apple-api-key' &&
        issue.file === 'secrets/AuthKey_ABCD1234.p8',
    ),
  );
});

async function createTempRepo({
  rootGitignore = releaseIgnoreRules(),
  androidGitignore = 'key.properties\n**/*.jks\n**/*.keystore\n',
  iosGitignore = 'Flutter/ephemeral/\n',
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-release-secrets-'));
  execFileSync('git', ['init', '-q'], { cwd: rootDir });
  await mkdir(join(rootDir, 'app/android'), { recursive: true });
  await mkdir(join(rootDir, 'app/ios'), { recursive: true });
  await writeFile(join(rootDir, '.gitignore'), rootGitignore);
  await writeFile(join(rootDir, 'app/android/.gitignore'), androidGitignore);
  await writeFile(join(rootDir, 'app/ios/.gitignore'), iosGitignore);
  await writeFile(join(rootDir, 'README.md'), 'fixture\n');
  execFileSync('git', ['add', 'README.md', '.gitignore', 'app/android/.gitignore', 'app/ios/.gitignore'], {
    cwd: rootDir,
  });
  return rootDir;
}

function releaseIgnoreRules() {
  return `app/android/key.properties
app/android/**/*.jks
app/android/**/*.keystore
app/android/fastlane/report.xml
app/android/fastlane/README.md
app/android/*service-account*.json
app/android/*service_account*.json
app/android/*play*.json
app/ios/fastlane/report.xml
app/ios/fastlane/README.md
app/ios/AuthKey*.p8
app/ios/fastlane/AuthKey*.p8
app/ios/fastlane/*api-key*.json
app/ios/fastlane/*api_key*.json
app/ios/**/*.mobileprovision
app/ios/**/*.provisionprofile
app/build/app/outputs/**/*.aab
app/build/app/outputs/**/*.apk
app/build/ios/**/*.ipa
`;
}
