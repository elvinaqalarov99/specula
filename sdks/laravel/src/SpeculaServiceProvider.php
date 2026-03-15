<?php

namespace Specula\Laravel;

use Illuminate\Support\ServiceProvider;
use Illuminate\Contracts\Http\Kernel;

class SpeculaServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->mergeConfigFrom(__DIR__ . '/../config/specula.php', 'specula');
    }

    public function boot(Kernel $kernel): void
    {
        $this->publishes([
            __DIR__ . '/../config/specula.php' => config_path('specula.php'),
        ], 'specula-config');

        if (config('specula.enabled', true)) {
            $kernel->pushMiddleware(SpeculaMiddleware::class);
        }
    }
}
