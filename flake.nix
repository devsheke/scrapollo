{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {inherit system;};
    in
      with pkgs; {
        devShell = mkShell {
          buildInputs = with pkgs; [
            chromium
            golangci-lint
            golangci-lint-langserver
            golines
            gopls
            gotools
          ];
          BROWSER = "${pkgs.chromium}/bin/chromium";
        };
      });
}
