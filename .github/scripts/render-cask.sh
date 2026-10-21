#!/bin/sh
# Gibt die Homebrew-Cask `trading-intelligence-app` auf stdout aus.
# Aufruf: render-cask.sh <version ohne v> <sha256 des ZIP>
set -eu

version="${1:?Version fehlt (z.B. 1.7.0)}"
sha256="${2:?SHA-256 fehlt}"

cat <<CASK
cask "trading-intelligence-app" do
  version "${version}"
  sha256 "${sha256}"

  url "https://github.com/monkey-trader/trading-intelligence-platform/releases/download/v#{version}/trading-intelligence-app_#{version}_darwin_universal.zip"
  name "Trading Intelligence"
  desc "Mac-App für das Trading-Journal mit BingX-Import (Fenster auf trading-intelligence)"
  homepage "https://github.com/monkey-trader/trading-intelligence-platform"

  depends_on formula: "monkey-trader/tap/trading-intelligence"
  depends_on macos: ">= :big_sur"

  app "Trading Intelligence.app"

  # Die App ist nur ad-hoc signiert (keine Apple-Developer-ID). Ohne diesen Schritt blockiert Gatekeeper den Start.
  postflight do
    system_command "/usr/bin/xattr",
                   args: ["-dr", "com.apple.quarantine", "#{appdir}/Trading Intelligence.app"]
  end

  # Nur Fenster-Einstellungen; die Datenbank (journal.db) bleibt bewusst erhalten.
  zap trash: "~/Library/Application Support/tip/window.json"

  caveats <<~EOS
    Die App ist nur ein Fenster auf den Dienst trading-intelligence. Der Dienst muss laufen:
      brew services start trading-intelligence
    BingX-Keys (nur Lese-Rechte) stehen in ~/.config/tip/env, siehe README.
  EOS
end
CASK
