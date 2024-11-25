# fake-log-ingester

A synthetic nginx log generator designed specifically for GreptimeDB, built using the official [greptimedb-ingester-go](https://github.com/GreptimeTeam/greptimedb-ingester-go) SDK. Perfect for testing, demos, and development environments when working with GreptimeDB time-series data.

## Features

- **Native GreptimeDB Integration**
  - Built using the official GreptimeDB Go ingester SDK
  - Efficient batch writing of time-series data
  - Automatic table schema creation
  - Secure connection support with authentication

- **Realistic Log Generation**
  - Configurable HTTP method distribution (GET, POST, PUT, PATCH, DELETE)
  - Valid path generation with customizable depth
  - Realistic status codes with configurable success rates
  - Proper user agents and both IPv4/IPv6 addresses
  - Realistic response body sizes based on status codes

- **Flexible Traffic Patterns**
  - Configurable steady-state request rates
  - Burst mode simulation with customizable multipliers
  - Multiple concurrent tables with independent traffic patterns
  - Adjustable cycle durations for periodic load testing

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | GreptimeDB host address | "" |
| `DATABASE` | Database name | "" |
| `DB_PORT` | Database port | 5001 |
| `DB_USERNAME` | Database username | "" |
| `DB_PASSWORD` | Database password | "" |
| `RATE` | Base requests per second | 2 |
| `TABLE_NUM` | Number of concurrent tables | 10 |
| `BURST_MULTIPLIER` | Traffic multiplication during burst | 10 |
| `BURST_DURATION` | Duration of burst period (seconds) | 30 |
| `CYCLE_DURATION` | Total cycle time (seconds) | 60 |
| ... | (and more) | ... |

## Schema

Generated tables follow this schema:
```sql
CREATE TABLE nginx_logs_N (
    time_local TIMESTAMP,
    ip STRING,
    http_method STRING,
    path STRING,
    http_version STRING,
    status_code INT32,
    body_bytes_sent INT32,
    referrer STRING,
    user_agent STRING
);
```

## Usage
```bash
# Configure GreptimeDB connection
export DB_HOST=localhost
export DATABASE=metrics
export DB_USERNAME=your_username
export DB_PASSWORD=your_password

# Optional: Configure generation parameters
export RATE=10
export TABLE_NUM=5
export STATUS_OK_PERCENT=90
```

## Run the ingester
```bash
./fake-log-ingester
```
