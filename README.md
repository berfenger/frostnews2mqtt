# Frostnews2MQTT - Fronius Inverter Bridge 2 MQTT

### Features
- Monitor power flow with customizable polling rate.
  - Inverter
    - Brand, model and firmware version
    - Operating state
    - PV power (DC side)
    - Cabinet temperature
    - AC power flow
    - DC-AC power (normal operation)
    - AC-DC power (while charging the battery from the grid)
    - Track real house power consumption (AC)
  - Battery
    - Operating state
    - State of charge (SoC), charge percentage
    - Max capacity (kWh)
    - Current capacity (kWh)
    - Power flow
    - Charge power (only while charging)
    - Discharge power (only while discharging)
  - Smart Meter
    - Power flow
    - Export power
    - Import power
    - Total energy exported
    - Total energy imported
    - Grid frequency
    - Grid voltage
- Battery (dis)charge control:
  - Force the battery to charge from the grid.
  - Set a target SoC to stop charging from the grid.
  - Hold energy (don't discharge the battery).
- Implements HomeAssistant discovery for an easy integration with Home Assistant.

### Connection
This software uses Modbus TCP to connect to solar inverters. Tested with Fronius inverters, but should be compatible with any inverter that implements SunSpec through Modbus TCP.

The following Sunspec models are required:
- 1: common
- 101-103: inverter
- 122: status
- 123: controls (optional, needed for feed-in control. not yet implemented)
- 124: storage (optional, needed for battery [dis]charge control)
- 160: mppt
- 201-204: ac_meter

### Tested devices
- Fronius Primo GEN24 4.0 Plus + BYD Battery-Box Premium HVS + Fronius Smart Meter
- Fronius Primo SnapINverter 3.0 + Fronius Smart Meter

On Fronius inverters, Modbus TCP must be enabled on the web UI. Check this [guide](./docs/fro_setup.md).

## Deploy using docker

A precompiled docker image is available on [docker hub](https://hub.docker.com/r/acasal/frostnews2mqtt).

A [docker-compose.yml](./docker-compose.yml) and the example config file [config.example.yml](./config.example.yml) are provided as reference.

#### Environment variables
- PORT: HTTP port to deploy a `/healthcheck` endpoint. Default 8080.
- CONFIG_FILE: path of the config file that will be loaded. Optional.

#### Volumes
- `/config.yml`: path of the config file the app will use. Check [config.example.yml](./config.example.yml) for reference.

```bash
$ docker run --name frostnews2mqtt -v ./config.yml:/config.yml acasal/frostnews2mqtt:latest
```

## Licensing
Copyright 2025 Arturo Casal

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.