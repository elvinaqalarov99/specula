<?php

namespace Spectra\Laravel;

use Illuminate\Support\ServiceProvider;
use Illuminate\Contracts\Http\Kernel;

class SpectraServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->mergeConfigFrom(__DIR__ . '/../config/spectra.php', 'spectra');
    }

    public function boot(Kernel $kernel): void
    {
        $this->publishes([
            __DIR__ . '/../config/spectra.php' => config_path('spectra.php'),
        ], 'spectra-config');

        if (config('spectra.enabled', true)) {
            $kernel->pushMiddleware(SpectraMiddleware::class);
        }
    }
}
