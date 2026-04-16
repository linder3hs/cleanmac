# typed: false
# frozen_string_literal: true

# This formula is auto-updated by GoReleaser on each release.
# Manual installs: see README.md
class Cleanmac < Formula
  desc "Fast interactive terminal disk cleaner for macOS"
  homepage "https://github.com/linder3hs/cleanmac"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/linder3hs/cleanmac/releases/download/v#{version}/cleanmac_#{version}_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_ARM64_SHA256"
    end
    on_intel do
      url "https://github.com/linder3hs/cleanmac/releases/download/v#{version}/cleanmac_#{version}_darwin_amd64.tar.gz"
      sha256 "REPLACE_WITH_AMD64_SHA256"
    end
  end

  def install
    bin.install "cleanmac"
  end

  test do
    assert_match "cleanmac v#{version}", shell_output("#{bin}/cleanmac version")
  end
end
