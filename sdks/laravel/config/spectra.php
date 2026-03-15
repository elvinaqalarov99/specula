<?php

return [
    /*
    |--------------------------------------------------------------------------
    | Enable Spectra observation
    |--------------------------------------------------------------------------
    | Set to false in production if you only want docs in local/staging.
    */
    'enabled' => env('SPECTRA_ENABLED', true),

    /*
    |--------------------------------------------------------------------------
    | Spectra server endpoint
    |--------------------------------------------------------------------------
    */
    'endpoint' => env('SPECTRA_ENDPOINT', 'http://localhost:7878'),

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
    'capture_bodies' => env('SPECTRA_CAPTURE_BODIES', true),
];
