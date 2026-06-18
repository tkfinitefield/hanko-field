import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { validateIosFastlaneConfig } from './ios_fastlane_config.mjs';

test('accepts the checked-in iOS fastlane configuration shape', async () => {
  const report = await validateIosFastlaneConfig();

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.deepEqual(report.parsed_files, [
    'app/ios/ExportOptions.plist',
    'app/ios/Gemfile',
    'app/ios/fastlane/Appfile',
    'app/ios/fastlane/Fastfile',
  ]);
});

test('rejects a metadata lane that could upload an IPA', async () => {
  const rootDir = await createTempRoot();
  const fastfilePath = join(rootDir, 'app/ios/fastlane/Fastfile');
  const fastfile = await readFile(fastfilePath, 'utf8');
  await writeFile(fastfilePath, fastfile.replace('skip_binary_upload: true', 'skip_binary_upload: false'));

  const report = await validateIosFastlaneConfig({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'ios-fastlane-required-config' &&
        issue.key === 'metadata.skip_binary_upload',
    ),
  );
});

test('rejects committed App Store Connect credential material in Appfile', async () => {
  const rootDir = await createTempRoot();
  await writeFile(
    join(rootDir, 'app/ios/fastlane/Appfile'),
    'app_identifier("org.finitefield.hankofield")\napi_key_path("AuthKey_ABCD1234.p8")\n',
  );

  const report = await validateIosFastlaneConfig({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'ios-fastlane-forbidden-config' &&
        issue.file === 'app/ios/fastlane/Appfile',
    ),
  );
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-ios-fastlane-'));
  await mkdir(join(rootDir, 'app/ios/fastlane'), { recursive: true });
  await writeFile(join(rootDir, 'app/ios/ExportOptions.plist'), validExportOptions());
  await writeFile(
    join(rootDir, 'app/ios/Gemfile'),
    'source "https://rubygems.org"\n\ngem "fastlane", "2.228.0"\n',
  );
  await writeFile(
    join(rootDir, 'app/ios/fastlane/Appfile'),
    'app_identifier("org.finitefield.hankofield")\n',
  );
  await writeFile(join(rootDir, 'app/ios/fastlane/Fastfile'), validFastfile());
  return rootDir;
}

function validFastfile() {
  return `require "shellwords"

ENV["FASTLANE_SKIP_UPDATE_CHECK"] = "1"

opt_out_usage

default_platform(:ios)

REPO_ROOT = File.expand_path("../../..", __dir__)
APP_ROOT = File.join(REPO_ROOT, "app")
APP_STORE_METADATA_PATH = File.join(REPO_ROOT, "release/store_metadata/app_store")
IOS_EXPORT_OPTIONS_PATH = File.join(APP_ROOT, "ios/ExportOptions.plist")
APP_IDENTIFIER = "org.finitefield.hankofield"

def required_env(name)
  value = ENV[name].to_s.strip
  UI.user_error!("\#{name} is required for this lane") if value.empty?
  value
end

def repo_sh(command)
  sh("cd \#{Shellwords.escape(REPO_ROOT)} && \#{command}")
end

def latest_ipa_path
  ipa = Dir[File.join(APP_ROOT, "build/ios/ipa/*.ipa")].max_by { |path| File.mtime(path) }
  UI.user_error!("No release IPA found in build/ios/ipa") if ipa.nil?
  ipa
end

platform :ios do
  desc "Validate generated App Store metadata locally without App Store Connect credentials."
  lane :metadata_check do
    repo_sh("make app-store-metadata-check")
    UI.success("Generated App Store metadata is in sync.")
  end

  desc "Validate App Store metadata locally, and upload metadata with deliver when APP_STORE_CONNECT_API_KEY_PATH is set."
  lane :metadata do
    metadata_check
    if ENV["APP_STORE_CONNECT_API_KEY_PATH"].to_s.strip.empty?
      UI.important("APP_STORE_CONNECT_API_KEY_PATH is not set; skipped App Store Connect metadata request.")
    else
      deliver(
        app_identifier: APP_IDENTIFIER,
        api_key_path: required_env("APP_STORE_CONNECT_API_KEY_PATH"),
        metadata_path: APP_STORE_METADATA_PATH,
        skip_binary_upload: true,
        skip_screenshots: true,
        submit_for_review: false,
        force: true
      )
    end
  end

  desc "Build the release IPA and upload it to TestFlight."
  lane :testflight_upload do
    sh("cd \#{Shellwords.escape(APP_ROOT)} && flutter build ipa --release --export-options-plist=\#{Shellwords.escape(IOS_EXPORT_OPTIONS_PATH)}")
    upload_to_testflight(
      app_identifier: APP_IDENTIFIER,
      api_key_path: required_env("APP_STORE_CONNECT_API_KEY_PATH"),
      ipa: latest_ipa_path,
      skip_submission: ENV.fetch("TESTFLIGHT_SKIP_SUBMISSION", "true") != "false"
    )
  end
end
`;
}

function validExportOptions() {
  return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
\t<key>destination</key>
\t<string>export</string>
\t<key>generateAppStoreInformation</key>
\t<false/>
\t<key>manageAppVersionAndBuildNumber</key>
\t<false/>
\t<key>method</key>
\t<string>app-store-connect</string>
\t<key>signingStyle</key>
\t<string>automatic</string>
\t<key>stripSwiftSymbols</key>
\t<true/>
\t<key>teamID</key>
\t<string>5267S9U4PR</string>
\t<key>testFlightInternalTestingOnly</key>
\t<false/>
\t<key>uploadSymbols</key>
\t<true/>
</dict>
</plist>
`;
}
