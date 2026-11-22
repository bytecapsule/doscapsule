# Homebrew formula for doscapsule.
#
# We publish TWO zips per release - arm64 (Apple Silicon) and x64 (Intel):
#   doscapsule-arm64-<TAG>.zip
#   doscapsule-x64-<TAG>.zip
#
# The formula picks the right one for the host CPU:
#   - Apple Silicon (M1/M2/...):  arm64
#   - Intel Mac:                  x64
# Both are native; no Rosetta translation needed on either side.
#
# `dosbox-staging` is a HARD dependency: brew installs it automatically when
# absent, matching the host arch (Intel -> x86_64, Apple Silicon -> arm64).
#
# To publish:
#   1. Set this repo up as a Homebrew tap named bytecapsule/homebrew-tap.
#   2. Replace bytecapsule/doscapsule below and <TAG> with a real Release tag.
#   3. Set the two sha256 lines to the zips' actual hashes.
#   4. Users run: brew tap bytecapsule/homebrew-tap && brew install doscapsule

class Doscapsule < Formula
  desc "Pure-local DOS game launcher (SolidJS UI embedded in a Go binary)"
  homepage "https://github.com/bytecapsule/doscapsule"
  license "MIT"

  if Hardware::CPU.arm?
    url "https://github.com/bytecapsule/doscapsule/releases/download/<TAG>/doscapsule-arm64-<TAG>.zip"
    sha256 "<SHA256_ARM64>"
  else
    url "https://github.com/bytecapsule/doscapsule/releases/download/<TAG>/doscapsule-x64-<TAG>.zip"
    sha256 "<SHA256_X64>"
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
