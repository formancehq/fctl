{
  description = "A Nix-flake-based Go 1.27.3 development environment";

  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2511";
    nixpkgs-unstable.url = "https://flakehub.com/f/NixOS/nixpkgs/0.1";

    go-tooling = {
      url = "github:NixOS/nixpkgs/8cec5ef1131078ea76689e84c362b0ed6a1fb6a1";
      flake = false;
    };

    # Published stable Go compiler and matching module builder only.
    nixpkgs-go-stable = {
      url = "https://api.flakehub.com/f/pinned/NixOS/nixpkgs/0.2605.1016949%2Brev-1f607aa79b6356ed45d3cd3c8079c196e79efa2c/01a17424-ae2d-7b20-83c4-cf14527294c3/source.tar.gz";
      flake = false;
    };

    goreleaser = {
      url = "github:goreleaser/nur/52282e9a2d7c6882e9e0e8675d593b928c89e323";
      flake = false;
    };
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, go-tooling, goreleaser, nixpkgs-go-stable }:
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
              overlays = [ ];
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

      speakeasyVersion = "1.759.2";
      speakeasyPlatforms = {
        "x86_64-linux"   = "linux_amd64";
        "aarch64-linux"  = "linux_arm64";
        "x86_64-darwin"  = "darwin_amd64";
        "aarch64-darwin" = "darwin_arm64";
      };
      # Checksums from https://github.com/speakeasy-api/speakeasy/releases/download/v1.759.2/checksums.txt
      speakeasyHashes = {
        "x86_64-linux"   = "9234e2e9138f03ac18f0ca034d0c5a0a7b8749ea91b16814b4a643afd680d8fd";
        "aarch64-linux"  = "ba92a8c5799ed14acba94d317694ed32e35883e9439a07b28c58f7c8c0ea16f5";
        "x86_64-darwin"  = "b4cfe13627e8822718b5820c68898f51b6381e604c9578650c9b0c3f40f800b0";
        "aarch64-darwin" = "dda057dbbd83bdaa47f9ccf3311e455013d957d11f45d8336b97b91ba2a56d6d";
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

            nativeBuildInputs = [ pkgs.unzip pkgs.python3 ]
              ++ pkgs.lib.optionals pkgs.stdenv.hostPlatform.isDarwin [
                pkgs.darwin.autoSignDarwinBinariesHook
              ];
            dontUnpack = true;

            installPhase = ''
              mkdir -p $out/bin
              unzip $src
              # The pinned generator embeds templates/go/config.ts. Update only
              # its Go SDK lint tool: Staticcheck 0.6.1 cannot read Go 1.27 exports.
              # Keep the generator, SDK templates, lint checks and release hashes.
              python3 - <<'PYTHON'
              from pathlib import Path

              binary = Path("speakeasy")
              data = binary.read_bytes()
              old = b'"honnef.co/go/tools/cmd/staticcheck@v0.6.1",\n            "-checks=inherit,-SA1019",'
              new = old.replace(b"@v0.6.1", b"@v0.8.2")
              # Equal length preserves embedded-file offsets on every platform.
              if len(old) != len(new) or data.count(old) != 1:
                  raise SystemExit("unexpected Speakeasy Go SDK lint template")
              binary.write_bytes(data.replace(old, new))
              PYTHON
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
          goPkgs = import nixpkgs-go-stable { inherit system; };
          go = goPkgs.go_1_27;
          buildGo127Module = goPkgs.buildGo127Module;
          # Retain golangci-lint 2.15.0; stable currently provides 2.12.2.
          golangci-lint = goPkgs.callPackage "${go-tooling}/pkgs/by-name/go/golangci-lint/package.nix" {
            inherit buildGo127Module;
          };
          stablePackages = with pkgs; [
            ginkgo
            go-tools
            goperf
            gotools
            jq
            just
            yq-go
          ];
          unstablePackages = with pkgs-unstable; [
            openapi-generator-cli
          ];
          otherPackages = [
            go
            golangci-lint
            (pkgs.callPackage "${goreleaser}/pkgs/goreleaser-pro" { })
            self.packages.${system}.speakeasy
          ];
        in
        {
          default = pkgs.mkShell {
            packages = stablePackages ++ unstablePackages ++ otherPackages;
            GOROOT = "${go}/share/go";
            GOTOOLCHAIN = "local";
          };
        }
      );
    };
}
