{
  description = "A Nix-flake-based Go 1.26 development environment";

  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.2511";
    nixpkgs-unstable.url = "https://flakehub.com/f/NixOS/nixpkgs/0.1";

    goreleaser = {
      url = "github:goreleaser/nur/52282e9a2d7c6882e9e0e8675d593b928c89e323";
      flake = false;
    };
  };

  outputs = { self, nixpkgs, nixpkgs-unstable, goreleaser }:
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

    in
    {
      devShells = forEachSupportedSystem ({ pkgs, pkgs-unstable, system }:
        let
          stablePackages = with pkgs; [
            go_1_26
            go-tools
            goperf
            gotools
            jq
            just
            yq-go
          ];
          unstablePackages = with pkgs-unstable; [
            golangci-lint
          ];
          otherPackages = [
            (pkgs.callPackage "${goreleaser}/pkgs/goreleaser-pro" { })
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
