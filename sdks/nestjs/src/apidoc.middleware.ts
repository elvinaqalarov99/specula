import { Injectable, NestMiddleware } from '@nestjs/common';
import { Request, Response, NextFunction } from 'express';

export interface ApiDocOptions {
  /** URL of the running apidoc server (default: http://localhost:7878) */
  endpoint?: string;
  /** Paths to ignore, e.g. ['/health', '/metrics'] */
  ignore?: string[];
  /** Whether to include request/response bodies (default: true) */
  captureBodies?: boolean;
}

const DEFAULT_OPTIONS: Required<ApiDocOptions> = {
  endpoint: 'http://localhost:7878',
  ignore: ['/health', '/metrics', '/favicon.ico'],
  captureBodies: true,
};

@Injectable()
export class ApiDocMiddleware implements NestMiddleware {
  private options: Required<ApiDocOptions>;
  private ingestUrl: string;

  constructor(options: ApiDocOptions = {}) {
    this.options = { ...DEFAULT_OPTIONS, ...options };
    this.ingestUrl = `${this.options.endpoint}/ingest`;
  }

  use(req: Request, res: Response, next: NextFunction): void {
    // Skip ignored paths
    if (this.options.ignore.some((p) => req.path.startsWith(p))) {
      return next();
    }

    const startedAt = Date.now();
    const chunks: Buffer[] = [];

    // Capture request body
    const originalBody = req.body;

    // Wrap res.write / res.end to capture response body
    const originalWrite = res.write.bind(res);
    const originalEnd = res.end.bind(res);

    res.write = (chunk: any, ...args: any[]) => {
      if (chunk) chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
      return (originalWrite as any)(chunk, ...args);
    };

    res.end = (chunk: any, ...args: any[]) => {
      if (chunk) chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));

      // Fire and forget — never slow down the response
      setImmediate(() => {
        this.sendObservation({
          method: req.method,
          rawPath: req.path,
          queryParams: req.query as Record<string, string>,
          requestBody: this.options.captureBodies ? JSON.stringify(originalBody) : undefined,
          statusCode: res.statusCode,
          responseBody: this.options.captureBodies ? Buffer.concat(chunks).toString('utf8') : undefined,
          contentType: req.headers['content-type'] ?? '',
          durationMs: Date.now() - startedAt,
        });
      });

      return (originalEnd as any)(chunk, ...args);
    };

    next();
  }

  private async sendObservation(obs: {
    method: string;
    rawPath: string;
    queryParams: Record<string, string>;
    requestBody?: string;
    statusCode: number;
    responseBody?: string;
    contentType: string;
    durationMs: number;
  }): Promise<void> {
    try {
      await fetch(this.ingestUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(obs),
        signal: AbortSignal.timeout(2000),
      });
    } catch {
      // Silently drop — observability must never affect production traffic
    }
  }
}

/**
 * Convenience module for use with app.use()
 *
 * @example
 * // main.ts
 * import { ApiDocModule } from '@apidoc/nestjs';
 * app.use(ApiDocModule.middleware({ endpoint: 'http://localhost:7878' }));
 */
export const ApiDocModule = {
  middleware(options?: ApiDocOptions) {
    const mw = new ApiDocMiddleware(options);
    return mw.use.bind(mw);
  },
};
