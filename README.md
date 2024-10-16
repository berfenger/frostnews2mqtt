# Frostnews2MQTT - Fronius Inverter Bridge 2 MQTT

### Features
- Monitor power flow with a selectable polling rate.
- Control battery charge:
  - Force battery charge from the grid.
  - Hold energy (don't discharge the battery).
- Implements HomeAssistant discovery for an easy integration with Home Assistant.

### Connection
This software uses Modbus TCP to connect to solar inverters. Tested with Fronius inverters, but should be compatible with any inverter that implements SunSpec through Modbus TCP.

The following Sunspec models are required:
- 1: common
- 101-103: inverter
- 122: status
- 123: controls (optional, needed for feed-in control)
- 124: storage (optional, needed for battery [dis]charge control)
- 160: mppt
- 201-204: ac_meter

### Tested devices
Fronius Primo GEN24 4.0 Plus + BYD Battery-Box Premium HVS + Fronius Smart Meter
Fronius Primo SnapINverter 3.0 + Fronius Smart Meter

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```
