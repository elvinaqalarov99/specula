# specula/laravel

> Laravel middleware for [Specula](https://github.com/elvinaqalarov99/specula) — API docs that can't lie.

Captures real HTTP traffic and ships observations to your local Specula server, which builds a live OpenAPI 3.0 spec automatically. Zero annotations required.

## Requirements

- PHP 8.1+
- Laravel 10 or 11

## Installation

```bash
composer require specula/laravel
```

The `SpeculaServiceProvider` is auto-discovered by Laravel — no manual registration needed.

## Configuration

Publish the config file:

```bash
php artisan vendor:publish --tag=specula-config
```

This creates `config/specula.php`:

```php
return [
    'enabled'        => env('SPECULA_ENABLED', true),
    'endpoint'       => env('SPECULA_ENDPOINT', 'http://localhost:7878'),
    'ignore'         => ['/health', '/metrics', '/telescope'],
    'capture_bodies' => env('SPECULA_CAPTURE_BODIES', true),
];
```

Add to your `.env`:

```env
SPECULA_ENABLED=true
SPECULA_ENDPOINT=http://localhost:7878
```

That's it. Open `http://localhost:7878/docs` to see your API docs populate in real time.

## Options

| Key | Default | Description |
|---|---|---|
| `enabled` | `true` | Toggle observation on/off |
| `endpoint` | `http://localhost:7878` | Specula server URL |
| `ignore` | `['/health', ...]` | Path prefixes to skip |
| `capture_bodies` | `true` | Include request/response bodies |

## How it works

The middleware runs after the response is built, then sends the full request/response pair to the Specula server's `/ingest` endpoint via a non-blocking raw socket. Response time is not affected. If the Specula server is unreachable, the observation is silently dropped.

## Starting the Specula server

```bash
# Download the binary
curl -sSL https://github.com/elvinaqalarov99/specula/releases/latest/download/specula-darwin-arm64 -o specula
chmod +x specula

# Start — proxy on :9999, docs on :7878
./specula start --target http://localhost:3000
```

## License

MIT © [Elvin Agalarov](https://github.com/elvinaqalarov99)
