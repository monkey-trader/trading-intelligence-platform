#!/bin/sh
# Gibt die Homebrew-Cask `tip-app` auf stdout aus.
# Aufruf: render-cask.sh <version ohne v> <sha256 des ZIP>
set -eu

version="${1:?Version fehlt (z.B. 1.7.0)}"
sha256="${2:?SHA-256 fehlt}"

cat <<CASK
cask "tip-app" do
  version "${version}"
  sha256 "${sha256}"

  url "https://github.com/monkey-trader/trading-intelligence-platform/releases/download/v#{version}/tip-app_#{version}_darwin_universal.zip"
  name "Trading Intelligence Platform"
  desc "Mac-App für das Trading-Journal mit BingX-Import (Fenster auf tip-server)"
  homepage "https://github.com/monkey-trader/trading-intelligence-platform"

  depends_on formula: "monkey-trader/tap/tip-server"
  depends_on macos: ">= :big_sur"

  app "Trading Intelligence Platform.app"

  # Die App ist nur ad-hoc signiert (keine Apple-Developer-ID). Ohne diesen Schritt blockiert Gatekeeper den Start.
  postflight do
    system_command "/usr/bin/xattr",
                   args: ["-dr", "com.apple.quarantine", "#{appdir}/Trading Intelligence Platform.app"]
  end

  # Nur Fenster-Einstellungen; die Datenbank (journal.db) bleibt bewusst erhalten.
  zap trash: "~/Library/Application Support/tip/window.json"

  caveats <<~EOS
    Die App ist nur ein Fenster auf den Dienst tip-server. Der Dienst muss laufen:
      brew services start tip-server
    BingX-Keys (nur Lese-Rechte) stehen in ~/.config/tip/env, siehe README.
  EOS
end
CASK
