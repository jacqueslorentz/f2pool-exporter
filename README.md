# F2Pool API Exporter

Export F2Pool API statistics to prometheus

If you have any question do not hesitate to contact me!

See: https://www.f2pool.com/developer/api

## Usage

```sh
git clone https://github.com/jacqueslorentz/f2pool-exporter
cd f2pool-exporter
# Change the --resources arguments as you want
docker compose up
```

Command line options:

- `--resources`: F2Pool API resource(s) separated by a comma (required argument, example: `bitcoin/youraccountname,ethereum/youraddress`)
- `--listen-address`: address an port the listener will use (default: `:5896`)
- `--telemetry-path`: path on which the exporter metrics will be exposed (default: `/metrics`)
