import { Injectable, NestMiddleware } from '@nestjs/common';
import { Request, Response, NextFunction } from 'express';

export interface SpeculaOptions {
  /** URL of the running Specula server (default: http://localhost:7878) */
  endpoint?: string;
  /** Paths to ignore, e.g. ['/health', '/metrics'] */
  ignore?: string[];
  /** Whether to include request/response bodies (default: true) */
  captureBodies?: boolean;
}

const DEFAULT_OPTIONS: Required<SpeculaOptions> = {
  endpoint: 'http://localhost:7878',
  ignore: ['/health', '/metrics', '/favicon.ico'],
  captureBodies: true,
};

@Injectable()
export class SpeculaMiddleware implements NestMiddleware {
  private options: Required<SpeculaOptions>;
  private ingestUrl: string;

  constructor(options: SpeculaOptions = {}) {
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

      // Skip HTML responses (web pages, not API endpoints)
      const ct = res.getHeader('content-type') ?? '';
      if (String(ct).includes('text/html')) {
        return (originalEnd as any)(chunk, ...args);
      }

      // Fire and forget — never slow down the response
      setImmediate(() => {
        // Use the matched route pattern (e.g. /login/auto/:id/:hash) and convert
        // Express :param syntax to OpenAPI {param} — gives real parameter names.
        const routePath = ((req as any).route?.path ?? req.path)
          .replace(/:([^/]+)/g, '{$1}');

        // Capture Location header for redirect responses
        const responseHeaders: Record<string, string> = {};
        const location = res.getHeader('location');
        if (location) responseHeaders['Location'] = String(location);

        // Only forward the response body if it's JSON — don't send binary file data
        let responseBody: string | undefined;
        if (this.options.captureBodies) {
          const raw = Buffer.concat(chunks).toString('utf8');
          const trimmed = raw.trimStart();
          if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
            responseBody = raw;
          }
        }

        this.sendObservation({
          method: req.method,
          rawPath: routePath,
          queryParams: req.query as Record<string, string>,
          requestBody: this.options.captureBodies ? JSON.stringify(originalBody) : undefined,
          statusCode: res.statusCode,
          responseBody,
          responseHeaders,
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
    responseHeaders?: Record<string, string>;
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
 * import { SpeculaModule } from '@specula/nestjs';
 * app.use(SpeculaModule.middleware({ endpoint: 'http://localhost:7878' }));
 */
export const SpeculaModule = {
  middleware(options?: SpeculaOptions) {
    const mw = new SpeculaMiddleware(options);
    return mw.use.bind(mw);
  },
};
