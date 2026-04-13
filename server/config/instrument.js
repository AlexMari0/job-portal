// Import with `import * as Sentry from "@sentry/node"` if you are using ESM
import * as Sentry from "@sentry/node";
import { nodeProfilingIntegration } from "@sentry/profiling-node";

Sentry.init({
  dsn: "https://b0b88524c4799cd7fb581761b7d69d86@o4511207013285888.ingest.us.sentry.io/4511207022657536",
  integrations: [
    nodeProfilingIntegration(),
    Sentry.mongooseIntegration()
  ],
// Tracing
// tracesSampleRate: 1.0,
});
// Manually call startProfiler and stopProfiller
// to profile the code in between
Sentry.profiler.startProfiler();

// Starts a transaction that will also be profiled
Sentry.startSpan({
    name: "My First Transaction",
}, () => {

})

Sentry.profiler.stopProfiler()