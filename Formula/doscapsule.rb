# Homebrew formula for doscapsule.
#
# Two zips are published per release - macos-arm64 (Apple Silicon) and macos-x64 (Intel) -
# and the formula picks by host CPU, so both are native (no Rosetta).
#
# `dosbox-staging` is a HARD dependency: brew installs it automatically when
# absent, matching the host architecture.
#
# Users: brew tap bytecapsule/doscapsule && brew install doscapsule
#
# Maintainers: after publishing a release, run
#   scripts/update-packaging.sh <tag>
# to rewrite the tag and both sha256 values below.

class Doscapsule < Formula
  desc "Pure-local DOS game launcher (SolidJS UI embedded in a Go binary)"
  homepage "https://github.com/bytecapsule/doscapsule"
  license "MIT"

  if Hardware::CPU.arm?
    url "https://github.com/bytecapsule/doscapsule/releases/download/20261122-2/doscapsule-arm64-20261122-2.zip"
    sha256 "20f2df1594267458d9b78f6b64511d9c4333ce8cf74849f64d9e4d58e88045bd"
  else
    url "https://github.com/bytecapsule/doscapsule/releases/download/20261122-2/doscapsule-x64-20261122-2.zip"
    sha256 "b2539b5d3825801a2f98ab973ec5983da5bba7cf4185cf4c08b5646fbaa1edea"
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
