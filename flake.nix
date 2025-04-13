{
  description = "Bobby Assistant Flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    (flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = rec {
          bobby-assistant-service = pkgs.buildGoModule {
            inherit system;
            pname = "bobby-assistant-service";
            version = "main";
            vendorHash = "sha256-1PffwFIOgFRpTxzJBZUGPm/DIczKSVvoAZQOn8twjro=";
            src = ./service;
          };

          default = bobby-assistant-service;
        };

        devShells = rec {
          bobby-assistant-service = with pkgs; mkShell {
            packages = [ go ];
          };

          default = bobby-assistant-service;
        };

        apps = rec {
          bobby-assistant-service = flake-utils.lib.mkApp {
            drv = self.packages.${system}.bobby-assistant-service;
            exePath = "/bin/service";
          };
          default = bobby-assistant-service;
        };

        nixosModules.default = { pkgs, lib, config, ... }:
          with lib;
          let
            cfg = config.services.bobby-assistant-service;
          in
          {
            options.services.bobby-assistant-service = {
              enable = mkOption {
                type = types.bool;
                default = false;
              };

              package = mkOption {
                type = types.package;
                default = self.packages.${pkgs.system}.bobby-assistant-service;
              };

              environmentFile = mkOption {
                type = types.path;
                default = null;
              };
            };

            config = mkIf cfg.enable {

              systemd.services.bobby-assistant-service = {
                description = "bobby-assistant-service";
                wantedBy = [ "multi-user.target" ];
                wants = [ "network-online.target" ];
                after = [ "network-online.target" ];

                serviceConfig = {
                  DynamicUser = true;
                  ExecStart = "${cfg.package}/bin/service";
                  EnvironmentFile = "${cfg.environmentFile}";
                  Restart = "on-failure";
                };

                unitConfig = {
                  StartLimitIntervalSec = 10;
                  StartLimitBurst = 3;
                };
              };
            };
          };
    }));
}

