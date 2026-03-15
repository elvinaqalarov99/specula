<?php

namespace ApiDoc\Laravel;

use Illuminate\Support\ServiceProvider;
use Illuminate\Contracts\Http\Kernel;

class ApiDocServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->mergeConfigFrom(__DIR__ . '/../config/apidoc.php', 'apidoc');
    }

    public function boot(Kernel $kernel): void
    {
        $this->publishes([
            __DIR__ . '/../config/apidoc.php' => config_path('apidoc.php'),
        ], 'apidoc-config');

        if (config('apidoc.enabled', true)) {
            $kernel->pushMiddleware(ApiDocMiddleware::class);
        }
    }
}
