{
  description = "A Nix-flake-based Go 1.25 development environment";

  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2505";
    nixpkgs-unstable.url = "https://flakehub.com/f/NixOS/nixpkgs/0.1";

    nur = {
      url = "github:nix-community/NUR";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, nur }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forEachSupportedSystem = f:
        nixpkgs.lib.genAttrs supportedSystems (system:
          let
            pkgs = import nixpkgs {
              inherit system;
              overlays = [ nur.overlays.default ];
              config.allowUnfreePredicate = pkg: builtins.elem (nixpkgs.lib.getName pkg) [
                "goreleaser-pro"
              ];
            };
            pkgs-unstable = import nixpkgs-unstable {
              inherit system;
            };
          in
          f { pkgs = pkgs; pkgs-unstable = pkgs-unstable; system = system; }
        );

      speakeasyVersion = "1.690.0";
      speakeasyPlatforms = {
        "x86_64-linux"   = "linux_amd64";
        "aarch64-linux"  = "linux_arm64";
        "x86_64-darwin"  = "darwin_amd64";
        "aarch64-darwin" = "darwin_arm64";
      };
      speakeasyHashes = {
        "x86_64-linux"   = "d0943a498f8df18fe85097f514d3b900beb56738f93e760235f8180def1bbd0a";
        "aarch64-linux"  = "cbdcfc032735c17b71714622743510737811ece0a4cd540ef4a6e6b46ec65075";
        "x86_64-darwin"  = "ee8806f46fdc6655a957f5220f7904b6de69afd1f96f7865de1bfecd92054541";
        "aarch64-darwin" = "33730320d768c89bf631fdc2bc8fd3d2c425c3a1015ae4224495ef91e3f883d7";
      };

    in
    {
      packages = forEachSupportedSystem ({ pkgs, pkgs-unstable, system }:
        {
          speakeasy = pkgs.stdenv.mkDerivation {
            pname = "speakeasy";
            version = speakeasyVersion;

            src = pkgs.fetchurl {
              url = "https://github.com/speakeasy-api/speakeasy/releases/download/v${speakeasyVersion}/speakeasy_${speakeasyPlatforms.${system}}.zip";
              sha256 = speakeasyHashes.${system};
            };

            nativeBuildInputs = [ pkgs.unzip ];
            dontUnpack = true;

            installPhase = ''
              mkdir -p $out/bin
              unzip $src
              ls -al
              install -m755 speakeasy $out/bin/
            '';

            name = "speakeasy";
          };
        }
      );

      defaultPackage.x86_64-linux   = self.packages.x86_64-linux.speakeasy;
      defaultPackage.aarch64-linux  = self.packages.aarch64-linux.speakeasy;
      defaultPackage.x86_64-darwin  = self.packages.x86_64-darwin.speakeasy;
      defaultPackage.aarch64-darwin = self.packages.aarch64-darwin.speakeasy;

      devShells = forEachSupportedSystem ({ pkgs, pkgs-unstable, system }:
        let
          stablePackages = with pkgs; [
            ginkgo
            go_1_25
            go-tools
            goperf
            gotools
            jq
            just
            yq-go
          ];
          unstablePackages = with pkgs-unstable; [
            golangci-lint
            openapi-generator-cli
          ];
          otherPackages = [
            pkgs.nur.repos.goreleaser.goreleaser-pro
            self.packages.${system}.speakeasy
          ];
        in
        {
          default = pkgs.mkShell {
            packages = stablePackages ++ unstablePackages ++ otherPackages;
          };
        }
      );
    };
}
