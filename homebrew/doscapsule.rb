# Homebrew formula for doscapsule.
#
# Two zips are published per release - arm64 (Apple Silicon) and x64 (Intel) -
# and the formula picks by host CPU, so both are native (no Rosetta).
#
# `dosbox-staging` is a HARD dependency: brew installs it automatically when
# absent, matching the host architecture.
#
# Users: brew tap bytecapsule/homebrew-tap && brew install doscapsule
#
# Maintainers: after publishing a release, run
#   scripts/update-packaging.sh <tag>
# to rewrite the tag and both sha256 values below.

class Doscapsule < Formula
  desc "Pure-local DOS game launcher (SolidJS UI embedded in a Go binary)"
  homepage "https://github.com/bytecapsule/doscapsule"
  license "MIT"

  if Hardware::CPU.arm?
    url "https://github.com/bytecapsule/doscapsule/releases/download/20261122-1/doscapsule-arm64-20261122-1.zip"
    sha256 "7301dfbc8796620bedc1c7b8f579a63bcacaa9831f2cc9a0971c631f5212879a"
  else
    url "https://github.com/bytecapsule/doscapsule/releases/download/20261122-1/doscapsule-x64-20261122-1.zip"
    sha256 "6c8479096b8b7243a2c04c48ef780fe51224ddd9a6ca0c323edb84be8b7ee207"
  end

  # Hard dependency: brew installs DOSBox Staging automatically if absent.
  depends_on "dosbox-staging"

  def install
    bin.install "doscapsule"
  end

  test do
    assert_match(/DOS Capsule/, shell_output("#{bin}/doscapsule -version"))
    assert_predicate Formula["dosbox-staging"].opt_bin/"dosbox-staging", :exist?
  end
end
