# Homebrew formula for tailbond on macOS (docs/macos.md): the client (hub)
# role, tailbondd + the patched mqvpn, run as a root LaunchDaemon by
# `sudo brew services start tailbond`.
#
# Each GitHub release has this file with its version and checksum filled in
# (packaging/macos/build-bundle.sh). Homebrew installs formulae from taps
# only, so use a local one:
#
#   brew tap-new --no-git local/tailbond
#   curl -fsSLo "$(brew --repository local/tailbond)/Formula/tailbond.rb" \
#     https://github.com/kaovilai/tailbond/releases/latest/download/tailbond.rb
#   brew install local/tailbond/tailbond
#
# This copy in the repository builds from source with --HEAD (Xcode command
# line tools, a few minutes): copy it into the tap and
# `brew install --HEAD local/tailbond/tailbond`.
#
# The binaries are unsigned: fine for software you build or install
# yourself, no Apple Developer ID needed.
class Tailbond < Formula
  desc "Bond Wi-Fi, Ethernet, an iPhone over USB and nearby phones' data in one tunnel"
  homepage "https://github.com/kaovilai/tailbond"
  version "0.0.0"
  url "https://github.com/kaovilai/tailbond/releases/download/v0.0.0/tailbond-macos-arm64.tar.gz"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"
  license "Apache-2.0"

  head do
    url "https://github.com/kaovilai/tailbond.git", branch: "main"
    depends_on "cmake" => :build
    depends_on "go" => :build
    depends_on "gpatch" => :build
    depends_on "libevent" => :build
    depends_on "librsvg" => :build
  end

  depends_on arch: :arm64
  depends_on :macos

  def install
    if build.head?
      ENV["LIBEVENT_PREFIX"] = Formula["libevent"].opt_prefix
      system "scripts/build-mqvpn-macos.sh"
      system "agent/build.sh", "darwin/arm64"
      bin.install "out/macos/mqvpn", "out/agent/darwin-arm64/tailbondd"
      (lib/"tailbond/hooks").install Dir["packaging/macos/hooks/*"]
      system "scripts/brand/render-macos-icons.sh", "out/macos/icons"
      (share/"tailbond/icons").install Dir["out/macos/icons/*"]
    else
      bin.install "bin/mqvpn", "bin/tailbondd"
      (lib/"tailbond/hooks").install Dir["hooks/*"]
      (share/"tailbond/icons").install Dir["icons/*"] if Dir.exist?("icons")
      # the menu bar app (docs/macos.md "Menu bar"); see caveats
      prefix.install "tailbond.app" if Dir.exist?("tailbond.app")
    end
  end

  def post_install
    (var/"lib/tailbond").mkpath
    (var/"log").mkpath
    conf = etc/"tailbond/tailbond.json"
    return if conf.exist?

    (etc/"tailbond").mkpath
    system bin/"tailbondd", "-init", "-role", "client", "-config", conf
    # Homebrew's own locations instead of /usr/local
    inreplace conf do |s|
      s.gsub! "/usr/local/bin/mqvpn", (opt_bin/"mqvpn").to_s
      s.gsub! "/usr/local/var/lib/tailbond", (var/"lib/tailbond").to_s
      s.gsub! "/usr/local/lib/tailbond/hooks", (opt_lib/"tailbond/hooks").to_s
    end
  end

  service do
    run [opt_bin/"tailbondd", "-config", etc/"tailbond/tailbond.json", "-quiet"]
    require_root true
    keep_alive true
    log_path var/"log/tailbond.log"
    error_log_path var/"log/tailbond.log"
  end

  def caveats
    <<~EOS
      Start tailbond (root: it opens a utun tunnel device):
        sudo brew services start tailbond
      Dashboard: http://127.0.0.1:8077 (Setup: paste your home server's pairing link)
      Menu bar app (release installs): open #{opt_prefix}/tailbond.app
        or link it into Applications: ln -sf #{opt_prefix}/tailbond.app /Applications/
    EOS
  end

  test do
    assert_match "tailbondd", shell_output("#{bin}/tailbondd -version")
    assert_match "control-port", shell_output("#{bin}/mqvpn --help 2>&1 || true")
  end
end
