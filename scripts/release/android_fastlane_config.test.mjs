import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { validateAndroidFastlaneConfig } from './android_fastlane_config.mjs';

test('accepts the checked-in Android fastlane configuration shape', async () => {
  const report = await validateAndroidFastlaneConfig();

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.deepEqual(report.parsed_files, [
    'app/android/Gemfile',
    'app/android/fastlane/Appfile',
    'app/android/fastlane/Fastfile',
  ]);
});

test('rejects a metadata lane that could upload an AAB', async () => {
  const rootDir = await createTempRoot();
  const fastfilePath = join(rootDir, 'app/android/fastlane/Fastfile');
  const fastfile = await readFile(fastfilePath, 'utf8');
  await writeFile(fastfilePath, fastfile.replace('skip_upload_aab: true', 'skip_upload_aab: false'));

  const report = await validateAndroidFastlaneConfig({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'android-fastlane-required-config' &&
        issue.key === 'metadata.skip_upload_aab',
    ),
  );
});

test('rejects committed Google Play credential material in Appfile', async () => {
  const rootDir = await createTempRoot();
  await writeFile(
    join(rootDir, 'app/android/fastlane/Appfile'),
    'package_name("org.finitefield.hankofield")\njson_key_file("play-service-account.json")\n',
  );

  const report = await validateAndroidFastlaneConfig({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'android-fastlane-forbidden-config' &&
        issue.file === 'app/android/fastlane/Appfile',
    ),
  );
});

test('rejects a production lane without manual signoff', async () => {
  const rootDir = await createTempRoot();
  const fastfilePath = join(rootDir, 'app/android/fastlane/Fastfile');
  const fastfile = await readFile(fastfilePath, 'utf8');
  await writeFile(fastfilePath, fastfile.replace('    require_release_signoff("android")\n', ''));

  const report = await validateAndroidFastlaneConfig({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'android-fastlane-required-config' &&
        issue.key === 'production.signoff',
    ),
  );
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-android-fastlane-'));
  await mkdir(join(rootDir, 'app/android/fastlane'), { recursive: true });
  await writeFile(
    join(rootDir, 'app/android/Gemfile'),
    'source "https://rubygems.org"\n\ngem "fastlane", "2.228.0"\n',
  );
  await writeFile(
    join(rootDir, 'app/android/fastlane/Appfile'),
    'package_name("org.finitefield.hankofield")\n',
  );
  await writeFile(join(rootDir, 'app/android/fastlane/Fastfile'), validFastfile());
  return rootDir;
}

function validFastfile() {
  return `require "json"
require "shellwords"

default_platform(:android)

REPO_ROOT = File.expand_path("../../..", __dir__)
APP_ROOT = File.join(REPO_ROOT, "app")
GOOGLE_PLAY_METADATA_PATH = File.join(REPO_ROOT, "release/store_metadata/google_play")
RELEASE_AAB_PATH = File.join(APP_ROOT, "build/app/outputs/bundle/release/app-release.aab")
PACKAGE_NAME = "org.finitefield.hankofield"
SIGNOFF_CONFIRMATION_PHRASE = "I confirm the Stone Signature production release"

def required_env(name)
  value = ENV[name].to_s.strip
  UI.user_error!("\#{name} is required for this lane") if value.empty?
  value
end

def repo_sh(command)
  sh("cd \#{Shellwords.escape(REPO_ROOT)} && \#{command}")
end

def require_release_signoff(platform)
  signoff_path = required_env("RELEASE_SIGNOFF_PATH")
  confirmation = required_env("RELEASE_SIGNOFF_CONFIRMATION")
  unless confirmation == SIGNOFF_CONFIRMATION_PHRASE
    UI.user_error!("RELEASE_SIGNOFF_CONFIRMATION must equal \#{SIGNOFF_CONFIRMATION_PHRASE.inspect}")
  end
  unless File.file?(signoff_path)
    UI.user_error!("RELEASE_SIGNOFF_PATH does not exist: \#{signoff_path}")
  end
  signoff = JSON.parse(File.read(signoff_path))
  unless signoff.dig("manual_confirmation", "required_phrase") == SIGNOFF_CONFIRMATION_PHRASE
    UI.user_error!("RELEASE_SIGNOFF_PATH does not record the required confirmation phrase")
  end
  unless signoff.dig("approval", "approved_for_m10_t07_execution") == true
    UI.user_error!("RELEASE_SIGNOFF_PATH is not approved for M10-T07 execution")
  end
  UI.success("Production release signoff recorded for \#{platform}: \#{signoff_path}")
  signoff_path
end

platform :android do
  desc "Validate generated Google Play metadata locally without Google Play credentials."
  lane :metadata_check do
    repo_sh("make google-play-metadata-check")
    UI.success("Generated Google Play metadata is in sync.")
  end

  desc "Validate Google Play metadata through supply without uploading APKs or AABs."
  lane :metadata do
    metadata_check
    upload_to_play_store(
      package_name: PACKAGE_NAME,
      json_key: required_env("SUPPLY_JSON_KEY"),
      metadata_path: GOOGLE_PLAY_METADATA_PATH,
      skip_upload_apk: true,
      skip_upload_aab: true,
      skip_upload_images: true,
      skip_upload_screenshots: true,
      validate_only: true
    )
  end

  desc "Build the release AAB and upload it to the Google Play internal track."
  lane :internal do
    sh("cd \#{Shellwords.escape(APP_ROOT)} && flutter build appbundle --release")
    upload_to_play_store(
      package_name: PACKAGE_NAME,
      json_key: required_env("SUPPLY_JSON_KEY"),
      track: "internal",
      aab: RELEASE_AAB_PATH,
      metadata_path: GOOGLE_PLAY_METADATA_PATH,
      skip_upload_images: true,
      skip_upload_screenshots: true,
      validate_only: ENV.fetch("SUPPLY_VALIDATE_ONLY", "true") != "false"
    )
  end

  desc "Build the release AAB and upload it to the Google Play production track after manual signoff."
  lane :production do
    require_release_signoff("android")
    sh("cd \#{Shellwords.escape(APP_ROOT)} && flutter build appbundle --release")
    upload_to_play_store(
      package_name: PACKAGE_NAME,
      json_key: required_env("SUPPLY_JSON_KEY"),
      track: "production",
      aab: RELEASE_AAB_PATH,
      metadata_path: GOOGLE_PLAY_METADATA_PATH,
      skip_upload_images: true,
      skip_upload_screenshots: true,
      release_status: ENV.fetch("SUPPLY_PRODUCTION_RELEASE_STATUS", "draft")
    )
  end
end
`;
}
