<?php

return [
    /*
    |--------------------------------------------------------------------------
    | Enable Specula observation
    |--------------------------------------------------------------------------
    | Set to false in production if you only want docs in local/staging.
    */
    'enabled' => env('SPECULA_ENABLED', true),

    /*
    |--------------------------------------------------------------------------
    | Specula server endpoint
    |--------------------------------------------------------------------------
    */
    'endpoint' => env('SPECULA_ENDPOINT', 'http://localhost:7878'),

    /*
    |--------------------------------------------------------------------------
    | Paths to ignore (prefix match)
    |--------------------------------------------------------------------------
    */
    'ignore' => [
        '/health',
        '/metrics',
        '/telescope',
        '/horizon',
        '/_debugbar',
    ],

    /*
    |--------------------------------------------------------------------------
    | Capture request/response bodies
    |--------------------------------------------------------------------------
    */
    'capture_bodies' => env('SPECULA_CAPTURE_BODIES', true),
];
