{ pkgs }:
let
  # Helper functions to derive Go arch from Nix arch
  nixArchToGoArch = arch: {
    "x86_64" = "amd64";
    "aarch64" = "arm64";
  }.${arch} or arch;

  # Split system into arch and os
  parseSystem = system:
    let
      parts = builtins.split "-" system;
      arch = builtins.elemAt parts 0;
      os = builtins.elemAt parts 2;
    in {
      goarch = nixArchToGoArch arch;
      goos = os;
      goURLPath = "${os}-${nixArchToGoArch arch}";
    };

  # Update the following to change the version:
  goVersion = "1.25.11";
  # The sha256 checksums can fetched from https://go.dev/dl/ for new versions.
  #
  # If using a version of nix < 2.25, it will be necessary to manually update the golang flake hash:
  #
  #  HASH=$(nix hash path ./nix/go)
  #  jq --arg hash "$HASH" \
  #    '.nodes["go-flake"].locked += {"lastModified": 1, "narHash": $hash}' \
  #    flake.lock > flake.lock.tmp
  #  mv flake.lock.tmp flake.lock
  #
  goSHA256s = {
    "linux-amd64" = "34f14304e856893f4ba30c2cacfe93906e9de7915c5f6aaaf3a81cdccd7ba30b";
    "linux-arm64" = "c30bf9e156a54ea4e31fbbbf31a712b32734b58cc9a22426fa5ee632d0885124";
    "darwin-amd64" = "26d0ee4071de42b5c332337db9fdd234072877697c547e46e85efb0f59507c66";
    "darwin-arm64" = "cd8d4920e7930d55da1a5a57ba43a64b1305f71cdf2ca3c76cd8c549272b1680";
  };

  targetSystem = parseSystem pkgs.stdenv.hostPlatform.system;
in
pkgs.stdenv.mkDerivation {
  name = "go-${goVersion}";
  version = goVersion;

  inherit (targetSystem) goos goarch;
  GOOS = targetSystem.goos;
  GOARCH = targetSystem.goarch;

  src = pkgs.fetchurl {
    url = "https://go.dev/dl/go${goVersion}.${targetSystem.goURLPath}.tar.gz";
    sha256 = goSHA256s.${targetSystem.goURLPath} or (throw "Unsupported system: ${pkgs.stdenv.hostPlatform.system}");
  };

  # Skip unpacking since we need special handling for the tarball
  dontUnpack = true;

  installPhase = ''
    mkdir -p $out
    # Extract directly to output, stripping the 'go' directory prefix
    # and ignoring permission/ownership issues in containers
    tar xzf $src -C $out --strip-components=1 --no-same-owner --no-same-permissions
    # Ensure go binary is executable
    chmod +x $out/bin/go
  '';
}
