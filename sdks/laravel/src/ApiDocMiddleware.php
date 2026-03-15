<?php

namespace ApiDoc\Laravel;

use Closure;
use Illuminate\Http\Request;
use Illuminate\Http\Response;

class ApiDocMiddleware
{
    private string $endpoint;
    private array $ignore;
    private bool $captureBodies;

    public function __construct(
        string $endpoint = 'http://localhost:7878',
        array  $ignore = ['/health', '/metrics'],
        bool   $captureBodies = true
    ) {
        $this->endpoint      = rtrim($endpoint, '/');
        $this->ignore        = $ignore;
        $this->captureBodies = $captureBodies;
    }

    public function handle(Request $request, Closure $next): mixed
    {
        // Skip ignored paths
        foreach ($this->ignore as $prefix) {
            if (str_starts_with($request->path(), ltrim($prefix, '/'))) {
                return $next($request);
            }
        }

        $startedAt = microtime(true);
        $response  = $next($request);
        $durationMs = (int) ((microtime(true) - $startedAt) * 1000);

        // Fire and forget via non-blocking socket
        $this->sendObservation([
            'method'       => $request->method(),
            'rawPath'      => '/' . $request->path(),
            'queryParams'  => $request->query(),
            'requestBody'  => $this->captureBodies ? $request->getContent() : null,
            'statusCode'   => $response->getStatusCode(),
            'responseBody' => $this->captureBodies ? $response->getContent() : null,
            'contentType'  => $request->header('Content-Type', ''),
            'durationMs'   => $durationMs,
        ]);

        return $response;
    }

    private function sendObservation(array $obs): void
    {
        $json = json_encode($obs);
        $url  = parse_url($this->endpoint . '/ingest');
        $host = $url['host'] ?? 'localhost';
        $port = $url['port'] ?? 7878;

        // Non-blocking fire-and-forget using a raw socket
        $errno = $errstr = null;
        $sock = @fsockopen($host, $port, $errno, $errstr, 0.05);
        if (!$sock) {
            return;
        }
        stream_set_blocking($sock, false);
        $payload = "POST /ingest HTTP/1.1\r\n"
            . "Host: $host:$port\r\n"
            . "Content-Type: application/json\r\n"
            . "Content-Length: " . strlen($json) . "\r\n"
            . "Connection: close\r\n\r\n"
            . $json;
        @fwrite($sock, $payload);
        @fclose($sock);
    }
}
