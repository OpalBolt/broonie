{
  description = "Broonie - self-hosted autonomous issue-to-PR pipeline";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    gomod2nix.url = "github:nix-community/gomod2nix";
  };

  outputs = { self, nixpkgs, gomod2nix }:
    let
      allSystems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ];
      linuxSystems = [ "x86_64-linux" "aarch64-linux" ];
    in
    {
      packages = nixpkgs.lib.genAttrs allSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          broonie = gomod2nix.legacyPackages.${system}.buildGoApplication {
            pname = "broonie";
            version = "0.1.0";
            src = ./.;
            modules = ./gomod2nix.toml;
          };
          attrs = {
            default = broonie;
            broonie = broonie;
          };
        in
        if builtins.elem system linuxSystems then
          attrs // {
            container = pkgs.dockerTools.buildLayeredImage {
              name = "broonie";
              tag = "latest";
              contents = [ broonie pkgs.cacert ];
              config = {
                Cmd = [ "${broonie}/bin/broonie" ];
                Volumes = {
                  "/data" = {};
                  "/sessions" = {};
                  "/workspaces" = {};
                };
              };
            };
          }
        else attrs
      );

      devShells = nixpkgs.lib.genAttrs allSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = [
              pkgs.go
              pkgs.gopls
              pkgs.podman
              gomod2nix.packages.${system}.default
            ];
          };
        }
      );
    };
}
